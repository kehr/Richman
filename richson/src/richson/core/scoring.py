"""Dimension scoring and percentile normalization engine.

Core algorithms:
1. blended_percentile: dual-window (1Y/5Y) percentile normalization
2. weighted_dimension_score: weighted average of available sub-indicators
3. compute_overall_score: combine four dimension scores with weights

All functions are pure computation (no IO).
"""

from __future__ import annotations

import numpy as np
import pandas as pd


def blended_percentile(
    current_value: float,
    history_1y: pd.Series,
    history_5y: pd.Series,
    invert: bool = False,
) -> float:
    """Compute dual-window blended percentile (TRD SS7.2).

    Uses 70% weight on 1-year percentile and 30% on 5-year percentile to
    balance recency sensitivity with long-term context.

    Args:
        current_value: the latest observation value.
        history_1y: series of observations for the past 1 year.
        history_5y: series of observations for the past 5 years.
        invert: if True, lower values are more bullish (e.g. real yield, DXY).

    Returns:
        Blended percentile 0-100 where 100 = most bullish for gold.
        Returns 50.0 if insufficient data.
    """
    if history_1y.empty:
        return 50.0

    pct_1y = float(np.sum(history_1y <= current_value) / len(history_1y) * 100)
    if len(history_5y) >= 90:
        pct_5y = float(np.sum(history_5y <= current_value) / len(history_5y) * 100)
    else:
        pct_5y = pct_1y

    blended = 0.70 * pct_1y + 0.30 * pct_5y
    return round((100.0 - blended) if invert else blended, 2)


def weighted_dimension_score(sub_indicators: list[dict]) -> float:
    """Compute weighted average score for a dimension.

    Handles missing/unavailable sub-indicators by re-normalizing weights
    among available indicators.

    Args:
        sub_indicators: list of sub-indicator dicts, each with keys:
            ``normalized_score`` (float | None),
            ``weight_in_dimension`` (float),
            ``status`` (str).

    Returns:
        Float 0-100. Returns 50.0 if no indicators are available.
    """
    available = [
        s for s in sub_indicators
        if s.get("status") == "ok" and s.get("normalized_score") is not None
    ]

    if not available:
        return 50.0  # neutral fallback

    total_weight = sum(s["weight_in_dimension"] for s in available)
    if total_weight <= 0:
        return 50.0

    weighted_sum = sum(
        s["normalized_score"] * s["weight_in_dimension"] for s in available
    )
    return round(weighted_sum / total_weight, 2)


def compute_overall_score(
    dimension_scores: dict[str, float | None],
    dimension_weights: dict[str, float],
) -> float:
    """Combine four dimension scores into a single composite score.

    Handles unavailable dimensions by re-normalizing weights of available ones.
    Requires at least 2 available dimensions (caller should check before calling).

    Args:
        dimension_scores: dict mapping dimension key (e.g. ``d1``) to score float.
            None values indicate unavailable dimensions.
        dimension_weights: dict mapping dimension key to weight (must sum to 1.0).

    Returns:
        Composite score 0-100.

    Raises:
        ValueError: if fewer than 2 dimensions are available.
    """
    available_dims = {
        k: v for k, v in dimension_scores.items()
        if v is not None
    }

    if len(available_dims) < 2:
        raise ValueError(
            f"At least 2 dimensions required for scoring, got {len(available_dims)}"
        )

    total_weight = sum(dimension_weights.get(k, 0.0) for k in available_dims)
    if total_weight <= 0:
        return 50.0

    weighted_sum = sum(
        v * dimension_weights.get(k, 0.0) for k, v in available_dims.items()
    )
    return round(weighted_sum / total_weight, 2)


def signal_level_from_score(score: float) -> str:
    """Map a composite score to a signal level string.

    Args:
        score: composite score 0-100.

    Returns:
        One of: ``strong_bullish``, ``moderate_bullish``, ``neutral``,
        ``moderate_bearish``, ``strong_bearish``.
    """
    if score >= 75:
        return "strong_bullish"
    elif score >= 60:
        return "moderate_bullish"
    elif score >= 40:
        return "neutral"
    elif score >= 25:
        return "moderate_bearish"
    else:
        return "strong_bearish"


def percentile_rank_in_history(score: float, history: list[float]) -> float:
    """Compute the percentile rank of a score within a historical distribution.

    Used by richman to compute ``percentileLabel`` (P90+, P75-89, etc.)

    Args:
        score: current composite score.
        history: list of past composite scores (recent first or any order).

    Returns:
        Percentile rank 0-100.
    """
    if not history:
        return 50.0
    arr = np.array(history, dtype=float)
    return float(np.sum(arr <= score) / len(arr) * 100)
