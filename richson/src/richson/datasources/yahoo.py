"""Yahoo Finance data wrapper.

Provides OHLCV price data and ETF holdings data for US market assets.
Uses yfinance under the hood. Falls back gracefully on network errors.

Primary use cases:
- Gold spot proxies: GLD, IAU, GC=F (futures)
- DXY index: DX-Y.NYB
- VIX: ^VIX
- ETF holdings (GLD top 10 holders)
"""

from __future__ import annotations

import logging
from typing import Any

import pandas as pd

from richson.datasources.cache import cache_get, cache_set

logger = logging.getLogger(__name__)

# How many calendar days of OHLCV history to fetch for percentile calculation
_OHLCV_HISTORY_DAYS = 365 * 5 + 30  # 5 years + buffer


class YahooFinanceClient:
    """Wrapper around yfinance for OHLCV and ETF data.

    Args:
        timeout: HTTP request timeout in seconds.
        max_retries: number of retries on transient errors.
    """

    def __init__(self, timeout: int = 10, max_retries: int = 2) -> None:
        self._timeout = timeout
        self._max_retries = max_retries

    def _download(self, ticker: str, period: str = "5y", interval: str = "1d") -> pd.DataFrame | None:
        """Download OHLCV data for a ticker with caching.

        Args:
            ticker: Yahoo Finance ticker symbol.
            period: lookback period string understood by yfinance (e.g. ``5y``).
            interval: data frequency (``1d``, ``1wk``).

        Returns:
            DataFrame with columns Open/High/Low/Close/Volume, datetime index.
            None on failure.
        """
        cache_key = f"{ticker}:{period}:{interval}"
        cached = cache_get("yahoo_price", cache_key)
        if cached is not None:
            return cached  # type: ignore[return-value]

        for attempt in range(self._max_retries + 1):
            try:
                import yfinance as yf  # noqa: PLC0415

                ticker_obj: Any = yf.Ticker(ticker)
                df: pd.DataFrame = ticker_obj.history(
                    period=period,
                    interval=interval,
                    auto_adjust=True,
                    actions=False,
                )
                if df.empty:
                    logger.warning("yahoo: empty data", ticker=ticker)
                    return None
                # Normalize column names to lowercase
                df.columns = [c.lower() for c in df.columns]
                # Ensure datetime index is timezone-naive
                if hasattr(df.index, "tz") and df.index.tz is not None:
                    df.index = df.index.tz_localize(None)
                cache_set("yahoo_price", cache_key, df)
                return df
            except Exception as exc:
                logger.warning(
                    "yahoo: download failed",
                    ticker=ticker,
                    attempt=attempt,
                    error=str(exc),
                )
        return None

    def get_ohlcv(self, ticker: str, period: str = "5y") -> pd.DataFrame | None:
        """Fetch daily OHLCV data.

        Args:
            ticker: Yahoo Finance symbol (e.g. ``GLD``, ``GC=F``).
            period: yfinance period string.

        Returns:
            DataFrame[open, high, low, close, volume] or None.
        """
        return self._download(ticker, period=period, interval="1d")

    def get_current_price(self, ticker: str) -> float | None:
        """Return the latest closing price for a ticker.

        Args:
            ticker: Yahoo Finance symbol.

        Returns:
            Most recent close price, or None on failure.
        """
        df = self._download(ticker, period="5d", interval="1d")
        if df is None or df.empty:
            return None
        return float(df["close"].iloc[-1])

    def get_etf_info(self, ticker: str) -> dict[str, Any] | None:
        """Fetch ETF metadata including AUM and fund type.

        Args:
            ticker: ETF ticker symbol (e.g. ``GLD``).

        Returns:
            Dict with fund info fields or None on failure.
        """
        cache_key = f"info:{ticker}"
        cached = cache_get("yahoo_etf", cache_key)
        if cached is not None:
            return cached  # type: ignore[return-value]

        for attempt in range(self._max_retries + 1):
            try:
                import yfinance as yf  # noqa: PLC0415

                ticker_obj: Any = yf.Ticker(ticker)
                info: dict[str, Any] = ticker_obj.info
                if not info:
                    return None
                cache_set("yahoo_etf", cache_key, info)
                return info
            except Exception as exc:
                logger.warning(
                    "yahoo: info fetch failed",
                    ticker=ticker,
                    attempt=attempt,
                    error=str(exc),
                )
        return None

    def get_dxy(self) -> pd.DataFrame | None:
        """DXY US Dollar index OHLCV data."""
        return self.get_ohlcv("DX-Y.NYB")

    def get_vix(self) -> pd.DataFrame | None:
        """VIX volatility index OHLCV data."""
        return self.get_ohlcv("^VIX")
