"""Composite confidence calculation (TRD SS7.5).

Computes overall analysis confidence (0-100) based on:
1. Direction agreement among the four dimension scores (base confidence)
2. Data completeness deductions (FRED freshness, Polymarket availability, LLM)
3. Partial dimension coverage deductions

Also computes confidence band [low, high] as +/-5 around the point estimate.
"""

from __future__ import annotations

import datetime


def compute_confidence(
    dimension_scores: dict[str, float | None],
    data_completeness: dict[str, object],
    llm_available: bool,
) -> tuple[float, float, float]:
    """Compute composite confidence score with band.

    Algorithm (TRD SS7.5):
    Base confidence from direction agreement:
    - All 4 agree (all >= 50 or all < 50): 90%
    - 3 agree, 1 diverge: 70%
    - 2v2 split: 50%
    - 3+ diverge: 30%

    Deductions:
    - FRED data delay > 3 days: -15%
    - Polymarket unavailable: -5%
    - LLM failed: -10%
    - Each unavailable dimension: -15% additional

    Args:
        dimension_scores: dict mapping dimension key (d1-d4) to score or None.
            None = dimension unavailable.
        data_completeness: dict with boolean flags:
            ``fred_fresh`` (bool) -- True if FRED data is within 3 days
            ``polymarket`` (bool) -- True if Polymarket data is available
        llm_available: True if Layer 2 LLM completed successfully.

    Returns:
        Tuple of (confidence, band_low, band_high) all in range [0, 100].
    """
    available_scores = [v for v in dimension_scores.values() if v is not None]

    # Base confidence from direction agreement
    if len(available_scores) >= 4:
        base = _direction_agreement_base(available_scores[:4])
    elif len(available_scores) >= 2:
        base = _direction_agreement_base(available_scores)
        # Fewer dimensions -> less certainty
        missing_count = 4 - len(available_scores)
        base -= missing_count * 15.0
    else:
        base = 20.0  # very low confidence with < 2 dimensions

    # Apply deductions
    if not data_completeness.get("fred_fresh", True):
        base -= 15.0
    if not data_completeness.get("polymarket", True):
        base -= 5.0
    if not llm_available:
        base -= 10.0

    confidence = max(0.0, min(100.0, base))
    band_low = max(0.0, confidence - 5.0)
    band_high = min(100.0, confidence + 5.0)

    return round(confidence, 2), round(band_low, 2), round(band_high, 2)


def _direction_agreement_base(scores: list[float]) -> float:
    """Compute base confidence from direction agreement.

    Args:
        scores: list of dimension scores (2-4 values).

    Returns:
        Base confidence percentage (0-100).
    """
    bullish_count = sum(1 for s in scores if s >= 50)
    bearish_count = len(scores) - bullish_count

    if bullish_count == len(scores) or bearish_count == len(scores):
        return 90.0  # all agree
    elif max(bullish_count, bearish_count) == len(scores) - 1:
        return 70.0  # 3 agree, 1 diverge (for 4 dims)
    else:
        return 50.0  # split


def check_fred_freshness(
    fred_last_date: datetime.date | None,
    max_delay_days: int = 3,
) -> bool:
    """Check if FRED data is fresh enough.

    Args:
        fred_last_date: date of the most recent FRED observation.
        max_delay_days: maximum allowed delay in calendar days.

    Returns:
        True if fresh (within allowed delay), False if stale or unknown.
    """
    if fred_last_date is None:
        return False
    today = datetime.date.today()
    delay = (today - fred_last_date).days
    return delay <= max_delay_days


def compute_data_coverage_label(
    dimension_scores: dict[str, float | None],
    fred_fresh: bool,
    polymarket_available: bool,
) -> str:
    """Determine data coverage label for rs_asset_analyses.

    Args:
        dimension_scores: dimension scores dict (None = unavailable).
        fred_fresh: whether FRED data is current.
        polymarket_available: whether Polymarket data fetched successfully.

    Returns:
        ``full`` | ``partial`` | ``degraded``
    """
    unavailable_count = sum(1 for v in dimension_scores.values() if v is None)

    if unavailable_count == 0 and fred_fresh and polymarket_available:
        return "full"
    elif unavailable_count <= 1 or not fred_fresh or not polymarket_available:
        return "partial"
    else:
        return "degraded"
