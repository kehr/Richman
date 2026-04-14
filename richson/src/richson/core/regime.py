"""Market regime detection (TRD SS7 / PRD SS3.3).

Determines market regime (risk-on / neutral / risk-off) from:
- VIX level and trend
- T10Y2Y yield curve spread

Regime classification:
- risk_off: VIX > 25 OR yield curve deeply inverted (T10Y2Y < -0.5%)
- risk_on : VIX < 15 AND yield curve not inverted (T10Y2Y > 0)
- neutral  : everything else

Input: VIX OHLCV DataFrame, T10Y2Y Series.
Output: dict with regime label, VIX level, and reasoning.
"""

from __future__ import annotations

import pandas as pd
import structlog

logger = structlog.get_logger(__name__)

# Regime thresholds
_VIX_HIGH = 25.0      # risk-off threshold
_VIX_LOW = 15.0       # risk-on threshold
_CURVE_INVERTED = -0.50  # deeply inverted yield curve threshold
_CURVE_POSITIVE = 0.0    # positive curve threshold


def detect_regime(
    vix_ohlcv: pd.DataFrame | None,
    t10y2y: pd.Series | None,
) -> dict:
    """Detect current market regime.

    Args:
        vix_ohlcv: VIX daily OHLCV DataFrame from Yahoo Finance.
        t10y2y: FRED T10Y2Y series (daily, percent).

    Returns:
        Dict with:
        - ``regime``: ``risk_on`` | ``neutral`` | ``risk_off``
        - ``vix_level``: float | None -- latest VIX close
        - ``vix_trend``: ``rising`` | ``falling`` | ``stable`` | None
        - ``yield_curve_spread``: float | None -- latest T10Y2Y value
        - ``reasoning``: list[str] -- factors driving the regime call
    """
    vix_level: float | None = None
    vix_trend: str | None = None
    yield_curve_spread: float | None = None
    reasoning: list[str] = []

    # --- VIX ---
    if vix_ohlcv is not None and not vix_ohlcv.empty and "close" in vix_ohlcv.columns:
        vix_close = vix_ohlcv["close"].dropna()
        if not vix_close.empty:
            vix_level = float(vix_close.iloc[-1])
            # Trend: compare 5-day average to 20-day average
            if len(vix_close) >= 20:
                avg5 = float(vix_close.iloc[-5:].mean())
                avg20 = float(vix_close.iloc[-20:].mean())
                if avg5 > avg20 * 1.05:
                    vix_trend = "rising"
                elif avg5 < avg20 * 0.95:
                    vix_trend = "falling"
                else:
                    vix_trend = "stable"

    # --- Yield curve ---
    if t10y2y is not None and not t10y2y.empty:
        yc = t10y2y.dropna()
        if not yc.empty:
            yield_curve_spread = float(yc.iloc[-1])

    # --- Regime determination ---
    is_risk_off = False

    if vix_level is not None:
        if vix_level > _VIX_HIGH:
            is_risk_off = True
            reasoning.append(f"VIX {vix_level:.1f} > {_VIX_HIGH} (elevated fear)")
        elif vix_level < _VIX_LOW:
            reasoning.append(f"VIX {vix_level:.1f} < {_VIX_LOW} (low volatility)")

    if yield_curve_spread is not None:
        if yield_curve_spread < _CURVE_INVERTED:
            is_risk_off = True
            reasoning.append(
                f"Yield curve deeply inverted ({yield_curve_spread:.2f}%)"
            )
        elif yield_curve_spread > _CURVE_POSITIVE:
            reasoning.append(
                f"Yield curve positive ({yield_curve_spread:.2f}%), normal)"
            )

    # Rising VIX even below threshold suggests developing stress
    if vix_trend == "rising" and vix_level and vix_level > 18:
        reasoning.append("VIX trending higher -- watch for risk-off transition")

    if is_risk_off:
        regime = "risk_off"
    elif (
        vix_level is not None
        and vix_level < _VIX_LOW
        and (yield_curve_spread is None or yield_curve_spread > _CURVE_POSITIVE)
    ):
        regime = "risk_on"
    else:
        regime = "neutral"

    if not reasoning:
        reasoning.append("No strong regime signal")

    return {
        "regime": regime,
        "vix_level": round(vix_level, 2) if vix_level is not None else None,
        "vix_trend": vix_trend,
        "yield_curve_spread": round(yield_curve_spread, 4) if yield_curve_spread is not None else None,
        "reasoning": reasoning,
    }
