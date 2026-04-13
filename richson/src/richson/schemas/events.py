"""Schemas for event radar endpoints (GET /events/radar)."""

from __future__ import annotations

from datetime import datetime
from typing import Literal

from pydantic import BaseModel, Field

EventImpact = Literal["high", "medium", "low"]
GoldDirection = Literal["bullish", "bearish", "neutral"]


class EventItem(BaseModel):
    date: str
    title: str
    category: str
    impact: EventImpact
    gold_direction: GoldDirection | None = Field(default=None, alias="goldDirection")
    probability: float | None = None
    probability_source: str | None = Field(default=None, alias="probabilitySource")
    probability_change_24h: float | None = Field(
        default=None, alias="probabilityChange24h"
    )

    model_config = {"populate_by_name": True}


class EventRadarData(BaseModel):
    events: list[EventItem]
    updated_at: datetime = Field(alias="updatedAt")

    model_config = {"populate_by_name": True}
