"""D4 Technical Position dimension indicator calculator.

Sub-indicators (default weights, ATR-adjusted per TRD SS7.3):
- SMA cross (price vs SMA50/SMA200)   -- weight 25%
- RSI(14)                              -- weight 20%
- ATR(14) percentile                   -- weight 15%
- Bollinger Band position              -- weight 20%
- MACD signal                          -- weight 20%

All indicators are computed from pure pandas without external TA libraries.
ATR percentile is also used to determine market regime for weight adjustment.

Input: OHLCV DataFrame (daily, minimum 250 days recommended).
Output: dict with per-indicator scores, adjusted weights, and base_score.
"""

from __future__ import annotations

import logging

import numpy as np
import pandas as pd

from richson.core.scoring import blended_percentile, weighted_dimension_score

logger = logging.getLogger(__name__)

# Default sub-indicator weights
_DEFAULT_WEIGHTS = {
    "sma_cross": 0.25,
    "rsi": 0.20,
    "bollinger": 0.20,
    "macd": 0.20,
    "atr": 0.15,
}

_MIN_OHLCV_DAYS = 60


def compute_d4_indicators(ohlcv: pd.DataFrame | None) -> dict:
    """Compute D4 technical position dimension scores.

    Args:
        ohlcv: daily OHLCV DataFrame with columns [open, high, low, close, volume].
               Minimum 60 rows required for basic indicators; 200+ recommended.

    Returns:
        Dict with structure::

            {
                "sub_indicators": [...],
                "base_score": float,
                "available_count": int,
                "total_count": int,
                "atr_regime": "trending" | "ranging" | "normal",
                "effective_weights": dict,
            }
    """
    if ohlcv is None or ohlcv.empty:
        return {
            "sub_indicators": [],
            "base_score": 50.0,  # neutral when no data
            "available_count": 0,
            "total_count": 5,
            "atr_regime": "normal",
            "effective_weights": _DEFAULT_WEIGHTS,
        }

    ohlcv = ohlcv.dropna(subset=["close"]).copy()
    if len(ohlcv) < _MIN_OHLCV_DAYS:
        return {
            "sub_indicators": [],
            "base_score": 50.0,
            "available_count": 0,
            "total_count": 5,
            "atr_regime": "normal",
            "effective_weights": _DEFAULT_WEIGHTS,
        }

    close = ohlcv["close"]
    high = ohlcv["high"] if "high" in ohlcv.columns else close
    low = ohlcv["low"] if "low" in ohlcv.columns else close

    # --- Compute raw indicators ---
    sma50 = _sma(close, 50)
    sma200 = _sma(close, 200)
    rsi14 = _rsi(close, 14)
    atr14 = _atr(high, low, close, 14)
    upper_bb, lower_bb = _bollinger(close, 20, 2.0)
    macd_line, signal_line = _macd(close, 12, 26, 9)

    # --- ATR regime detection (TRD SS7.3) ---
    atr_regime, adjusted_weights = _detect_atr_regime(atr14, close, _DEFAULT_WEIGHTS)

    # --- Score each indicator ---
    sub_indicators = []

    # SMA cross score
    sma_result = _score_sma_cross(close, sma50, sma200, adjusted_weights["sma_cross"])
    sub_indicators.append(sma_result)

    # RSI score
    rsi_result = _score_rsi(rsi14, adjusted_weights["rsi"])
    sub_indicators.append(rsi_result)

    # Bollinger Band position score
    bb_result = _score_bollinger(close, upper_bb, lower_bb, adjusted_weights["bollinger"])
    sub_indicators.append(bb_result)

    # MACD score
    macd_result = _score_macd(macd_line, signal_line, adjusted_weights["macd"])
    sub_indicators.append(macd_result)

    # ATR percentile score
    atr_result = _score_atr(atr14, close, adjusted_weights["atr"])
    sub_indicators.append(atr_result)

    base_score = weighted_dimension_score(sub_indicators)

    return {
        "sub_indicators": sub_indicators,
        "base_score": base_score,
        "available_count": sum(1 for s in sub_indicators if s["status"] == "ok"),
        "total_count": len(sub_indicators),
        "atr_regime": atr_regime,
        "effective_weights": adjusted_weights,
    }


