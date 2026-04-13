"""Event radar endpoint.

GET /events/radar - upcoming macro events with Polymarket probabilities
"""

from __future__ import annotations

import asyncio
from datetime import UTC, datetime, timedelta

import structlog
from fastapi import APIRouter, Depends

from richson.api.auth import require_api_key

router = APIRouter(prefix="/events", dependencies=[Depends(require_api_key)])
logger = structlog.get_logger()

# Built-in economic calendar (fixed scheduled events)
# Each entry: (day_offset_from_today, title, category, impact, gold_direction)
_FIXED_EVENTS: list[tuple[int, str, str, str, str]] = [
    # These represent typical event windows; real deployment would use a calendar API
    (3, "FOMC Meeting Minutes Release", "monetary_policy", "high", "bullish"),
    (5, "US CPI Data Release", "inflation", "high", "bullish"),
    (7, "US Non-Farm Payrolls", "employment", "high", "neutral"),
    (10, "US PPI Data", "inflation", "medium", "neutral"),
    (14, "FOMC Member Speeches", "monetary_policy", "medium", "neutral"),
]


@router.get("/radar")
async def get_event_radar() -> dict:
    """Return upcoming macro events with Polymarket probabilities.

    Events are sourced from the built-in economic calendar plus Polymarket.
    When Polymarket is unavailable, events are shown without probability data.
    """
    from richson.datasources.polymarket import PolymarketClient  # noqa: PLC0415

    poly_client = PolymarketClient()

    # Fetch Polymarket data (non-blocking)
    polymarket_markets: list[dict] = []
    try:
        polymarket_markets = await asyncio.to_thread(poly_client.get_gold_relevant_markets)
    except Exception as exc:
        logger.warning("polymarket_unavailable", error=str(exc))

    # Build probability lookup from Polymarket
    poly_by_keyword: dict[str, tuple[float, float | None]] = {}
    for market in polymarket_markets:
        prob = market.get("yes_probability")
        question = market.get("question", "").lower()
        if prob is not None:
            poly_by_keyword[question] = (float(prob), None)  # second item = 24h change (unknown)

    today = datetime.now(tz=UTC)
    events: list[dict] = []

    # Fixed calendar events
    for day_offset, title, category, impact, gold_direction in _FIXED_EVENTS:
        event_date = today + timedelta(days=day_offset)

        # Try to match with Polymarket
        probability = None
        probability_source = None
        probability_change_24h = None

        title_lower = title.lower()
        for question, (prob, delta) in poly_by_keyword.items():
            if any(kw in question for kw in title_lower.split()[:3] if len(kw) > 3):
                probability = round(prob, 4)
                probability_source = "polymarket"
                probability_change_24h = delta
                break

        events.append({
            "date": event_date.strftime("%Y-%m-%d"),
            "title": title,
            "category": category,
            "impact": impact,
            "goldDirection": gold_direction,
            "probability": probability,
            "probabilitySource": probability_source,
            "probabilityChange24h": probability_change_24h,
        })

    # Add Polymarket-only events (events without a calendar match)
    for market in polymarket_markets[:5]:  # limit to top 5
        question = market.get("question", "")
        if not question:
            continue
        prob = market.get("yes_probability")
        end_date = market.get("end_date_iso", "")
        gold_direction = _infer_gold_direction(question)

        events.append({
            "date": end_date[:10] if end_date else today.strftime("%Y-%m-%d"),
            "title": question,
            "category": "market_event",
            "impact": "medium",
            "goldDirection": gold_direction,
            "probability": round(float(prob), 4) if prob is not None else None,
            "probabilitySource": "polymarket" if prob is not None else None,
            "probabilityChange24h": None,
        })

    # Sort by date
    events.sort(key=lambda e: e["date"])

    return {
        "data": {
            "events": events,
            "updatedAt": today.isoformat(),
        }
    }


def _infer_gold_direction(question: str) -> str | None:
    q = question.lower()
    bullish_kw = ["rate cut", "easing", "war", "conflict", "recession", "crisis", "geopolitical"]
    bearish_kw = ["rate hike", "hawkish", "risk on", "growth"]
    if any(kw in q for kw in bullish_kw):
        return "bullish"
    if any(kw in q for kw in bearish_kw):
        return "bearish"
    return None
