"""Schemas for synchronous analysis endpoints (holding analysis, demo plan)."""

from __future__ import annotations

import uuid
from datetime import datetime
from decimal import Decimal
from typing import Any, Literal

from pydantic import BaseModel, Field

from richson.schemas.common import LLMConfig

# ---------------------------------------------------------------------------
# Shared sub-schemas
# ---------------------------------------------------------------------------

RiskPreference = Literal["conservative", "moderate", "aggressive"]
SignalLevel = Literal[
    "strong_bullish", "moderate_bullish", "neutral", "moderate_bearish", "strong_bearish"
]
ConcentrationLevel = Literal["green", "yellow", "blue", "red"]


class HoldingInput(BaseModel):
    holding_id: int = Field(alias="holdingId")
    cost_price: Decimal = Field(alias="costPrice")
    position_ratio: float = Field(alias="positionRatio", description="Percent 0-100")
    quantity: float

    model_config = {"populate_by_name": True}


class Scenario(BaseModel):
    condition: str
    action: str
    lot_count: int = Field(alias="lotCount")
    rationale: str
    priority: int = Field(ge=1, description="Lower = higher priority; stop-loss is always 1")
    exclusion_group: str | None = Field(default=None, alias="exclusionGroup")

    model_config = {"populate_by_name": True}


class ExecutionPlanData(BaseModel):
    action: str
    action_label: str = Field(alias="actionLabel")
    default_action: str = Field(alias="defaultAction")
    current_position: float = Field(alias="currentPosition")
    target_position: float = Field(alias="targetPosition")
    scenarios: list[Scenario]
    stop_loss: float = Field(alias="stopLoss")
    take_profit: float = Field(alias="takeProfit")
    # Bound mirrors the TRD contract (richson SS21.8): execution plans expire
    # between 1 and 90 days, so accept only values in that range. The default
    # continues to be 7 days.
    valid_days: int = Field(default=7, ge=1, le=90, alias="validDays")
    no_trigger_note: str = Field(alias="noTriggerNote")
    concentration_level: ConcentrationLevel | None = Field(
        default=None, alias="concentrationLevel"
    )
    concentration_message: str | None = Field(default=None, alias="concentrationMessage")
    is_demo_plan: bool = Field(default=False, alias="isDemoPlan")

    model_config = {"populate_by_name": True}


# ---------------------------------------------------------------------------
# Holding analysis (Mode B)
# ---------------------------------------------------------------------------


class AnalyzeHoldingRequest(BaseModel):
    """POST /analyze/holding request body."""

    asset_code: str = Field(alias="assetCode")
    asset_analysis_id: int = Field(alias="assetAnalysisId")
    holding: HoldingInput
    risk_preference: RiskPreference = Field(alias="riskPreference")
    peer_exposure: float = Field(alias="peerExposure", description="Percent 0-100")
    language: str = Field(default="zh", pattern="^(zh|en)$")
    llm_config: LLMConfig | None = Field(default=None, alias="llmConfig")
    request_id: uuid.UUID | None = Field(default=None, alias="requestId")

    model_config = {"populate_by_name": True}


# ---------------------------------------------------------------------------
# Demo plan (Mode C)
# ---------------------------------------------------------------------------


class DemoPlanRequest(BaseModel):
    """POST /analyze/demo-plan request body."""

    asset_code: str = Field(alias="assetCode")
    language: str = Field(default="zh", pattern="^(zh|en)$")
    llm_config: LLMConfig | None = Field(default=None, alias="llmConfig")
    request_id: uuid.UUID | None = Field(default=None, alias="requestId")

    model_config = {"populate_by_name": True}


# ---------------------------------------------------------------------------
# Sub-indicator detail (for dimension panels)
# ---------------------------------------------------------------------------


class SubIndicatorDetail(BaseModel):
    name: str
    raw_value: Decimal | None = Field(default=None, alias="rawValue")
    percentile_1y: float | None = Field(default=None, alias="percentile1y")
    percentile_5y: float | None = Field(default=None, alias="percentile5y")
    blended_percentile: float | None = Field(default=None, alias="blendedPercentile")
    normalized_score: float | None = Field(default=None, alias="normalizedScore")
    weight_in_dimension: float | None = Field(default=None, alias="weightInDimension")
    data_source: str | None = Field(default=None, alias="dataSource")
    data_as_of: str | None = Field(default=None, alias="dataAsOf")

    model_config = {"populate_by_name": True}


class DimensionDetail(BaseModel):
    dimension: str  # D1 | D2 | D3 | D4
    name_zh: str = Field(alias="nameZh")
    name_en: str = Field(alias="nameEn")
    score: float | None = None
    base_score: float | None = Field(default=None, alias="baseScore")
    llm_adjustment: float | None = Field(default=None, alias="llmAdjustment")
    llm_anomaly_flag: bool = Field(default=False, alias="llmAnomalyFlag")
    weight: float
    sub_indicators: list[SubIndicatorDetail] = Field(
        default_factory=list, alias="subIndicators"
    )

    model_config = {"populate_by_name": True}


class DrawdownReference(BaseModel):
    current_bull_run_start: str = Field(alias="currentBullRunStart")
    max_drawdown: float = Field(alias="maxDrawdown")
    max_drawdown_date: str = Field(alias="maxDrawdownDate")
    historical_avg_drawdown: float = Field(alias="historicalAvgDrawdown")

    model_config = {"populate_by_name": True}


class AnalysisDetail(BaseModel):
    """Full analysis object returned in GET /api/v2/market/{code}."""

    asset_analysis_id: int = Field(alias="assetAnalysisId")
    overall_score: float = Field(alias="overallScore")
    signal_level: SignalLevel = Field(alias="signalLevel")
    confidence: float
    confidence_band_low: float = Field(alias="confidenceBandLow")
    confidence_band_high: float = Field(alias="confidenceBandHigh")
    model_version: str = Field(alias="modelVersion")
    market_interpretation: str = Field(alias="marketInterpretation")
    risk_factors: list[str] = Field(alias="riskFactors", default_factory=list)
    regime_summary: str = Field(alias="regimeSummary")
    conflict_type: str | None = Field(default=None, alias="conflictType")
    conflict_message: str | None = Field(default=None, alias="conflictMessage")
    score_delta: float | None = Field(default=None, alias="scoreDelta")
    change_summary: str | None = Field(default=None, alias="changeSummary")
    major_change_recap: str | None = Field(default=None, alias="majorChangeRecap")
    usd_exchange_rate: Decimal | None = Field(default=None, alias="usdExchangeRate")
    price_at_analysis: Decimal | None = Field(default=None, alias="priceAtAnalysis")
    analyzed_at: datetime = Field(alias="analyzedAt")
    generated_by: str = Field(alias="generatedBy")
    llm_skipped: bool = Field(alias="llmSkipped")
    drawdown_reference: DrawdownReference | None = Field(
        default=None, alias="drawdownReference"
    )
    demo_plan: dict[str, Any] | None = Field(default=None, alias="demoPlan")
    dimensions: list[DimensionDetail] = Field(default_factory=list)

    model_config = {"populate_by_name": True}
