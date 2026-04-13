"""LLM qualitative-to-numeric adjustment mapping (TRD SS7.4).

Maps Layer 2 LLM structured output (direction, magnitude, confidence)
to a numeric score adjustment for each dimension.

Constraints (PRD SS3.1 / SS3.4):
- Per-source cap: +/-8 points
- Total adjustment cap: +/-15 points across all sources
- ``magnitude=major`` + ``confidence=low`` -> treated as ``moderate``/``medium``
- ``magnitude=major`` from single source -> capped to ``moderate`` range
"""

from __future__ import annotations

from typing import Literal

# (magnitude, confidence) -> (min_adj, max_adj)
_ADJUSTMENT_MAP: dict[tuple[str, str], tuple[float, float]] = {
    ("major", "high"):      (12.0, 15.0),
    ("major", "medium"):    (8.0,  11.0),
    ("major", "low"):       (3.0,   5.0),  # degraded to moderate/medium range
    ("moderate", "high"):   (6.0,   8.0),
    ("moderate", "medium"): (3.0,   5.0),
    ("moderate", "low"):    (1.0,   2.0),
    ("minor", "high"):      (1.0,   2.0),
    ("minor", "medium"):    (1.0,   2.0),
    ("minor", "low"):       (0.5,   1.0),
}

_PER_SOURCE_CAP = 8.0
_TOTAL_CAP = 15.0


class LLMAdjustmentEvent:
    """Represents a single qualitative judgment from Layer 2 LLM.

    Attributes:
        dimension: dimension identifier (``D1``, ``D2``, ``D3``).
        direction: ``bullish``, ``bearish``, or ``neutral``.
        magnitude: ``major``, ``moderate``, or ``minor``.
        confidence: ``high``, ``medium``, or ``low``.
        source_count: number of independent sources supporting this judgment.
    """

    def __init__(
        self,
        dimension: str,
        direction: Literal["bullish", "bearish", "neutral"],
        magnitude: Literal["major", "moderate", "minor"],
        confidence: Literal["high", "medium", "low"],
        source_count: int = 1,
    ) -> None:
        self.dimension = dimension
        self.direction = direction
        self.magnitude = magnitude
        self.confidence = confidence
        self.source_count = source_count


def compute_adjustment(events: list[LLMAdjustmentEvent]) -> dict[str, float]:
    """Compute per-dimension numeric adjustments from LLM events.

    Applies per-source cap (+/-8) and total cap (+/-15) per dimension.
    Events from the same dimension are accumulated.

    Args:
        events: list of LLMAdjustmentEvent from Layer 2 research_agent.

    Returns:
        Dict mapping dimension key (``D1``, ``D2``, ``D3``) to adjustment float.
        Positive = bullish, negative = bearish.
        D4 is not included (no LLM adjustment for technical dimension).
    """
    # Group events by dimension
    dimension_events: dict[str, list[LLMAdjustmentEvent]] = {}
    for event in events:
        dim = event.dimension.upper()
        if dim not in dimension_events:
            dimension_events[dim] = []
        dimension_events[dim].append(event)

    result: dict[str, float] = {}

    for dim, dim_events in dimension_events.items():
        total_adj = 0.0
        for event in dim_events:
            single_adj = _compute_single_event_adjustment(event)
            total_adj += single_adj

        # Apply total cap
        total_adj = max(-_TOTAL_CAP, min(_TOTAL_CAP, total_adj))
        result[dim] = round(total_adj, 2)

    return result


def _compute_single_event_adjustment(event: LLMAdjustmentEvent) -> float:
    """Compute numeric adjustment for one LLM event.

    Args:
        event: single LLMAdjustmentEvent.

    Returns:
        Signed adjustment float (before total cap).
    """
    if event.direction == "neutral":
        return 0.0

    magnitude = event.magnitude
    confidence = event.confidence

    # Degrade major+low to moderate range
    if magnitude == "major" and confidence == "low":
        magnitude = "moderate"
        confidence = "medium"

    # Single-source major events are capped to moderate range
    if magnitude == "major" and event.source_count < 2:
        magnitude = "moderate"
        confidence = max_confidence(confidence, "medium")

    key = (magnitude, confidence)
    min_adj, max_adj = _ADJUSTMENT_MAP.get(key, (1.0, 2.0))
    midpoint = (min_adj + max_adj) / 2.0

    # Apply per-source cap
    midpoint = min(midpoint, _PER_SOURCE_CAP)

    return midpoint if event.direction == "bullish" else -midpoint


def max_confidence(c1: str, c2: str) -> str:
    """Return the lower of two confidence levels (more conservative).

    Ordering: high > medium > low
    """
    order = {"high": 2, "medium": 1, "low": 0}
    return c1 if order.get(c1, 0) <= order.get(c2, 0) else c2


def apply_adjustment_to_score(base_score: float, adjustment: float) -> float:
    """Apply an LLM adjustment to a base dimension score.

    Ensures the result stays within [0, 100].

    Args:
        base_score: quantitative base score (0-100).
        adjustment: LLM adjustment (capped at +/-15).

    Returns:
        Final score 0-100.
    """
    return round(max(0.0, min(100.0, base_score + adjustment)), 2)
