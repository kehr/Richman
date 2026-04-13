"""SQLAlchemy 2.0 declarative models for rs_* tables."""

from __future__ import annotations

import uuid
from datetime import date, datetime
from decimal import Decimal
from typing import Any

from sqlalchemy import (
    BigInteger,
    Boolean,
    Date,
    DateTime,
    ForeignKey,
    Numeric,
    SmallInteger,
    String,
    Text,
    func,
)
from sqlalchemy.dialects.postgresql import JSONB, UUID
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship


class Base(DeclarativeBase):
    """Base class shared by all richson ORM models."""

    pass


class AssetAnalysis(Base):
    """rs_asset_analyses: stores per-asset analysis results."""

    __tablename__ = "rs_asset_analyses"

    asset_analysis_id: Mapped[int] = mapped_column(BigInteger, primary_key=True)
    asset_code: Mapped[str] = mapped_column(String(32), nullable=False)
    locale: Mapped[str] = mapped_column(String(8), nullable=False, default="zh")

    # Overall scoring
    overall_score: Mapped[Decimal] = mapped_column(Numeric(5, 2), nullable=False)
    signal_level: Mapped[str] = mapped_column(String(32), nullable=False)
    confidence: Mapped[Decimal] = mapped_column(Numeric(5, 2), nullable=False)
    confidence_band_low: Mapped[Decimal] = mapped_column(Numeric(5, 2), nullable=False)
    confidence_band_high: Mapped[Decimal] = mapped_column(Numeric(5, 2), nullable=False)
    model_version: Mapped[str] = mapped_column(String(32), nullable=False)

    # Layer 3 text output
    market_interpretation: Mapped[str] = mapped_column(Text, nullable=False, default="")
    risk_factors: Mapped[list[Any]] = mapped_column(JSONB, nullable=False, default=list)
    regime_summary: Mapped[str] = mapped_column(Text, nullable=False, default="")

    # Dimension scores (denormalized for query performance)
    d1_score: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    d1_base_score: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    d1_llm_adjustment: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), default=Decimal("0"))
    d2_score: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    d2_base_score: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    d2_llm_adjustment: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), default=Decimal("0"))
    d3_score: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    d3_base_score: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    d3_llm_adjustment: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), default=Decimal("0"))
    d4_score: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    d4_base_score: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)

    # Weight snapshot at analysis time
    d1_weight: Mapped[Decimal] = mapped_column(Numeric(4, 2), nullable=False, default=Decimal("0.30"))
    d2_weight: Mapped[Decimal] = mapped_column(Numeric(4, 2), nullable=False, default=Decimal("0.25"))
    d3_weight: Mapped[Decimal] = mapped_column(Numeric(4, 2), nullable=False, default=Decimal("0.25"))
    d4_weight: Mapped[Decimal] = mapped_column(Numeric(4, 2), nullable=False, default=Decimal("0.20"))

    # Degradation markers
    llm_skipped: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)
    data_coverage: Mapped[str] = mapped_column(String(16), nullable=False, default="full")

    # Conflict detection
    conflict_type: Mapped[str | None] = mapped_column(String(16), nullable=True)
    conflict_message: Mapped[str | None] = mapped_column(Text, nullable=True)

    # Change tracking (vs previous analysis)
    prev_analysis_id: Mapped[int | None] = mapped_column(BigInteger, nullable=True)
    score_delta: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    change_summary: Mapped[str | None] = mapped_column(Text, nullable=True)
    major_change_recap: Mapped[str | None] = mapped_column(Text, nullable=True)

    # Currency conversion (for CNY assets)
    usd_exchange_rate: Mapped[Decimal | None] = mapped_column(Numeric(12, 6), nullable=True)

    # Data freshness
    data_snapshot_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    price_at_analysis: Mapped[Decimal | None] = mapped_column(Numeric(20, 6), nullable=True)

    # Pre-computed demo plan
    demo_plan: Mapped[dict[str, Any] | None] = mapped_column(JSONB, nullable=True)

    # Extensible metadata (drawdown_reference, etc.)
    analysis_metadata: Mapped[dict[str, Any]] = mapped_column(
        JSONB, nullable=False, default=dict
    )

    # Generation mode: full | l1_only | backfill
    generated_by: Mapped[str] = mapped_column(String(16), nullable=False, default="full")

    # Audit fields
    source: Mapped[str] = mapped_column(String(16), nullable=False, default="scheduled")
    job_id: Mapped[uuid.UUID | None] = mapped_column(UUID(as_uuid=True), nullable=True)
    analyzed_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    creator: Mapped[str] = mapped_column(String(64), nullable=False, default="richson")
    modifier: Mapped[str] = mapped_column(String(64), nullable=False, default="richson")
    is_deleted: Mapped[int] = mapped_column(SmallInteger, nullable=False, default=0)

    # Relationships
    dimensions: Mapped[list[AssetAnalysisDimension]] = relationship(
        "AssetAnalysisDimension",
        back_populates="analysis",
        primaryjoin="and_(AssetAnalysis.asset_analysis_id == AssetAnalysisDimension.asset_analysis_id, AssetAnalysisDimension.is_deleted == 0)",
        lazy="select",
    )


