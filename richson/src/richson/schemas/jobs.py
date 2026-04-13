"""Schemas for async job endpoints (POST /jobs/analyze-asset, GET /jobs/{jobId})."""

from __future__ import annotations

import uuid
from datetime import datetime
from typing import Any, Literal

from pydantic import BaseModel, Field

from richson.schemas.common import LLMConfig

# ---------------------------------------------------------------------------
# Request schemas
# ---------------------------------------------------------------------------


class AnalyzeAssetRequest(BaseModel):
    """POST /jobs/analyze-asset request body."""

    asset_code: str = Field(alias="assetCode")
    locale: str = Field(default="zh", pattern="^(zh|en)$")
    llm_config: LLMConfig = Field(alias="llmConfig")
    request_id: uuid.UUID | None = Field(default=None, alias="requestId")

    model_config = {"populate_by_name": True}


class BatchAnalyzeAsset(BaseModel):
    asset_code: str = Field(alias="assetCode")
    locale: str = Field(default="zh")

    model_config = {"populate_by_name": True}


class BatchAnalyzeRequest(BaseModel):
    """POST /jobs/batch-analyze request body."""

    assets: list[BatchAnalyzeAsset]
    llm_config: LLMConfig = Field(alias="llmConfig")
    request_id: uuid.UUID | None = Field(default=None, alias="requestId")

    model_config = {"populate_by_name": True}


# ---------------------------------------------------------------------------
# Response schemas
# ---------------------------------------------------------------------------

JobStatus = Literal["pending", "running", "completed", "failed"]
StepStatus = Literal["pending", "running", "completed", "failed", "skipped"]


class StepInfo(BaseModel):
    name: str
    status: StepStatus
    duration_ms: int | None = Field(default=None, alias="durationMs")

    model_config = {"populate_by_name": True}


class JobSummary(BaseModel):
    """Minimal job info returned from create/batch endpoints."""

    job_id: uuid.UUID = Field(alias="jobId")
    status: JobStatus
    asset_code: str = Field(alias="assetCode")
    created_at: datetime = Field(alias="createdAt")

    model_config = {"populate_by_name": True}


class JobDetail(BaseModel):
    """Full job detail returned from GET /jobs/{jobId}."""

    job_id: uuid.UUID = Field(alias="jobId")
    asset_code: str = Field(alias="assetCode")
    status: JobStatus
    current_step: str | None = Field(default=None, alias="currentStep")
    progress: float
    steps: list[StepInfo]
    error: str | None = None
    created_at: datetime = Field(alias="createdAt")
    started_at: datetime | None = Field(default=None, alias="startedAt")
    completed_at: datetime | None = Field(default=None, alias="completedAt")

    model_config = {"populate_by_name": True}


class BatchJobSkipped(BaseModel):
    asset_code: str = Field(alias="assetCode")
    reason: str

    model_config = {"populate_by_name": True}


class BatchAnalyzeResponse(BaseModel):
    jobs: list[JobSummary]
    skipped: list[BatchJobSkipped] = Field(default_factory=list)


class AnalyzeAssetResponse(BaseModel):
    job_id: uuid.UUID = Field(alias="jobId")
    status: JobStatus
    asset_code: str = Field(alias="assetCode")
    created_at: datetime = Field(alias="createdAt")

    model_config = {"populate_by_name": True}
