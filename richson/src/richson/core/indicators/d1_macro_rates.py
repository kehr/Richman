"""D1 Macro/Rate dimension indicator calculator.

Computes sub-indicators for the macro interest rate dimension:
- Federal funds rate (FEDFUNDS) -- weight 20%
- Real yield 10Y TIPS (DFII10)  -- weight 35%  [inverted: lower = more bullish]
- Yield curve spread (T10Y2Y)   -- weight 25%
- Nominal yield 10Y (DGS10)     -- weight 10%
- Fed rate-cut probability (Polymarket) -- weight 10%

All sub-indicators are normalized to 0-100 via dual-window blended percentile.
Higher score = more bullish for gold.

Input: dict of pandas Series from FRED and Polymarket probability float.
Output: dict with per-indicator raw values, percentiles, and weighted score.
"""

from __future__ import annotations

import numpy as np
import pandas as pd
import structlog

from richson.core.scoring import blended_percentile, weighted_dimension_score

logger = structlog.get_logger(__name__)

# Sub-indicator weights (must sum to 1.0)
_WEIGHTS = {
    "real_yield_10y": 0.35,    # DFII10 -- most important for gold
    "yield_curve": 0.25,       # T10Y2Y
    "fed_funds_rate": 0.20,    # FEDFUNDS
    "nominal_yield_10y": 0.10, # DGS10
    "rate_cut_probability": 0.10,  # Polymarket
}


def compute_d1_indicators(
    fedfunds: pd.Series | None,
    t10y2y: pd.Series | None,
    dfii10: pd.Series | None,
    dgs10: pd.Series | None,
    rate_cut_probability: float | None = None,
) -> dict:
    """Compute D1 macro/rate dimension indicator scores.

    Args:
        fedfunds: FRED FEDFUNDS series (daily/monthly, percent).
        t10y2y: FRED T10Y2Y series (daily, percent). Negative = inverted curve.
        dfii10: FRED DFII10 series (daily, percent). Lower = more bullish gold.
        dgs10: FRED DGS10 series (daily, percent).
        rate_cut_probability: Polymarket Yes probability for rate cut (0.0-1.0).

    Returns:
        Dict with structure::

            {
                "sub_indicators": [
                    {
                        "name": str,
                        "raw_value": float | None,
                        "percentile_1y": float | None,
                        "percentile_5y": float | None,
                        "blended_percentile": float | None,
                        "normalized_score": float | None,
                        "weight_in_dimension": float,
                        "data_source": str,
                        "status": "ok" | "unavailable",
                    },
                    ...
                ],
                "base_score": float,  # 0-100, weighted average of available indicators
                "available_count": int,
                "total_count": int,
            }
    """
    sub_indicators = []

    # --- Real yield 10Y (DFII10) -- inverted: lower real yield is bullish gold ---
    dfii10_result = _compute_series_indicator(
        name="10Y TIPS Yield",
        series=dfii10,
        weight=_WEIGHTS["real_yield_10y"],
        data_source="FRED/DFII10",
        invert=True,  # lower real yield -> higher gold score
    )
    sub_indicators.append(dfii10_result)

    # --- Yield curve spread (T10Y2Y) ---
    # Inverted curve (negative spread) historically associated with risk-off / gold demand
    # We invert: lower (more inverted) spread -> higher score
    yield_curve_result = _compute_series_indicator(
        name="10Y-2Y Yield Spread",
        series=t10y2y,
        weight=_WEIGHTS["yield_curve"],
        data_source="FRED/T10Y2Y",
        invert=True,
    )
    sub_indicators.append(yield_curve_result)

    # --- Federal funds rate (FEDFUNDS) -- inverted: lower rate is bullish gold ---
    fedfunds_result = _compute_series_indicator(
        name="Fed Funds Rate",
        series=fedfunds,
        weight=_WEIGHTS["fed_funds_rate"],
        data_source="FRED/FEDFUNDS",
        invert=True,  # lower rate -> higher score
    )
    sub_indicators.append(fedfunds_result)

    # --- Nominal yield 10Y (DGS10) -- inverted ---
    dgs10_result = _compute_series_indicator(
        name="10Y Treasury Yield",
        series=dgs10,
        weight=_WEIGHTS["nominal_yield_10y"],
        data_source="FRED/DGS10",
        invert=True,
    )
    sub_indicators.append(dgs10_result)

    # --- Rate cut probability (Polymarket) ---
    # Higher probability of rate cut -> more bullish for gold -> direct (not inverted)
    rate_cut_result = _compute_probability_indicator(
        name="Fed Rate Cut Probability",
        probability=rate_cut_probability,
        weight=_WEIGHTS["rate_cut_probability"],
        data_source="Polymarket",
    )
    sub_indicators.append(rate_cut_result)

    base_score = weighted_dimension_score(sub_indicators)

    return {
        "sub_indicators": sub_indicators,
        "base_score": base_score,
        "available_count": sum(1 for s in sub_indicators if s["status"] == "ok"),
        "total_count": len(sub_indicators),
    }


