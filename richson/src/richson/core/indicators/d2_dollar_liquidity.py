"""D2 Dollar/Liquidity dimension indicator calculator.

Sub-indicators:
- DXY (US Dollar Index)         -- weight 40%  [inverted: weaker dollar = bullish gold]
- M2 money supply growth (YoY)  -- weight 35%  [higher M2 growth = bullish gold]
- TGA balance proxy             -- weight 25%  [lower TGA = more liquidity = bullish gold]

TGA (Treasury General Account) is a US government cash account at the Fed.
When TGA depletes, dollars flow into the financial system (liquidity positive for gold).
We approximate TGA balance from the FRED series WTREGEN.

Input: DXY OHLCV DataFrame, M2SL Series, optional TGA Series.
Output: dict with per-indicator scores and weighted base_score.
"""

from __future__ import annotations

import numpy as np
import pandas as pd
import structlog

from richson.core.scoring import blended_percentile, weighted_dimension_score

logger = structlog.get_logger(__name__)

# Sub-indicator weights
_WEIGHTS = {
    "dxy": 0.40,
    "m2_growth": 0.35,
    "tga_balance": 0.25,
}

_MIN_HISTORY_DAYS = 90


def compute_d2_indicators(
    dxy_ohlcv: pd.DataFrame | None,
    m2sl: pd.Series | None,
    tga: pd.Series | None = None,
) -> dict:
    """Compute D2 dollar/liquidity dimension scores.

    Args:
        dxy_ohlcv: DXY daily OHLCV DataFrame from Yahoo Finance.
        m2sl: FRED M2SL series (monthly, billions USD).
        tga: FRED WTREGEN series (weekly, millions USD). Optional.

    Returns:
        Dict with ``sub_indicators``, ``base_score``, ``available_count``, ``total_count``.
    """
    sub_indicators = []

    # --- DXY (inverted: weaker dollar is bullish gold) ---
    dxy_result = _compute_dxy(dxy_ohlcv)
    sub_indicators.append(dxy_result)

    # --- M2 year-over-year growth rate ---
    m2_result = _compute_m2_growth(m2sl)
    sub_indicators.append(m2_result)

    # --- TGA balance (inverted: lower TGA = more liquidity = bullish gold) ---
    tga_result = _compute_tga(tga)
    sub_indicators.append(tga_result)

    base_score = weighted_dimension_score(sub_indicators)

    return {
        "sub_indicators": sub_indicators,
        "base_score": base_score,
        "available_count": sum(1 for s in sub_indicators if s["status"] == "ok"),
        "total_count": len(sub_indicators),
    }


def _compute_dxy(dxy_ohlcv: pd.DataFrame | None) -> dict:
    """Compute DXY indicator score.

    Uses daily close prices. Inverted so weaker dollar = higher score.
    """
    if dxy_ohlcv is None or dxy_ohlcv.empty:
        return _unavailable("DXY", _WEIGHTS["dxy"], "Yahoo/DX-Y.NYB")

    close = dxy_ohlcv["close"].dropna()
    if len(close) < _MIN_HISTORY_DAYS:
        return _unavailable("DXY", _WEIGHTS["dxy"], "Yahoo/DX-Y.NYB")

    raw_value = float(close.iloc[-1])
    now = close.index[-1]
    history_1y = close[close.index >= now - pd.DateOffset(years=1)]
    history_5y = close[close.index >= now - pd.DateOffset(years=5)]

    pct_1y = float(np.sum(history_1y <= raw_value) / len(history_1y) * 100)
    pct_5y = float(np.sum(history_5y <= raw_value) / len(history_5y) * 100) if len(history_5y) >= 90 else pct_1y
    blended = blended_percentile(raw_value, history_1y, history_5y, invert=True)

    return {
        "name": "DXY",
        "raw_value": round(raw_value, 2),
        "percentile_1y": round(pct_1y, 2),
        "percentile_5y": round(pct_5y, 2),
        "blended_percentile": round(blended, 2),
        "normalized_score": round(blended, 2),
        "weight_in_dimension": _WEIGHTS["dxy"],
        "data_source": "Yahoo/DX-Y.NYB",
        "status": "ok",
    }


