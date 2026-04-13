"""Polymarket event probability change monitoring (TRD SS3.6).

Monitors gold-relevant Polymarket events and detects significant probability
changes (>= 20% threshold). Generates alert records for writing to
rs_event_alerts table.

This module is purely computational -- it does not write to the database.
The caller (pipeline or scheduler) is responsible for persistence.
"""

from __future__ import annotations

import logging
from typing import Any

logger = logging.getLogger(__name__)

# Default change threshold for triggering an alert
_DEFAULT_THRESHOLD = 0.20  # 20 percentage points

# Mapping from market keywords to gold price direction implication
_KEYWORD_GOLD_DIRECTION: list[tuple[list[str], str]] = [
    (["rate cut", "fed cut", "lower rates", "easing"], "bullish"),
    (["rate hike", "hawkish", "tighten", "higher rates"], "bearish"),
    (["war", "conflict", "sanctions", "geopolitical"], "bullish"),
    (["recession", "crisis", "default"], "bullish"),
    (["risk on", "growth", "equity rally"], "bearish"),
]


def detect_probability_changes(
    current_markets: list[dict[str, Any]],
    previous_markets: list[dict[str, Any]],
    threshold: float = _DEFAULT_THRESHOLD,
) -> list[dict[str, Any]]:
    """Compare current vs previous Polymarket snapshots to detect probability shifts.

    Args:
        current_markets: list of market dicts from PolymarketClient (current poll).
        previous_markets: list of market dicts from the previous poll.
        threshold: minimum absolute probability change to trigger an alert (0.0-1.0).

    Returns:
        List of alert dicts with structure::

            {
                "event_slug": str,
                "event_title": str,
                "source": "polymarket",
                "prev_probability": float,
                "curr_probability": float,
                "delta": float,
                "threshold": float,
                "gold_direction": str | None,
            }
        Empty list if no significant changes detected.
    """
    # Index previous markets by slug
    prev_by_slug: dict[str, dict[str, Any]] = {
        m.get("slug", ""): m for m in previous_markets if m.get("slug")
    }

    alerts: list[dict[str, Any]] = []

    for market in current_markets:
        slug = market.get("slug", "")
        if not slug:
            continue

        curr_prob = float(market.get("yes_probability", 0.5))
        prev_market = prev_by_slug.get(slug)
        if prev_market is None:
            # New market - no previous data to compare
            continue

        prev_prob = float(prev_market.get("yes_probability", 0.5))
        delta = curr_prob - prev_prob

        if abs(delta) < threshold:
            continue

        # Determine gold price direction implication
        gold_direction = _infer_gold_direction(market.get("question", ""))

        alerts.append(
            {
                "event_slug": slug,
                "event_title": market.get("question", ""),
                "source": "polymarket",
                "prev_probability": round(prev_prob, 4),
                "curr_probability": round(curr_prob, 4),
                "delta": round(delta, 4),
                "threshold": threshold,
                "gold_direction": gold_direction,
            }
        )
        logger.info(
            "event monitor: probability change detected",
            slug=slug,
            delta=delta,
            gold_direction=gold_direction,
        )

    return alerts


def _infer_gold_direction(question: str) -> str | None:
    """Infer gold price implication from event question text.

    Args:
        question: Polymarket market question string.

    Returns:
        ``bullish``, ``bearish``, or None if direction is unclear.
    """
    question_lower = question.lower()

    for keywords, direction in _KEYWORD_GOLD_DIRECTION:
        if any(kw in question_lower for kw in keywords):
            return direction

    return None


def build_event_snapshot(markets: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Extract a minimal snapshot of market probabilities for comparison.

    Strips heavy fields to reduce memory usage when storing the previous snapshot.

    Args:
        markets: full market dicts from PolymarketClient.

    Returns:
        List of lightweight snapshot dicts: {slug, question, yes_probability}.
    """
    return [
        {
            "slug": m.get("slug", ""),
            "question": m.get("question", ""),
            "yes_probability": m.get("yes_probability", 0.5),
        }
        for m in markets
        if m.get("slug")
    ]
