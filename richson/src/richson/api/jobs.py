"""Async job endpoints for asset analysis (Mode A).

POST /jobs/analyze-asset  - trigger single asset analysis
POST /jobs/batch-analyze  - trigger batch asset analysis
GET  /jobs/{jobId}        - poll job status
"""

from __future__ import annotations

import asyncio
import uuid

import structlog
from fastapi import APIRouter, Depends, HTTPException, Request
from sqlalchemy.exc import IntegrityError

from richson.api.auth import require_api_key
from richson.config import settings
from richson.db import repository as repo
from richson.schemas.jobs import (
    AnalyzeAssetRequest,
    BatchAnalyzeRequest,
    JobSummary,
)

router = APIRouter(prefix="/jobs", dependencies=[Depends(require_api_key)])
logger = structlog.get_logger()


def _build_job_summary(job: object) -> JobSummary:
    return JobSummary(
        jobId=job.job_id,  # type: ignore[attr-defined]
        status=job.status,  # type: ignore[attr-defined]
        assetCode=job.asset_code,  # type: ignore[attr-defined]
        createdAt=job.created_at,  # type: ignore[attr-defined]
    )


@router.post("/analyze-asset", status_code=202)
async def analyze_asset(
    body: AnalyzeAssetRequest,
    request: Request,
) -> dict:
    """Trigger single-asset analysis job.

    Returns 202 with job info, 409 if a job is already running for this asset.
    """
    from richson.main import get_session_factory  # noqa: PLC0415

    session_factory = get_session_factory()

    async with session_factory() as sess:
        existing = await repo.get_active_job_for_asset(sess, body.asset_code)
        if existing is not None:
            raise HTTPException(
                status_code=409,
                detail={
                    "error": {
                        "code": "ANALYSIS_IN_PROGRESS",
                        "message": f"An analysis job is already running for {body.asset_code}",
                        "details": [],
                    }
                },
            )

        job = await repo.create_analysis_job(
            sess,
            asset_code=body.asset_code,
            locale=body.locale,
            request_id=body.request_id,
        )
        await sess.commit()
        await sess.refresh(job)

    log = logger.bind(job_id=str(job.job_id), asset_code=body.asset_code)
    log.info("job_created")

    # Check LLM budget
    budget_exceeded = False
    if hasattr(settings, "daily_llm_budget_usd") and settings.daily_llm_budget_usd <= 0:
        budget_exceeded = True

    # Fire and forget background task
    from richson.core.pipeline import run_asset_analysis_pipeline  # noqa: PLC0415

    asyncio.create_task(
        run_asset_analysis_pipeline(
            job_id=job.job_id,
            asset_code=body.asset_code,
            locale=body.locale,
            llm_config=body.llm_config,
            session_factory=session_factory,
            request_id=body.request_id,
            budget_exceeded=budget_exceeded,
        )
    )

    return {
        "data": {
            "jobId": str(job.job_id),
            "status": job.status,
            "assetCode": job.asset_code,
            "createdAt": job.created_at.isoformat(),
        }
    }


@router.post("/batch-analyze", status_code=202)
async def batch_analyze(
    body: BatchAnalyzeRequest,
    request: Request,
) -> dict:
    """Trigger batch asset analysis.

    Returns 202 with list of created jobs and list of skipped assets (already running).
    """
    from richson.main import get_session_factory  # noqa: PLC0415

    session_factory = get_session_factory()

    jobs_created: list[dict] = []
    skipped: list[dict] = []

    budget_exceeded = False
    if hasattr(settings, "daily_llm_budget_usd") and settings.daily_llm_budget_usd <= 0:
        budget_exceeded = True

    from richson.core.pipeline import run_asset_analysis_pipeline  # noqa: PLC0415

    for asset_item in body.assets:
        async with session_factory() as sess:
            existing = await repo.get_active_job_for_asset(sess, asset_item.asset_code)
            if existing is not None:
                skipped.append({
                    "assetCode": asset_item.asset_code,
                    "reason": "ANALYSIS_IN_PROGRESS",
                })
                continue

            try:
                job = await repo.create_analysis_job(
                    sess,
                    asset_code=asset_item.asset_code,
                    locale=asset_item.locale,
                    request_id=body.request_id,
                )
                await sess.commit()
                await sess.refresh(job)
            except IntegrityError:
                await sess.rollback()
                skipped.append({
                    "assetCode": asset_item.asset_code,
                    "reason": "ANALYSIS_IN_PROGRESS",
                })
                continue

        logger.info("batch_job_created", job_id=str(job.job_id), asset_code=asset_item.asset_code)
        jobs_created.append({
            "jobId": str(job.job_id),
            "assetCode": job.asset_code,
            "status": job.status,
        })

        asyncio.create_task(
            run_asset_analysis_pipeline(
                job_id=job.job_id,
                asset_code=asset_item.asset_code,
                locale=asset_item.locale,
                llm_config=body.llm_config,
                session_factory=session_factory,
                request_id=body.request_id,
                budget_exceeded=budget_exceeded,
            )
        )

    return {
        "data": {
            "jobs": jobs_created,
            "skipped": skipped,
        }
    }


@router.get("/{job_id}")
async def get_job_status(job_id: uuid.UUID) -> dict:
    """Poll job status and progress."""
    from richson.main import get_session_factory  # noqa: PLC0415

    session_factory = get_session_factory()

    async with session_factory() as sess:
        job = await repo.get_job(sess, job_id)

    if job is None:
        raise HTTPException(status_code=404, detail={
            "error": {
                "code": "JOB_NOT_FOUND",
                "message": f"Job {job_id} not found",
                "details": [],
            }
        })

    steps = []
    for step in (job.steps or []):
        steps.append({
            "name": step.get("name"),
            "status": step.get("status"),
            "durationMs": step.get("durationMs"),
        })

    return {
        "data": {
            "jobId": str(job.job_id),
            "assetCode": job.asset_code,
            "status": job.status,
            "currentStep": job.current_step,
            "progress": float(job.progress),
            "steps": steps,
            "error": job.error_message,
            "createdAt": job.created_at.isoformat(),
            "startedAt": job.started_at.isoformat() if job.started_at else None,
            "completedAt": job.completed_at.isoformat() if job.completed_at else None,
        }
    }