# ---------------------------------------------------------------------------
# Technical indicator computations (pure pandas/numpy)
# ---------------------------------------------------------------------------

def _sma(series: pd.Series, period: int) -> pd.Series:
    return series.rolling(window=period).mean()


def _rsi(series: pd.Series, period: int = 14) -> pd.Series:
    """Compute RSI using Wilder's smoothing method."""
    delta = series.diff()
    gain = delta.clip(lower=0)
    loss = (-delta).clip(lower=0)
    avg_gain = gain.ewm(com=period - 1, adjust=False).mean()
    avg_loss = loss.ewm(com=period - 1, adjust=False).mean()
    rs = avg_gain / avg_loss.replace(0, np.nan)
    return 100.0 - (100.0 / (1.0 + rs))


def _atr(high: pd.Series, low: pd.Series, close: pd.Series, period: int = 14) -> pd.Series:
    """Compute Average True Range."""
    prev_close = close.shift(1)
    tr = pd.concat(
        [
            high - low,
            (high - prev_close).abs(),
            (low - prev_close).abs(),
        ],
        axis=1,
    ).max(axis=1)
    return tr.ewm(com=period - 1, adjust=False).mean()


def _bollinger(series: pd.Series, period: int = 20, num_std: float = 2.0) -> tuple[pd.Series, pd.Series]:
    """Return (upper_band, lower_band)."""
    mid = series.rolling(window=period).mean()
    std = series.rolling(window=period).std()
    return mid + num_std * std, mid - num_std * std


def _macd(series: pd.Series, fast: int = 12, slow: int = 26, signal: int = 9) -> tuple[pd.Series, pd.Series]:
    """Return (macd_line, signal_line)."""
    ema_fast = series.ewm(span=fast, adjust=False).mean()
    ema_slow = series.ewm(span=slow, adjust=False).mean()
    macd_line = ema_fast - ema_slow
    signal_line = macd_line.ewm(span=signal, adjust=False).mean()
    return macd_line, signal_line


# ---------------------------------------------------------------------------
# ATR regime detection and weight adjustment (TRD SS7.3)
# ---------------------------------------------------------------------------

def _detect_atr_regime(
    atr14: pd.Series,
    close: pd.Series,
    base_weights: dict[str, float],
) -> tuple[str, dict[str, float]]:
    """Determine ATR regime and adjust D4 sub-indicator weights.

    ATR percentile (5-year) determines regime:
    - > P75: trending market -> momentum indicators get higher weight
    - < P25: ranging market  -> mean-reversion indicators get higher weight
    - P25-P75: normal        -> default weights

    Args:
        atr14: ATR(14) series.
        close: close price series.
        base_weights: default sub-indicator weights dict.

    Returns:
        Tuple of (regime_str, adjusted_weights_dict).
    """
    atr14 = atr14.dropna()
    if atr14.empty:
        return "normal", dict(base_weights)

    # Normalize ATR by price for comparability across price levels
    atr_pct = (atr14 / close).dropna()
    if atr_pct.empty:
        return "normal", dict(base_weights)

    current_atr_pct = float(atr_pct.iloc[-1])
    now = atr_pct.index[-1]
    history_5y = atr_pct[atr_pct.index >= now - pd.DateOffset(years=5)]

    if len(history_5y) < 90:
        return "normal", dict(base_weights)

    pct_rank = float(np.sum(history_5y <= current_atr_pct) / len(history_5y))

    weights = dict(base_weights)

    if pct_rank > 0.75:
        # Trending market: favor momentum (RSI, SMA cross, MACD/Donchian)
        regime = "trending"
        weights["rsi"] = min(1.0, weights.get("rsi", 0.20) + 0.05)
        weights["sma_cross"] = min(1.0, weights.get("sma_cross", 0.25) + 0.05)
        weights["macd"] = min(1.0, weights.get("macd", 0.20) + 0.05)
        weights["bollinger"] = max(0.0, weights.get("bollinger", 0.20) - 0.10)
        weights["atr"] = max(0.0, weights.get("atr", 0.15) - 0.05)
    elif pct_rank < 0.25:
        # Ranging market: favor mean-reversion (Bollinger, RSI)
        regime = "ranging"
        weights["bollinger"] = min(1.0, weights.get("bollinger", 0.20) + 0.05)
        weights["rsi"] = min(1.0, weights.get("rsi", 0.20) + 0.05)
        weights["sma_cross"] = max(0.0, weights.get("sma_cross", 0.25) - 0.05)
        weights["macd"] = max(0.0, weights.get("macd", 0.20) - 0.05)
    else:
        regime = "normal"

    # Re-normalize to ensure weights sum to 1.0
    total = sum(weights.values())
    if total > 0:
        weights = {k: v / total for k, v in weights.items()}

    return regime, weights


