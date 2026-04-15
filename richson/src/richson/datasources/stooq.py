"""Stooq fallback data source for price data.

Used when Yahoo Finance is unavailable. Fetches daily OHLCV via the
Stooq CSV endpoint which does not require authentication.

Stooq URL format:
    https://stooq.com/q/d/l/?s={symbol}&d1={start}&d2={end}&i=d
    where dates are formatted as YYYYMMDD.

Typical usage: GLD (maps to GLD.US on Stooq).
"""

from __future__ import annotations

import io

import httpx
import pandas as pd
import structlog

from richson.datasources.cache import cache_get, cache_set

logger = structlog.get_logger(__name__)

_STOOQ_BASE = "https://stooq.com/q/d/l/"
_HISTORY_YEARS = 5


def _to_stooq_symbol(yahoo_ticker: str) -> str:
    """Map a Yahoo Finance ticker to its Stooq equivalent.

    Handles common cases only; caller may pass a Stooq symbol directly.
    """
    mapping = {
        "GLD": "gld.us",
        "IAU": "iau.us",
        "GC=F": "gc.f",
        "DX-Y.NYB": "dxy",
        "^VIX": "vix",
        "^GSPC": "^spx",
    }
    return mapping.get(yahoo_ticker, yahoo_ticker.lower().replace("^", ""))


class StooqClient:
    """Stooq fallback price source.

    Args:
        timeout: HTTP request timeout in seconds.
        max_retries: retries on transient errors.
    """

    def __init__(self, timeout: int = 10, max_retries: int = 2) -> None:
        self._timeout = timeout
        self._max_retries = max_retries

    def get_ohlcv(self, ticker: str, years: int = _HISTORY_YEARS) -> pd.DataFrame | None:
        """Fetch daily OHLCV from Stooq.

        Args:
            ticker: Yahoo Finance ticker or Stooq symbol.
            years: number of years of history to fetch.

        Returns:
            DataFrame[open, high, low, close, volume] sorted ascending by date,
            or None on failure.
        """
        stooq_sym = _to_stooq_symbol(ticker)
        cache_key = f"{stooq_sym}:{years}y"
        cached = cache_get("yahoo_price", cache_key)  # reuse yahoo_price cache TTL
        if cached is not None:
            return cached  # type: ignore[return-value]

        import datetime

        end = datetime.date.today()
        start = end - datetime.timedelta(days=years * 365 + 30)
        params = {
            "s": stooq_sym,
            "d1": start.strftime("%Y%m%d"),
            "d2": end.strftime("%Y%m%d"),
            "i": "d",
        }

        for attempt in range(self._max_retries + 1):
            try:
                with httpx.Client(timeout=self._timeout) as client:
                    resp = client.get(_STOOQ_BASE, params=params)
                    resp.raise_for_status()
                    content = resp.text
                    # Stooq returns 200 OK with a non-CSV body (HTML or
                    # plain "No data") for unknown symbols. Treat any such
                    # response as a permanent failure: retrying just burns
                    # 1.4s per attempt without changing the answer.
                    if not content.startswith("Date,"):
                        logger.warning(
                            "stooq: non-csv response, treating as no data",
                            ticker=ticker,
                            stooq_sym=stooq_sym,
                            head=content[:120],
                        )
                        return None
                    df = pd.read_csv(io.StringIO(content))
                    if df.empty:
                        return None
                    # Normalize column names
                    df.columns = [c.lower() for c in df.columns]
                    df["date"] = pd.to_datetime(df["date"])
                    df = df.set_index("date").sort_index()
                    df = df.rename(columns={"vol": "volume"})
                    cache_set("yahoo_price", cache_key, df)
                    return df
            except Exception as exc:
                logger.warning(
                    "stooq: fetch failed",
                    ticker=ticker,
                    attempt=attempt,
                    error=str(exc),
                )
        return None

    def get_current_price(self, ticker: str) -> float | None:
        """Return the latest closing price.

        Args:
            ticker: Yahoo or Stooq symbol.

        Returns:
            Most recent close price, or None on failure.
        """
        df = self.get_ohlcv(ticker, years=1)
        if df is None or df.empty:
            return None
        return float(df["close"].iloc[-1])
