"""Event radar endpoint.

GET /events/radar - upcoming macro events from FRED + Polymarket.
"""

from __future__ import annotations

import asyncio
from datetime import UTC, datetime, timedelta
from typing import Any

import structlog
from fastapi import APIRouter, Depends

from richson.api.auth import require_api_key
from richson.config import settings
from richson.config.event_metadata import (
    FOMC_CALENDAR_URL,
    FOMC_MEETINGS,
    FRED_RELEASE_METADATA,
    fred_release_url,
    polymarket_event_url,
)
from richson.datasources.fred import FREDClient, FREDReleaseDate
from richson.datasources.polymarket import PolymarketClient

router = APIRouter(prefix="/events", dependencies=[Depends(require_api_key)])
logger = structlog.get_logger()

# Keep in sync with frontend i18n market.overview.eventRadar.subtitle
# ("Upcoming key macro events in the next 7 days").
EVENT_WINDOW_DAYS = 7

# Cap Polymarket entries to keep the radar focused on the highest-volume
# markets, matching the prior implementation.
_MAX_POLYMARKET_ENTRIES = 5


@router.get("/radar")
async def get_event_radar() -> dict[str, Any]:
    """Return upcoming macro events with optional Polymarket probabilities.

    FRED entries come from the official release calendar within a fixed
    7-day horizon; Polymarket entries come from gold-relevant markets whose
    end_date falls in the same horizon. Each source degrades independently.
    """
    fred_client = FREDClient(api_key=settings.fred_api_key)
    poly_client = PolymarketClient()

    fred_task = asyncio.to_thread(fred_client.get_upcoming_releases, EVENT_WINDOW_DAYS)
    poly_task = asyncio.to_thread(poly_client.get_gold_relevant_markets)

    fred_result, poly_result = await asyncio.gather(
        fred_task, poly_task, return_exceptions=True
    )

    fred_releases: list[FREDReleaseDate] = []
    if isinstance(fred_result, BaseException):
        logger.warning("fred unavailable for event radar", error=str(fred_result))
    else:
        fred_releases = fred_result

    poly_markets: list[dict[str, Any]] = []
    if isinstance(poly_result, BaseException):
        logger.warning("polymarket unavailable for event radar", error=str(poly_result))
    else:
        poly_markets = poly_result

    today = datetime.now(tz=UTC)
    horizon = today + timedelta(days=EVENT_WINDOW_DAYS)
    events: list[dict[str, Any]] = []

    # Build FRED entries from the whitelisted release metadata.
    for release in fred_releases:
        meta = FRED_RELEASE_METADATA.get(release.release_id)
        if meta is None:
            # Defensive: the client also filters by the whitelist.
            continue
        events.append({
            "date": release.date,
            "title": meta.en_title,
            "category": meta.category,
            "impact": meta.impact,
            "goldDirection": meta.gold_direction,
            "probability": None,
            "probabilitySource": None,
            "probabilityChange24h": None,
            "sourceUrl": fred_release_url(release.release_id),
            "sourceName": "FRED",
            "releaseId": release.release_id,
        })

    # Append FOMC meetings from the hand-maintained calendar. FRED's
    # releases/dates API cannot be used here: its FOMC-related release_ids
    # fire every business day (associated daily series updates), not on
    # actual meeting days. See event_metadata.FOMC_MEETINGS for source.
    today_date = today.date()
    horizon_date = horizon.date()
    for meeting in FOMC_MEETINGS:
        if meeting.date < today_date or meeting.date > horizon_date:
            continue
        events.append({
            "date": meeting.date.isoformat(),
            "title": meeting.en_title,
            "category": "monetary_policy",
            "impact": "high",
            "goldDirection": "bullish",
            "probability": None,
            "probabilitySource": None,
            "probabilityChange24h": None,
            "sourceUrl": FOMC_CALENDAR_URL,
            "sourceName": "Federal Reserve",
            "releaseId": None,
        })

    # Build Polymarket entries: highest-volume markets within the horizon.
    # polymarket.py:143 stores `market.get("endDate")` under key `"end_date"`.
    # The previous events.py used `"end_date_iso"` which never existed
    # (latent bug); read the correct key here.
    for market in poly_markets[:_MAX_POLYMARKET_ENTRIES]:
        slug = market.get("slug")
        question = market.get("question") or ""
        end_date_raw = market.get("end_date") or ""
        if not slug or not question:
            continue
        date_str = end_date_raw[:10] if end_date_raw else today.strftime("%Y-%m-%d")
        try:
            event_dt = datetime.fromisoformat(date_str).replace(tzinfo=UTC)
        except ValueError:
            continue
        if event_dt < today or event_dt > horizon:
            continue
        prob = market.get("yes_probability")
        events.append({
            "date": date_str,
            "title": question,
            "category": "market_event",
            "impact": "medium",
            "goldDirection": None,
            "probability": round(float(prob), 4) if prob is not None else None,
            "probabilitySource": "polymarket" if prob is not None else None,
            "probabilityChange24h": None,
            "sourceUrl": polymarket_event_url(str(slug)),
            "sourceName": "Polymarket",
            "releaseId": None,
        })

    events.sort(key=lambda e: e["date"])

    fred_count = sum(1 for e in events if e["sourceName"] == "FRED")
    fomc_count = sum(1 for e in events if e["sourceName"] == "Federal Reserve")
    polymarket_count = sum(1 for e in events if e["sourceName"] == "Polymarket")
    logger.info(
        "event radar built",
        fred_count=fred_count,
        fomc_count=fomc_count,
        polymarket_count=polymarket_count,
        total=len(events),
    )

    return {
        "data": {
            "events": events,
            "updatedAt": today.isoformat(),
        }
    }
