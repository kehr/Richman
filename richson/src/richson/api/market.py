"""Market data endpoints (Mode C, synchronous light queries).

GET /market/regime          - current macro regime
GET /market/ohlcv/{code}   - OHLCV candlestick data
"""

from __future__ import annotations

import asyncio
from datetime import UTC, datetime
from typing import Annotated, Literal

import structlog
from fastapi import APIRouter, Depends, HTTPException, Query

from richson.api.auth import require_api_key

router = APIRouter(prefix="/market", dependencies=[Depends(require_api_key)])
logger = structlog.get_logger()

PeriodLiteral = Literal["1D", "1W", "1M", "3M", "1Y"]

_PERIOD_TO_DAYS: dict[str, int] = {
    "1D": 1,
    "1W": 7,
    "1M": 30,
    "3M": 90,
    "1Y": 365,
}

_REGIME_LABELS: dict[str, str] = {
    "risk_on": "风险偏好",
    "neutral": "中性观望",
    "risk_off": "风险规避",
}

# Major indices to include in regime response
_INDEX_CODES = [
    ("S&P 500", "^GSPC"),
    ("Nasdaq", "^IXIC"),
    ("Shanghai Composite", "000001.SS"),
    ("Gold", "GC=F"),
]


@router.get("/regime")
async def get_market_regime() -> dict:
    """Return current market regime (risk_on / neutral / risk_off).

    Fetches VIX and T10Y2Y from external data sources. Results cached internally.
    """
    from richson.config import settings  # noqa: PLC0415
    from richson.core.regime import detect_regime  # noqa: PLC0415
    from richson.datasources.fred import FREDClient  # noqa: PLC0415
    from richson.datasources.yahoo import YahooFinanceClient  # noqa: PLC0415

    fred_client = FREDClient(api_key=settings.fred_api_key)
    yahoo_client = YahooFinanceClient()

    def _fetch() -> tuple:
        all_fred = fred_client.get_all_series()
        t10y2y = all_fred.get("T10Y2Y")
        bamlc = all_fred.get("BAMLC0A0CM")  # may not be in default SERIES_IDS; fallback None
        if bamlc is None:
            # BAMLC0A0CM not in standard SERIES_IDS; fetch directly via private method
            try:
                bamlc = fred_client._fetch_series("BAMLC0A0CM")
            except Exception:
                bamlc = None
        vix_df = yahoo_client.get_vix()
        return t10y2y, bamlc, vix_df

    t10y2y, bamlc, vix_df = await asyncio.to_thread(_fetch)

    regime_data = detect_regime(vix_df, t10y2y)

    vix_level = regime_data.get("vix_level")
    t10y2y_val = regime_data.get("yield_curve_spread")
    credit_spread = None
    if bamlc is not None and not bamlc.empty:
        credit_spread = float(bamlc.dropna().iloc[-1])

    regime = regime_data.get("regime", "neutral")
    regime_label = _REGIME_LABELS.get(regime, "中性观望")

    # Build one-sentence reason from reasoning list
    reason_parts = regime_data.get("reasoning", [])
    reason = "; ".join(reason_parts[:2]) if reason_parts else "No strong regime signal"

    # Fetch index snapshots
    def _fetch_indices() -> list[dict]:
        indices = []
        for name, code in _INDEX_CODES:
            try:
                df = yahoo_client.get_ohlcv(code, period="5d")
                if df is not None and not df.empty:
                    close_col = "Close" if "Close" in df.columns else "close"
                    if close_col in df.columns:
                        closes = df[close_col].dropna()
                        if len(closes) >= 2:
                            price = float(closes.iloc[-1])
                            prev = float(closes.iloc[-2])
                            change_pct = round((price - prev) / prev * 100, 2)
                        elif len(closes) == 1:
                            price = float(closes.iloc[-1])
                            change_pct = 0.0
                        else:
                            continue
                        indices.append({
                            "name": name,
                            "code": code,
                            "price": round(price, 2),
                            "changePercent": change_pct,
                        })
            except Exception:
                pass
        return indices

    indices = await asyncio.to_thread(_fetch_indices)

    return {
        "data": {
            "regime": regime,
            "regimeLabel": regime_label,
            "reason": reason,
            "vix": round(vix_level, 2) if vix_level is not None else None,
            "t10y2y": round(t10y2y_val, 4) if t10y2y_val is not None else None,
            "creditSpread": round(credit_spread, 4) if credit_spread is not None else None,
            "indices": indices,
            "updatedAt": datetime.now(tz=UTC).isoformat(),
        }
    }


@router.get("/ohlcv/{asset_code}")
async def get_ohlcv(
    asset_code: str,
    period: Annotated[PeriodLiteral, Query()] = "3M",
) -> dict:
    """Return OHLCV candlestick data for an asset.

    Supports period query param: 1D | 1W | 1M | 3M | 1Y.
    """
    from richson.core.support_resistance import compute_support_resistance  # noqa: PLC0415
    from richson.datasources.stooq import StooqClient  # noqa: PLC0415
    from richson.datasources.yahoo import YahooFinanceClient  # noqa: PLC0415

    yahoo_client = YahooFinanceClient()
    stooq_client = StooqClient()
    days = _PERIOD_TO_DAYS.get(period, 90)

    def _fetch() -> object:
        df = yahoo_client.get_ohlcv(asset_code)
        if df is None or (hasattr(df, "empty") and df.empty):
            df = stooq_client.get_ohlcv(asset_code)
        return df

    ohlcv = await asyncio.to_thread(_fetch)

    if ohlcv is None or (hasattr(ohlcv, "empty") and ohlcv.empty):
        raise HTTPException(status_code=502, detail={
            "error": {
                "code": "DATA_SOURCE_UNAVAILABLE",
                "message": f"OHLCV data not available for {asset_code}",
                "details": [],
            }
        })

    import pandas as pd  # noqa: PLC0415

    df = ohlcv.copy()
    df.columns = [c.lower() for c in df.columns]

    # Filter to requested period
    if len(df) > 0:
        cutoff = df.index[-1] - pd.Timedelta(days=days)
        df_period = df[df.index >= cutoff]
    else:
        df_period = df

    # Build candles
    candles = []
    for ts, row in df_period.iterrows():
        date_str = ts.strftime("%Y-%m-%d") if hasattr(ts, "strftime") else str(ts)[:10]
        candles.append({
            "date": date_str,
            "open": float(row.get("open", 0)),
            "high": float(row.get("high", 0)),
            "low": float(row.get("low", 0)),
            "close": float(row.get("close", 0)),
            "volume": float(row.get("volume", 0)),
        })

    # SMA 200
    close_series = df["close"].dropna()
    sma200 = None
    if len(close_series) >= 200:
        sma200 = round(float(close_series.rolling(200).mean().iloc[-1]), 2)

    # Support/resistance
    support_levels: list[float] = []
    resistance_levels: list[float] = []
    if sma200 is not None:
        try:
            sr_result = compute_support_resistance(df, sma200)
            support_levels = sr_result.get("support_levels", [])
            resistance_levels = sr_result.get("resistance_levels", [])
        except Exception:
            pass

    # Determine currency
    currency = "CNY" if any(c in asset_code for c in ["518880", ".SS", ".SZ", ".SH"]) else "USD"

    return {
        "data": {
            "assetCode": asset_code,
            "currency": currency,
            "period": period,
            "candles": candles,
            "sma200": sma200,
            "supportLevels": support_levels,
            "resistanceLevels": resistance_levels,
        }
    }