class AssetAnalysisDimension(Base):
    """rs_asset_analysis_dimensions: sub-indicator details per dimension."""

    __tablename__ = "rs_asset_analysis_dimensions"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True)
    asset_analysis_id: Mapped[int] = mapped_column(
        BigInteger,
        ForeignKey("rs_asset_analyses.asset_analysis_id"),
        nullable=False,
    )
    dimension: Mapped[str] = mapped_column(String(8), nullable=False)
    sub_indicator: Mapped[str] = mapped_column(String(64), nullable=False)
    raw_value: Mapped[Decimal | None] = mapped_column(Numeric(20, 6), nullable=True)
    percentile_1y: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    percentile_5y: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    blended_percentile: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    normalized_score: Mapped[Decimal | None] = mapped_column(Numeric(5, 2), nullable=True)
    weight_in_dimension: Mapped[Decimal | None] = mapped_column(Numeric(4, 2), nullable=True)
    data_source: Mapped[str | None] = mapped_column(String(32), nullable=True)
    data_as_of: Mapped[date | None] = mapped_column(Date, nullable=True)

    # Audit fields
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    creator: Mapped[str] = mapped_column(String(64), nullable=False, default="richson")
    modifier: Mapped[str] = mapped_column(String(64), nullable=False, default="richson")
    is_deleted: Mapped[int] = mapped_column(SmallInteger, nullable=False, default=0)

    # Relationships
    analysis: Mapped[AssetAnalysis] = relationship(
        "AssetAnalysis", back_populates="dimensions"
    )


class AnalysisJob(Base):
    """rs_analysis_jobs: async job tracking for asset-level analyses."""

    __tablename__ = "rs_analysis_jobs"

    job_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True),
        primary_key=True,
        default=uuid.uuid4,
    )
    asset_code: Mapped[str] = mapped_column(String(32), nullable=False)
    job_type: Mapped[str] = mapped_column(String(32), nullable=False, default="asset_analysis")
    status: Mapped[str] = mapped_column(String(16), nullable=False, default="pending")
    progress: Mapped[Decimal] = mapped_column(Numeric(4, 2), nullable=False, default=Decimal("0"))
    current_step: Mapped[str | None] = mapped_column(String(64), nullable=True)
    steps: Mapped[list[Any]] = mapped_column(JSONB, nullable=False, default=list)
    error_message: Mapped[str | None] = mapped_column(Text, nullable=True)
    error_code: Mapped[str | None] = mapped_column(String(64), nullable=True)

    # Result reference
    asset_analysis_id: Mapped[int | None] = mapped_column(BigInteger, nullable=True)

    # Timing
    expires_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        nullable=False,
        server_default=func.now(),  # application sets correct value on insert
    )
    started_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    completed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)

    # Metadata
    request_id: Mapped[uuid.UUID | None] = mapped_column(UUID(as_uuid=True), nullable=True)
    locale: Mapped[str | None] = mapped_column(String(8), nullable=True)

    # Audit fields
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    creator: Mapped[str] = mapped_column(String(64), nullable=False, default="richson")
    modifier: Mapped[str] = mapped_column(String(64), nullable=False, default="richson")
    is_deleted: Mapped[int] = mapped_column(SmallInteger, nullable=False, default=0)


class EventAlert(Base):
    """rs_event_alerts: Polymarket event probability delta monitoring."""

    __tablename__ = "rs_event_alerts"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True)
    event_slug: Mapped[str] = mapped_column(String(128), nullable=False)
    event_title: Mapped[str] = mapped_column(Text, nullable=False)
    source: Mapped[str] = mapped_column(String(32), nullable=False, default="polymarket")
    prev_probability: Mapped[Decimal] = mapped_column(Numeric(5, 4), nullable=False)
    curr_probability: Mapped[Decimal] = mapped_column(Numeric(5, 4), nullable=False)
    delta: Mapped[Decimal] = mapped_column(Numeric(5, 4), nullable=False)
    threshold: Mapped[Decimal] = mapped_column(Numeric(5, 4), nullable=False, default=Decimal("0.20"))
    gold_direction: Mapped[str | None] = mapped_column(String(16), nullable=True)
    alerted: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)
    detected_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )

    # Audit fields
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    creator: Mapped[str] = mapped_column(String(64), nullable=False, default="richson")
    modifier: Mapped[str] = mapped_column(String(64), nullable=False, default="richson")
    is_deleted: Mapped[int] = mapped_column(SmallInteger, nullable=False, default=0)


class DimensionDefinition(Base):
    """rs_dimension_definitions: dimension weight configuration per asset type."""

    __tablename__ = "rs_dimension_definitions"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True)
    asset_type: Mapped[str] = mapped_column(String(32), nullable=False)
    dimension: Mapped[str] = mapped_column(String(8), nullable=False)
    name_zh: Mapped[str] = mapped_column(String(32), nullable=False)
    name_en: Mapped[str] = mapped_column(String(32), nullable=False)
    weight: Mapped[Decimal] = mapped_column(Numeric(4, 2), nullable=False)
    description_zh: Mapped[str | None] = mapped_column(Text, nullable=True)
    description_en: Mapped[str | None] = mapped_column(Text, nullable=True)
    display_order: Mapped[int] = mapped_column(SmallInteger, nullable=False, default=0)
    model_version: Mapped[str] = mapped_column(String(32), nullable=False)

    # Audit fields
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    creator: Mapped[str] = mapped_column(String(64), nullable=False, default="system")
    modifier: Mapped[str] = mapped_column(String(64), nullable=False, default="system")
    is_deleted: Mapped[int] = mapped_column(SmallInteger, nullable=False, default=0)
