"""Cross-dimension conflict detection (TRD SS7.8).

Detects when dimension scores point in conflicting directions:
- Strong conflict: any two dimensions where one >= 70 and another <= 30
- Weak conflict: max - min > 40, but no strong conflict

Used to populate rs_asset_analyses.conflict_type and .conflict_message.

Input: dict of dimension scores.
Output: (conflict_type, conflict_message) tuple.
"""

from __future__ import annotations

_STRONG_BULLISH = 70.0
_STRONG_BEARISH = 30.0
_WEAK_SPREAD = 40.0


def detect_conflict(
    dimension_scores: dict[str, float],
    dimension_names: dict[str, str] | None = None,
) -> tuple[str | None, str | None]:
    """Detect cross-dimension conflicts (TRD SS7.8).

    Args:
        dimension_scores: dict mapping dimension key (e.g. ``d1``) to score float.
            Only include available (non-None) dimensions.
        dimension_names: optional dict mapping dimension key to display name.
            Used to build the conflict message. If None, uses key as name.

    Returns:
        Tuple of (conflict_type, conflict_message):
        - conflict_type: ``strong`` | ``weak`` | None
        - conflict_message: human-readable description or None
    """
    if not dimension_scores:
        return None, None

    names = dimension_names or {}
    scores_list = list(dimension_scores.items())

    # Check for strong conflict
    for i, (k1, s1) in enumerate(scores_list):
        for k2, s2 in scores_list[i + 1 :]:
            if (s1 >= _STRONG_BULLISH and s2 <= _STRONG_BEARISH) or (
                s2 >= _STRONG_BULLISH and s1 <= _STRONG_BEARISH
            ):
                bullish_dim = k1 if s1 >= _STRONG_BULLISH else k2
                bearish_dim = k2 if s1 >= _STRONG_BULLISH else k1
                bullish_name = names.get(bullish_dim, bullish_dim.upper())
                bearish_name = names.get(bearish_dim, bearish_dim.upper())
                bullish_score = dimension_scores[bullish_dim]
                bearish_score = dimension_scores[bearish_dim]
                message = (
                    f"{bullish_name}({bullish_score:.0f}) and "
                    f"{bearish_name}({bearish_score:.0f}) signal conflicting directions"
                )
                return "strong", message

    # Check for weak conflict
    if len(dimension_scores) >= 2:
        max_score = max(dimension_scores.values())
        min_score = min(dimension_scores.values())
        if max_score - min_score > _WEAK_SPREAD:
            max_dim = max(dimension_scores, key=lambda k: dimension_scores[k])
            min_dim = min(dimension_scores, key=lambda k: dimension_scores[k])
            max_name = names.get(max_dim, max_dim.upper())
            min_name = names.get(min_dim, min_dim.upper())
            message = (
                f"Spread between {max_name}({max_score:.0f}) and "
                f"{min_name}({min_score:.0f}) is elevated ({max_score - min_score:.0f} points)"
            )
            return "weak", message

    return None, None


def check_llm_anomaly(
    dimension: str,
    llm_adjustment: float,
    anomaly_threshold: float = 10.0,
) -> bool:
    """Check if an LLM adjustment qualifies as anomalous (TRD SS8.2).

    An anomaly flag is set when abs(llm_adjustment) > 10 for a dimension.
    This is a runtime flag only -- not persisted to the database.

    Args:
        dimension: dimension identifier (e.g. ``D1``).
        llm_adjustment: numeric adjustment applied by LLM research.
        anomaly_threshold: threshold for flagging (default 10.0).

    Returns:
        True if the adjustment is anomalous.
    """
    return abs(llm_adjustment) > anomaly_threshold
