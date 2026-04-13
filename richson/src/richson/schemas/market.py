"""Schemas for market data endpoints (regime, OHLCV, score history)."""

from __future__ import annotations

from datetime import datetime
from typing import Literal

from pydantic import BaseModel, Field

MarketRegime = Literal["risk_on", "neutral", "risk_off"]


class IndexSnapshot(BaseModel):
    name: str
    code: str
    price: float
    change_percent: float = Field(alias="changePercent")

    model_config = {"populate_by_name": True}


class MarketRegimeData(BaseModel):
    regime: MarketRegime
    regime_label: str = Field(alias="regimeLabel")
    reason: str
    vix: float
    t10y2y: float
    credit_spread: float = Field(alias="creditSpread")
    indices: list[IndexSnapshot] = Field(default_factory=list)
    updated_at: datetime = Field(alias="updatedAt")

    model_config = {"populate_by_name": True}


class OHLCVCandle(BaseModel):
    date: str
    open: float
    high: float
    low: float
    close: float
    volume: float


class OHLCVData(BaseModel):
    asset_code: str = Field(alias="assetCode")
    currency: str
    period: str
    candles: list[OHLCVCandle]
    sma200: float | None = None
    support_levels: list[float] = Field(default_factory=list, alias="supportLevels")
    resistance_levels: list[float] = Field(default_factory=list, alias="resistanceLevels")

    model_config = {"populate_by_name": True}


class ScorePoint(BaseModel):
    date: str
    overall_score: float = Field(alias="overallScore")
    d1_score: float | None = Field(default=None, alias="d1Score")
    d2_score: float | None = Field(default=None, alias="d2Score")
    d3_score: float | None = Field(default=None, alias="d3Score")
    d4_score: float | None = Field(default=None, alias="d4Score")
    model_version: str = Field(alias="modelVersion")

    model_config = {"populate_by_name": True}


class VersionChange(BaseModel):
    date: str
    from_version: str = Field(alias="fromVersion")
    to_version: str = Field(alias="toVersion")
    note: str

    model_config = {"populate_by_name": True}


class ScoreHistoryData(BaseModel):
    asset_code: str = Field(alias="assetCode")
    scores: list[ScorePoint]
    version_changes: list[VersionChange] = Field(
        default_factory=list, alias="versionChanges"
    )

    model_config = {"populate_by_name": True}