# ---------------------------------------------------------------------------
# Sub-indicator scoring functions
# ---------------------------------------------------------------------------

def _score_sma_cross(
    close: pd.Series,
    sma50: pd.Series,
    sma200: pd.Series,
    weight: float,
) -> dict:
    """Score based on price position relative to SMA50 and SMA200."""
    name = "SMA Cross"
    source = "computed"

    latest_close = close.dropna().iloc[-1] if not close.dropna().empty else None
    latest_sma50 = sma50.dropna().iloc[-1] if not sma50.dropna().empty else None
    latest_sma200 = sma200.dropna().iloc[-1] if not sma200.dropna().empty else None

    if latest_close is None:
        return _unavailable(name, weight, source)

    # Score logic:
    # Price above both SMA50 and SMA200, SMA50 > SMA200 (golden cross): 80-100
    # Price above SMA200 only: 60-79
    # Price between SMA50 and SMA200: 40-59
    # Price below SMA200 only: 20-39
    # Price below both: 0-20
    if latest_sma50 is not None and latest_sma200 is not None:
        above_50 = latest_close > latest_sma50
        above_200 = latest_close > latest_sma200
        golden_cross = latest_sma50 > latest_sma200

        if above_50 and above_200 and golden_cross:
            score = 85.0
        elif above_50 and above_200:
            score = 70.0
        elif above_200:
            score = 55.0
        elif above_50:
            score = 45.0
        else:
            score = 20.0
    elif latest_sma200 is not None:
        score = 70.0 if latest_close > latest_sma200 else 30.0
    else:
        score = 50.0  # neutral if insufficient history

    return {
        "name": name,
        "raw_value": round(float(latest_close), 4),
        "percentile_1y": None,
        "percentile_5y": None,
        "blended_percentile": score,
        "normalized_score": round(score, 2),
        "weight_in_dimension": round(weight, 4),
        "data_source": source,
        "status": "ok",
    }


def _score_rsi(rsi14: pd.Series, weight: float) -> dict:
    """Score RSI(14). Overbought/oversold mapping to 0-100."""
    name = "RSI(14)"
    source = "computed"

    rsi_clean = rsi14.dropna()
    if rsi_clean.empty:
        return _unavailable(name, weight, source)

    raw = float(rsi_clean.iloc[-1])

    # Map RSI to bullish score:
    # RSI 40-60 (neutral): score 50
    # RSI 60-70 (strong): score 70
    # RSI > 70 (overbought - bearish reversal risk): score 30
    # RSI 30-40 (oversold approaching): score 60
    # RSI < 30 (deeply oversold - contrarian buy): score 35
    if raw > 70:
        score = 30.0  # overbought, reversal risk
    elif raw > 60:
        score = 70.0  # bullish momentum
    elif raw > 50:
        score = 58.0
    elif raw > 40:
        score = 45.0
    elif raw > 30:
        score = 60.0  # oversold -> potential bounce
    else:
        score = 35.0  # deeply oversold

    return {
        "name": name,
        "raw_value": round(raw, 2),
        "percentile_1y": None,
        "percentile_5y": None,
        "blended_percentile": score,
        "normalized_score": round(score, 2),
        "weight_in_dimension": round(weight, 4),
        "data_source": source,
        "status": "ok",
    }


