"""CRUD operations for richson rs_* tables."""

from __future__ import annotations

import uuid
from datetime import datetime, timezone
from decimal import Decimal
from typing import Any, Sequence

from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from richson.db.models import (
    AnalysisJob,
    AssetAnalysis,
    AssetAnalysisDimension,
    DimensionDefinition,
    EventAlert,
)


# ---------------------------------------------------------------------------
# AssetAnalysis
# ---------------------------------------------------------------------------


async def create_asset_analysis(
    session: AsyncSession,
    data: dict[str, Any],
) -> AssetAnalysis:
    """Insert a new rs_asset_analyses record and return it."""
    analysis = AssetAnalysis(**data)
    session.add(analysis)
    await session.flush()
    await session.refresh(analysis)
    return analysis


async def get_latest_asset_analysis(
    session: AsyncSession,
    asset_code: str,
    locale: str = "zh",
) -> AssetAnalysis | None:
    """Return the most recent non-deleted analysis for the given asset."""
    stmt = (
        select(AssetAnalysis)
        .where(
            AssetAnalysis.asset_code == asset_code,
            AssetAnalysis.locale == locale,
            AssetAnalysis.is_deleted == 0,
        )
        .order_by(AssetAnalysis.analyzed_at.desc())
        .limit(1)
    )
    result = await session.execute(stmt)
    return result.scalar_one_or_none()


async def get_asset_analysis_by_id(
    session: AsyncSession,
    asset_analysis_id: int,
) -> AssetAnalysis | None:
    """Fetch a single analysis record by primary key."""
    stmt = select(AssetAnalysis).where(
        AssetAnalysis.asset_analysis_id == asset_analysis_id,
        AssetAnalysis.is_deleted == 0,
    )
    result = await session.execute(stmt)
    return result.scalar_one_or_none()


async def get_score_history(
    session: AsyncSession,
    asset_code: str,
    days: int = 90,
) -> Sequence[AssetAnalysis]:
    """Return recent analysis records for trend-line rendering."""
    from datetime import timedelta

    cutoff = datetime.now(tz=timezone.utc) - timedelta(days=days)
    stmt = (
        select(AssetAnalysis)
        .where(
            AssetAnalysis.asset_code == asset_code,
            AssetAnalysis.is_deleted == 0,
            AssetAnalysis.analyzed_at >= cutoff,
        )
        .order_by(AssetAnalysis.analyzed_at.asc())
    )
    result = await session.execute(stmt)
    return result.scalars().all()


# ---------------------------------------------------------------------------
# AssetAnalysisDimension
# ---------------------------------------------------------------------------


async def bulk_create_dimensions(
    session: AsyncSession,
    asset_analysis_id: int,
    dimensions: list[dict[str, Any]],
) -> list[AssetAnalysisDimension]:
    """Bulk-insert sub-indicator rows for a given analysis."""
    objs = [
        AssetAnalysisDimension(asset_analysis_id=asset_analysis_id, **d)
        for d in dimensions
    ]
    session.add_all(objs)
    await session.flush()
    return objs


async def get_dimensions_for_analysis(
    session: AsyncSession,
    asset_analysis_id: int,
) -> Sequence[AssetAnalysisDimension]:
    """Return all non-deleted dimension rows for an analysis."""
    stmt = select(AssetAnalysisDimension).where(
        AssetAnalysisDimension.asset_analysis_id == asset_analysis_id,
        AssetAnalysisDimension.is_deleted == 0,
    )
    result = await session.execute(stmt)
    return result.scalars().all()


# ---------------------------------------------------------------------------
# AnalysisJob
# ---------------------------------------------------------------------------


