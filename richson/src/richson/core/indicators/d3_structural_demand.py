"""D3 Structural Demand dimension indicator calculator.

Sub-indicators:
- COT managed money net position   -- weight 30%
- Central bank net purchases       -- weight 40%
- ETF flows (GLD AUM change)       -- weight 20%
- Geopolitical risk (Polymarket)   -- weight 10%

COT data is weekly; central bank data is quarterly (held constant between updates).
ETF AUM change is computed from Yahoo Finance GLD daily data.

Input: COT DataFrame, WGC quarterly data dict, GLD OHLCV DataFrame,
       geopolitical risk probability float.
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
    "central_bank": 0.40,
    "cot_managed_money": 0.30,
    "etf_flows": 0.20,
    "geopolitical_risk": 0.10,
}

_MIN_COT_OBS = 26  # 6 months of weekly data


def compute_d3_indicators(
    cot_data: pd.DataFrame | None,
    wgc_data: dict | None,
    gld_ohlcv: pd.DataFrame | None,
    current_gold_price: float | None = None,
    aisc: float | None = None,
    geo_risk_prob: float | None = None,
) -> dict:
    """Compute D3 structural demand dimension scores.

    Args:
        cot_data: CFTC COT DataFrame with mm_net column (from COTClient).
        wgc_data: WGC quarterly data dict with central_bank_net_tonnes and aisc_usd_per_oz.
        gld_ohlcv: GLD ETF OHLCV from Yahoo Finance (for ETF AUM proxy).
        current_gold_price: current gold spot price in USD/oz.
        aisc: AISC in USD/oz (if provided separately; otherwise from wgc_data).
        geo_risk_prob: geopolitical risk index from Polymarket (0.0-1.0).

    Returns:
        Dict with ``sub_indicators``, ``base_score``, ``available_count``, ``total_count``.
    """
    sub_indicators = []

    # --- Central bank net purchases ---
    cb_tonnes = None
    effective_aisc = aisc
    if wgc_data:
        cb_tonnes = wgc_data.get("central_bank_net_tonnes")
        if effective_aisc is None:
            effective_aisc = wgc_data.get("aisc_usd_per_oz")

    cb_result = _compute_central_bank(cb_tonnes)
    sub_indicators.append(cb_result)

    # --- COT managed money net position ---
    cot_result = _compute_cot(cot_data)
    sub_indicators.append(cot_result)

    # --- ETF flows (GLD AUM proxy) ---
    etf_result = _compute_etf_flows(gld_ohlcv)
    sub_indicators.append(etf_result)

    # --- Geopolitical risk (higher risk = more gold demand = bullish) ---
    geo_result = _compute_geo_risk(geo_risk_prob)
    sub_indicators.append(geo_result)

    base_score = weighted_dimension_score(sub_indicators)

    result = {
        "sub_indicators": sub_indicators,
        "base_score": base_score,
        "available_count": sum(1 for s in sub_indicators if s["status"] == "ok"),
        "total_count": len(sub_indicators),
    }

    # Include AISC profit margin as metadata (not a scored sub-indicator)
    if effective_aisc and current_gold_price:
        aisc_margin = (current_gold_price - effective_aisc) / effective_aisc
        result["aisc_profit_margin"] = round(aisc_margin, 4)

    return result


def _compute_central_bank(cb_tonnes: float | None) -> dict:
    """Score central bank net purchases.

    Uses a simple threshold mapping (no historical percentile available
    for quarterly data with limited history):
    - >= 1000t/y : score 95
    - 800-1000t/y: score 80
    - 500-800t/y : score 65
    - 200-500t/y : score 50
    - 0-200t/y   : score 35
    - < 0 (net selling): score 10
    """
    name = "Central Bank Net Purchases"
    weight = _WEIGHTS["central_bank"]
    source = "WGC"

    if cb_tonnes is None:
        return _unavailable(name, weight, source)

    cb_tonnes = float(cb_tonnes)

    if cb_tonnes >= 1000:
        score = 95.0
    elif cb_tonnes >= 800:
        score = 80.0
    elif cb_tonnes >= 500:
        score = 65.0
    elif cb_tonnes >= 200:
        score = 50.0
    elif cb_tonnes >= 0:
        score = 35.0
    else:
        score = 10.0

    return {
        "name": name,
        "raw_value": round(cb_tonnes, 1),
        "percentile_1y": None,
        "percentile_5y": None,
        "blended_percentile": score,
        "normalized_score": score,
        "weight_in_dimension": weight,
        "data_source": source,
        "status": "ok",
    }


def _compute_cot(cot_data: pd.DataFrame | None) -> dict:
    """Compute COT managed money net position percentile.

    Higher net long position historically associated with bullish sentiment.
    Not inverted.
    """
    name = "COT Managed Money Net"
    weight = _WEIGHTS["cot_managed_money"]
    source = "CFTC/COT"

    if cot_data is None or cot_data.empty or "mm_net" not in cot_data.columns:
        return _unavailable(name, weight, source)

    mm_net = cot_data["mm_net"].dropna()
    if len(mm_net) < _MIN_COT_OBS:
        return _unavailable(name, weight, source)

    raw_value = float(mm_net.iloc[-1])
    now = mm_net.index[-1]

    history_1y = mm_net[mm_net.index >= now - pd.DateOffset(years=1)]
    history_5y = mm_net[mm_net.index >= now - pd.DateOffset(years=5)]

    if len(history_1y) < 10:
        return _unavailable(name, weight, source)

    pct_1y = float(np.sum(history_1y <= raw_value) / len(history_1y) * 100)
    pct_5y = float(np.sum(history_5y <= raw_value) / len(history_5y) * 100) if len(history_5y) >= 10 else pct_1y
    blended = blended_percentile(raw_value, history_1y, history_5y, invert=False)

    return {
        "name": name,
        "raw_value": int(raw_value),
        "percentile_1y": round(pct_1y, 2),
        "percentile_5y": round(pct_5y, 2),
        "blended_percentile": round(blended, 2),
        "normalized_score": round(blended, 2),
        "weight_in_dimension": weight,
        "data_source": source,
        "status": "ok",
    }


def _compute_etf_flows(gld_ohlcv: pd.DataFrame | None) -> dict:
    """Estimate ETF flows via GLD volume/price change as AUM proxy.

    Uses 20-day average volume change as a proxy for ETF inflows/outflows.
    Higher volume with rising price = inflows = bullish.
    """
    name = "ETF Flows (GLD)"
    weight = _WEIGHTS["etf_flows"]
    source = "Yahoo/GLD"

    if gld_ohlcv is None or gld_ohlcv.empty:
        return _unavailable(name, weight, source)

    required_cols = {"close", "volume"}
    if not required_cols.issubset(set(gld_ohlcv.columns)):
        return _unavailable(name, weight, source)

    ohlcv = gld_ohlcv.dropna(subset=["close", "volume"])
    if len(ohlcv) < 60:
        return _unavailable(name, weight, source)

    # AUM proxy = close * volume (price * shares traded)
    # Compute 20-day vs 60-day rolling average AUM as momentum signal
    aum_proxy = ohlcv["close"] * ohlcv["volume"]
    avg_20 = aum_proxy.rolling(20).mean()
    avg_60 = aum_proxy.rolling(60).mean()

    if avg_20.dropna().empty or avg_60.dropna().empty:
        return _unavailable(name, weight, source)

    latest_60 = float(avg_60.iloc[-1])

    if latest_60 == 0:
        return _unavailable(name, weight, source)

    # Compute ratio history for percentile
    ratio_series = (avg_20 / avg_60).dropna()
    raw_value = float(ratio_series.iloc[-1])
    now = ratio_series.index[-1]
    history_1y = ratio_series[ratio_series.index >= now - pd.DateOffset(years=1)]
    history_5y = ratio_series[ratio_series.index >= now - pd.DateOffset(years=5)]

    if len(history_1y) < 20:
        return _unavailable(name, weight, source)

    pct_1y = float(np.sum(history_1y <= raw_value) / len(history_1y) * 100)
    pct_5y = float(np.sum(history_5y <= raw_value) / len(history_5y) * 100) if len(history_5y) >= 20 else pct_1y
    blended = blended_percentile(raw_value, history_1y, history_5y, invert=False)

    return {
        "name": name,
        "raw_value": round(raw_value, 4),
        "percentile_1y": round(pct_1y, 2),
        "percentile_5y": round(pct_5y, 2),
        "blended_percentile": round(blended, 2),
        "normalized_score": round(blended, 2),
        "weight_in_dimension": weight,
        "data_source": source,
        "status": "ok",
    }


def _compute_geo_risk(geo_risk_prob: float | None) -> dict:
    """Score geopolitical risk level from Polymarket.

    Higher risk probability -> more safe-haven demand for gold -> bullish.
    """
    name = "Geopolitical Risk"
    weight = _WEIGHTS["geopolitical_risk"]
    source = "Polymarket"

    if geo_risk_prob is None:
        return _unavailable(name, weight, source)

    score = float(geo_risk_prob) * 100.0
    score = max(0.0, min(100.0, score))

    return {
        "name": name,
        "raw_value": round(float(geo_risk_prob), 4),
        "percentile_1y": None,
        "percentile_5y": None,
        "blended_percentile": score,
        "normalized_score": round(score, 2),
        "weight_in_dimension": weight,
        "data_source": source,
        "status": "ok",
    }


def _unavailable(name: str, weight: float, data_source: str) -> dict:
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