def _score_bollinger(
    close: pd.Series,
    upper_bb: pd.Series,
    lower_bb: pd.Series,
    weight: float,
) -> dict:
    """Score price position within Bollinger Bands."""
    name = "Bollinger Band"
    source = "computed"

    close_clean = close.dropna()
    upper_clean = upper_bb.dropna()
    lower_clean = lower_bb.dropna()

    if close_clean.empty or upper_clean.empty or lower_clean.empty:
        return _unavailable(name, weight, source)

    latest_close = float(close_clean.iloc[-1])
    latest_upper = float(upper_clean.iloc[-1])
    latest_lower = float(lower_clean.iloc[-1])

    band_width = latest_upper - latest_lower
    if band_width <= 0:
        return _unavailable(name, weight, source)

    # Position within band: 0 = at lower band, 1 = at upper band
    position = (latest_close - latest_lower) / band_width
    position = max(0.0, min(1.0, position))

    # Bullish interpretation:
    # Near upper band (position 0.75-1.0) with trend = bullish continuation
    # Near lower band (position 0.0-0.25) = oversold / support zone
    # Middle band = neutral
    if position > 0.85:
        score = 75.0  # strong momentum, near upper band
    elif position > 0.65:
        score = 65.0
    elif position > 0.35:
        score = 50.0
    elif position > 0.15:
        score = 55.0  # near support
    else:
        score = 45.0  # at/below lower band, support test

    return {
        "name": name,
        "raw_value": round(position, 4),
        "percentile_1y": None,
        "percentile_5y": None,
        "blended_percentile": score,
        "normalized_score": round(score, 2),
        "weight_in_dimension": round(weight, 4),
        "data_source": source,
        "status": "ok",
    }


def _score_macd(
    macd_line: pd.Series,
    signal_line: pd.Series,
    weight: float,
) -> dict:
    """Score MACD signal crossover."""
    name = "MACD Signal"
    source = "computed"

    macd_clean = macd_line.dropna()
    signal_clean = signal_line.dropna()

    if macd_clean.empty or signal_clean.empty:
        return _unavailable(name, weight, source)

    latest_macd = float(macd_clean.iloc[-1])
    latest_signal = float(signal_clean.iloc[-1])
    histogram = latest_macd - latest_signal

    # Score based on histogram sign and magnitude
    if histogram > 0 and latest_macd > 0:
        score = 75.0  # bullish: MACD above signal, both positive
    elif histogram > 0:
        score = 60.0  # MACD crossing above signal (improving)
    elif histogram < 0 and latest_macd < 0:
        score = 25.0  # bearish: MACD below signal, both negative
    else:
        score = 40.0  # MACD crossing below signal (deteriorating)

    return {
        "name": name,
        "raw_value": round(histogram, 6),
        "percentile_1y": None,
        "percentile_5y": None,
        "blended_percentile": score,
        "normalized_score": round(score, 2),
        "weight_in_dimension": round(weight, 4),
        "data_source": source,
        "status": "ok",
    }


def _score_atr(atr14: pd.Series, close: pd.Series, weight: float) -> dict:
    """Score ATR percentile.

    High ATR relative to history indicates trending/volatile market.
    For gold, high ATR during uptrend is bullish confirmation.
    We use a blended percentile (not inverted) as the score.
    """
    name = "ATR(14) Percentile"
    source = "computed"

    atr_clean = atr14.dropna()
    close_clean = close.dropna()

    if atr_clean.empty or close_clean.empty:
        return _unavailable(name, weight, source)

    # Normalize ATR by price
    atr_pct = (atr_clean / close_clean).dropna()
    if atr_pct.empty:
        return _unavailable(name, weight, source)

    raw_value = float(atr_pct.iloc[-1])
    now = atr_pct.index[-1]
    history_1y = atr_pct[atr_pct.index >= now - pd.DateOffset(years=1)]
    history_5y = atr_pct[atr_pct.index >= now - pd.DateOffset(years=5)]

    if len(history_1y) < 20:
        return _unavailable(name, weight, source)

    pct_1y = float(np.sum(history_1y <= raw_value) / len(history_1y) * 100)
    pct_5y = float(np.sum(history_5y <= raw_value) / len(history_5y) * 100) if len(history_5y) >= 20 else pct_1y
    blended = blended_percentile(raw_value, history_1y, history_5y, invert=False)

    return {
        "name": name,
        "raw_value": round(raw_value, 6),
        "percentile_1y": round(pct_1y, 2),
        "percentile_5y": round(pct_5y, 2),
        "blended_percentile": round(blended, 2),
        "normalized_score": round(blended, 2),
        "weight_in_dimension": round(weight, 4),
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