async def create_analysis_job(
    session: AsyncSession,
    asset_code: str,
    locale: str,
    request_id: uuid.UUID | None = None,
    job_type: str = "asset_analysis",
) -> AnalysisJob:
    """Create a new job in pending state, expiring in 1 hour."""
    from datetime import timedelta

    expires_at = datetime.now(tz=timezone.utc) + timedelta(hours=1)
    job = AnalysisJob(
        job_id=uuid.uuid4(),
        asset_code=asset_code,
        locale=locale,
        job_type=job_type,
        status="pending",
        progress=Decimal("0"),
        steps=[],
        expires_at=expires_at,
        request_id=request_id,
    )
    session.add(job)
    await session.flush()
    await session.refresh(job)
    return job


async def get_job(session: AsyncSession, job_id: uuid.UUID) -> AnalysisJob | None:
    """Fetch a job by its UUID primary key."""
    stmt = select(AnalysisJob).where(
        AnalysisJob.job_id == job_id,
        AnalysisJob.is_deleted == 0,
    )
    result = await session.execute(stmt)
    return result.scalar_one_or_none()


async def get_active_job_for_asset(
    session: AsyncSession,
    asset_code: str,
) -> AnalysisJob | None:
    """Return any currently pending/running job for the given asset."""
    stmt = select(AnalysisJob).where(
        AnalysisJob.asset_code == asset_code,
        AnalysisJob.status.in_(["pending", "running"]),
        AnalysisJob.is_deleted == 0,
    )
    result = await session.execute(stmt)
    return result.scalar_one_or_none()


async def update_job_status(
    session: AsyncSession,
    job_id: uuid.UUID,
    status: str,
    current_step: str | None = None,
    progress: float | None = None,
    steps: list[Any] | None = None,
    error_message: str | None = None,
    error_code: str | None = None,
    asset_analysis_id: int | None = None,
) -> None:
    """Partially update job tracking fields."""
    values: dict[str, Any] = {
        "status": status,
        "updated_at": datetime.now(tz=timezone.utc),
    }
    if current_step is not None:
        values["current_step"] = current_step
    if progress is not None:
        values["progress"] = Decimal(str(progress))
    if steps is not None:
        values["steps"] = steps
    if error_message is not None:
        values["error_message"] = error_message
    if error_code is not None:
        values["error_code"] = error_code
    if asset_analysis_id is not None:
        values["asset_analysis_id"] = asset_analysis_id
    if status == "running" and "started_at" not in values:
        values["started_at"] = datetime.now(tz=timezone.utc)
    if status in ("completed", "failed"):
        values["completed_at"] = datetime.now(tz=timezone.utc)

    stmt = (
        update(AnalysisJob)
        .where(AnalysisJob.job_id == job_id)
        .values(**values)
    )
    await session.execute(stmt)


# ---------------------------------------------------------------------------
# EventAlert
# ---------------------------------------------------------------------------


async def create_event_alert(
    session: AsyncSession,
    data: dict[str, Any],
) -> EventAlert:
    """Insert a new event alert record."""
    alert = EventAlert(**data)
    session.add(alert)
    await session.flush()
    await session.refresh(alert)
    return alert


async def get_unalerted_events(
    session: AsyncSession,
) -> Sequence[EventAlert]:
    """Return all pending (not yet alerted) event alerts."""
    stmt = select(EventAlert).where(
        EventAlert.alerted.is_(False),
        EventAlert.is_deleted == 0,
    )
    result = await session.execute(stmt)
    return result.scalars().all()


# ---------------------------------------------------------------------------
# DimensionDefinition
# ---------------------------------------------------------------------------


async def get_dimension_definitions(
    session: AsyncSession,
    asset_type: str,
    model_version: str,
) -> Sequence[DimensionDefinition]:
    """Return active dimension definitions for the given asset type and model version."""
    stmt = (
        select(DimensionDefinition)
        .where(
            DimensionDefinition.asset_type == asset_type,
            DimensionDefinition.model_version == model_version,
            DimensionDefinition.is_deleted == 0,
        )
        .order_by(DimensionDefinition.display_order.asc())
    )
    result = await session.execute(stmt)
    return result.scalars().all()