def _compute_m2_growth(m2sl: pd.Series | None) -> dict:
    """Compute M2 year-over-year growth rate and its blended percentile.

    Higher M2 growth is bullish for gold (not inverted).
    """
    if m2sl is None or m2sl.empty:
        return _unavailable("M2 YoY Growth", _WEIGHTS["m2_growth"], "FRED/M2SL")

    m2sl = m2sl.dropna().sort_index()
    if len(m2sl) < 13:  # Need at least 13 monthly observations for YoY
        return _unavailable("M2 YoY Growth", _WEIGHTS["m2_growth"], "FRED/M2SL")

    # Compute YoY growth rate
    m2_yoy = m2sl.pct_change(periods=12) * 100  # 12 months back
    m2_yoy = m2_yoy.dropna()
    if m2_yoy.empty:
        return _unavailable("M2 YoY Growth", _WEIGHTS["m2_growth"], "FRED/M2SL")

    raw_value = float(m2_yoy.iloc[-1])
    now = m2_yoy.index[-1]
    history_1y = m2_yoy[m2_yoy.index >= now - pd.DateOffset(years=1)]
    history_5y = m2_yoy[m2_yoy.index >= now - pd.DateOffset(years=5)]

    if len(history_1y) < 3:
        return _unavailable("M2 YoY Growth", _WEIGHTS["m2_growth"], "FRED/M2SL")

    pct_1y = float(np.sum(history_1y <= raw_value) / len(history_1y) * 100)
    pct_5y = float(np.sum(history_5y <= raw_value) / len(history_5y) * 100) if len(history_5y) >= 3 else pct_1y
    blended = blended_percentile(raw_value, history_1y, history_5y, invert=False)

    return {
        "name": "M2 YoY Growth",
        "raw_value": round(raw_value, 4),
        "percentile_1y": round(pct_1y, 2),
        "percentile_5y": round(pct_5y, 2),
        "blended_percentile": round(blended, 2),
        "normalized_score": round(blended, 2),
        "weight_in_dimension": _WEIGHTS["m2_growth"],
        "data_source": "FRED/M2SL",
        "status": "ok",
    }


def _compute_tga(tga: pd.Series | None) -> dict:
    """Compute TGA balance indicator score.

    Inverted: lower TGA balance = more liquidity injected = bullish gold.
    If TGA data is unavailable, the indicator is marked unavailable and
    its weight is redistributed by the scoring layer.
    """
    if tga is None or tga.empty:
        return _unavailable("TGA Balance", _WEIGHTS["tga_balance"], "FRED/WTREGEN")

    tga = tga.dropna().sort_index()
    if len(tga) < _MIN_HISTORY_DAYS // 7:  # Weekly data, ~13 observations minimum
        return _unavailable("TGA Balance", _WEIGHTS["tga_balance"], "FRED/WTREGEN")

    raw_value = float(tga.iloc[-1])
    now = tga.index[-1]
    history_1y = tga[tga.index >= now - pd.DateOffset(years=1)]
    history_5y = tga[tga.index >= now - pd.DateOffset(years=5)]

    pct_1y = float(np.sum(history_1y <= raw_value) / len(history_1y) * 100)
    pct_5y = float(np.sum(history_5y <= raw_value) / len(history_5y) * 100) if len(history_5y) >= 5 else pct_1y
    blended = blended_percentile(raw_value, history_1y, history_5y, invert=True)

    return {
        "name": "TGA Balance",
        "raw_value": round(raw_value, 0),
        "percentile_1y": round(pct_1y, 2),
        "percentile_5y": round(pct_5y, 2),
        "blended_percentile": round(blended, 2),
        "normalized_score": round(blended, 2),
        "weight_in_dimension": _WEIGHTS["tga_balance"],
        "data_source": "FRED/WTREGEN",
        "status": "ok",
    }


def _unavailable(name: str, weight: float, data_source: str) -> dict:
    """Return an unavailable sub-indicator dict."""
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
