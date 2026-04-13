"""Support and resistance level computation (TRD SS7.6).

Computes price support and resistance levels from OHLCV data using:
- 20-day Donchian channel (upper/lower bands)
- Most recent significant high/low in past 60 days (price rebounded > 3%)
- 200-day SMA
- All-time high (within available data)

Returns the levels closest to the current price for each category.

Input: OHLCV DataFrame.
Output: dict with support_levels and resistance_levels lists.
"""

from __future__ import annotations

import numpy as np
import pandas as pd


def compute_support_resistance(
    ohlcv: pd.DataFrame,
    sma200: float | None = None,
) -> dict:
    """Compute support and resistance price levels (TRD SS7.6).

    Args:
        ohlcv: daily OHLCV DataFrame with at least 60 rows.
               Expected columns: open, high, low, close.
        sma200: pre-computed 200-day SMA of close price. If None, computed
                from ohlcv if sufficient data is available.

    Returns:
        Dict with:
        - ``support_levels`` (list[float]): sorted ascending by price
        - ``resistance_levels`` (list[float]): sorted ascending by price
        - ``current_price`` (float): latest close
        - ``nearest_support`` (float | None): support level closest to current price
        - ``nearest_resistance`` (float | None): resistance level closest to current price
    """
    if ohlcv is None or len(ohlcv) < 20:
        return {
            "support_levels": [],
            "resistance_levels": [],
            "current_price": None,
            "nearest_support": None,
            "nearest_resistance": None,
        }

    close = ohlcv["close"].dropna()
    high = ohlcv["high"].dropna() if "high" in ohlcv.columns else close
    low = ohlcv["low"].dropna() if "low" in ohlcv.columns else close

    if close.empty:
        return {
            "support_levels": [],
            "resistance_levels": [],
            "current_price": None,
            "nearest_support": None,
            "nearest_resistance": None,
        }

    current_price = float(close.iloc[-1])
    support_levels: list[float] = []
    resistance_levels: list[float] = []

    # --- Donchian Channel (20-day) ---
    if len(low) >= 20:
        donchian_lower = float(low.rolling(20).min().iloc[-1])
        donchian_upper = float(high.rolling(20).max().iloc[-1])
        support_levels.append(donchian_lower)
        resistance_levels.append(donchian_upper)

    # --- Most recent significant low/high in past 60 days ---
    if len(ohlcv) >= 60:
        recent_60 = ohlcv.iloc[-60:]
        sig_low = _find_significant_low(recent_60["low"] if "low" in recent_60.columns else recent_60["close"])
        sig_high = _find_significant_high(recent_60["high"] if "high" in recent_60.columns else recent_60["close"])
        if sig_low is not None:
            support_levels.append(sig_low)
        if sig_high is not None:
            resistance_levels.append(sig_high)

    # --- 200-day SMA ---
    if sma200 is not None:
        effective_sma200 = sma200
    elif len(close) >= 200:
        effective_sma200 = float(close.rolling(200).mean().iloc[-1])
    else:
        effective_sma200 = None

    if effective_sma200 is not None:
        if effective_sma200 < current_price:
            support_levels.append(effective_sma200)
        else:
            resistance_levels.append(effective_sma200)

    # --- All-time high within available data ---
    all_time_high = float(high.max())
    if all_time_high > current_price:
        resistance_levels.append(all_time_high)

    # Deduplicate and sort
    support_levels = sorted(set(round(v, 2) for v in support_levels if v < current_price))
    resistance_levels = sorted(set(round(v, 2) for v in resistance_levels if v > current_price))

    nearest_support = max(support_levels) if support_levels else None
    nearest_resistance = min(resistance_levels) if resistance_levels else None

    return {
        "support_levels": support_levels,
        "resistance_levels": resistance_levels,
        "current_price": round(current_price, 2),
        "nearest_support": nearest_support,
        "nearest_resistance": nearest_resistance,
    }


def _find_significant_low(low: pd.Series, rebound_threshold: float = 0.03) -> float | None:
    """Find the most recent local low where price subsequently rebounded by >= threshold.

    A significant low is a bar whose low is lower than the surrounding bars
    AND from which price rebounded by at least ``rebound_threshold`` (3% default).

    Args:
        low: series of low prices.
        rebound_threshold: minimum subsequent rebound required (e.g. 0.03 = 3%).

    Returns:
        Most recent significant low price, or None if not found.
    """
    low_arr = low.values.astype(float)
    n = len(low_arr)
    if n < 5:
        return None

    significant_lows = []
    for i in range(2, n - 2):
        if low_arr[i] <= low_arr[i - 1] and low_arr[i] <= low_arr[i + 1]:
            # Local low: check for rebound
            future_high = float(np.max(low_arr[i + 1 :]))
            if low_arr[i] > 0 and (future_high - low_arr[i]) / low_arr[i] >= rebound_threshold:
                significant_lows.append((i, float(low_arr[i])))

    if not significant_lows:
        return None
    # Return the most recent (highest index)
    return significant_lows[-1][1]


def _find_significant_high(high: pd.Series, pullback_threshold: float = 0.03) -> float | None:
    """Find the most recent local high from which price pulled back by >= threshold.

    Args:
        high: series of high prices.
        pullback_threshold: minimum subsequent pullback required.

    Returns:
        Most recent significant high price, or None if not found.
    """
    high_arr = high.values.astype(float)
    n = len(high_arr)
    if n < 5:
        return None

    significant_highs = []
    for i in range(2, n - 2):
        if high_arr[i] >= high_arr[i - 1] and high_arr[i] >= high_arr[i + 1]:
            future_low = float(np.min(high_arr[i + 1 :]))
            if high_arr[i] > 0 and (high_arr[i] - future_low) / high_arr[i] >= pullback_threshold:
                significant_highs.append((i, float(high_arr[i])))

    if not significant_highs:
        return None
    return significant_highs[-1][1]