def _compute_series_indicator(
    name: str,
    series: pd.Series | None,
    weight: float,
    data_source: str,
    invert: bool = False,
) -> dict:
    """Compute blended percentile for a single FRED series.

    Args:
        name: human-readable indicator name.
        series: time series of values.
        weight: weight within the D1 dimension (0.0-1.0).
        data_source: source identifier string.
        invert: if True, lower values = more bullish.

    Returns:
        Sub-indicator result dict.
    """
    if series is None or series.empty:
        return {
            "name": name,
            "raw_value": None,
            "percentile_1y": None,
            "percentile_5y": None,
            "blended_percentile": None,
            "normalized_score": None,
            "weight_in_dimension": weight,
            "data_source": data_source,
            "status": "unavailable",
        }

    series = series.dropna()
    if series.empty:
        return {
            "name": name,
            "raw_value": None,
            "percentile_1y": None,
            "percentile_5y": None,
            "blended_percentile": None,
            "normalized_score": None,
            "weight_in_dimension": weight,
            "data_source": data_source,
            "status": "unavailable",
        }

    raw_value = float(series.iloc[-1])

    now = series.index[-1]
    cutoff_1y = now - pd.DateOffset(years=1)
    cutoff_5y = now - pd.DateOffset(years=5)

    history_1y = series[series.index >= cutoff_1y]
    history_5y = series[series.index >= cutoff_5y]

    if len(history_1y) < 90:
        # Insufficient data
        return {
            "name": name,
            "raw_value": raw_value,
            "percentile_1y": None,
            "percentile_5y": None,
            "blended_percentile": None,
            "normalized_score": None,
            "weight_in_dimension": weight,
            "data_source": data_source,
            "status": "unavailable",
        }

    pct_1y = float(np.sum(history_1y <= raw_value) / len(history_1y) * 100)
    pct_5y = float(np.sum(history_5y <= raw_value) / len(history_5y) * 100) if len(history_5y) >= 90 else pct_1y

    blended = blended_percentile(raw_value, history_1y, history_5y, invert=invert)

    return {
        "name": name,
        "raw_value": raw_value,
        "percentile_1y": round(pct_1y, 2),
        "percentile_5y": round(pct_5y, 2),
        "blended_percentile": round(blended, 2),
        "normalized_score": round(blended, 2),
        "weight_in_dimension": weight,
        "data_source": data_source,
        "status": "ok",
    }


def _compute_probability_indicator(
    name: str,
    probability: float | None,
    weight: float,
    data_source: str,
) -> dict:
    """Convert a raw 0-1 probability to a 0-100 score.

    No historical percentile is computed; probability is directly scaled.

    Args:
        name: indicator name.
        probability: raw probability 0.0-1.0, or None if unavailable.
        weight: dimension weight.
        data_source: source identifier.

    Returns:
        Sub-indicator result dict.
    """
    if probability is None:
        return {
            "name": name,
            "raw_value": None,
            "percentile_1y": None,
            "percentile_5y": None,
            "blended_percentile": None,
            "normalized_score": None,
            "weight_in_dimension": weight,
            "data_source": data_source,
            "status": "unavailable",
        }

    score = float(probability) * 100.0
    score = max(0.0, min(100.0, score))

    return {
        "name": name,
        "raw_value": round(float(probability), 4),
        "percentile_1y": None,
        "percentile_5y": None,
        "blended_percentile": score,
        "normalized_score": round(score, 2),
        "weight_in_dimension": weight,
        "data_source": data_source,
        "status": "ok",
    }
