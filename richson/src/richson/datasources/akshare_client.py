"""AKShare wrapper for A-share market data.

Provides daily OHLCV data for A-share ETFs (e.g. 518880 gold ETF).
AKShare has a large API surface; only the subset needed by richson is wrapped.

Key symbols:
- 518880: Huaan Gold ETF (A-share, trading on Shanghai Exchange)
- 159934: Bosera Gold ETF (Shenzhen Exchange)

Source strategy: sina is the primary source because eastmoney's push2his
endpoint aggressively geo-filters non-mainland egress IPs (TCP connect
succeeds then the server closes without a response). Eastmoney is kept as a
fallback so that if sina is down for maintenance the scheduler can still
pull qfq-adjusted data.
"""

from __future__ import annotations

import datetime as _dt

import pandas as pd
import structlog

from richson.datasources.cache import cache_get, cache_set

logger = structlog.get_logger(__name__)

_HISTORY_YEARS = 5


def _sina_symbol(code: str) -> str:
    """Convert a 6-digit A-share ETF code into sina's exchange-prefixed form.

    Shanghai-listed ETFs use codes starting with 5; Shenzhen-listed ETFs use
    codes starting with 1. Any other prefix is a coding error and falls back
    to sh, matching the most common case for mainland index products.
    """

    head = code[:1]
    if head == "1":
        return f"sz{code}"
    return f"sh{code}"


class AKShareClient:
    """AKShare data wrapper for A-share ETF price data.

    Args:
        timeout: implied via akshare internals; passed as hint only.
        max_retries: retries on transient errors within each source.
    """

    def __init__(self, timeout: int = 10, max_retries: int = 2) -> None:
        self._timeout = timeout
        self._max_retries = max_retries

    def get_etf_ohlcv(self, code: str, years: int = _HISTORY_YEARS) -> pd.DataFrame | None:
        """Fetch daily OHLCV for an A-share ETF.

        Tries sina first, then eastmoney. The first source that returns a
        non-empty frame wins and is cached; downstream callers see a uniform
        DataFrame[open, high, low, close, volume] indexed by date.

        Args:
            code: 6-digit A-share ETF code (e.g. ``518880``).
            years: history length in years.

        Returns:
            DataFrame sorted ascending by date, or None when both sources fail.
        """
        cache_key = f"etf:{code}:{years}y"
        cached = cache_get("akshare", cache_key)
        if cached is not None:
            return cached

        end = _dt.date.today()
        start = end - _dt.timedelta(days=years * 365 + 30)

        df = self._fetch_from_sina(code, start, end)
        if df is None:
            df = self._fetch_from_eastmoney(code, start, end)
        if df is None:
            return None

        cache_set("akshare", cache_key, df)
        return df

    def _fetch_from_sina(
        self,
        code: str,
        start: _dt.date,
        end: _dt.date,
    ) -> pd.DataFrame | None:
        """Pull from sina (primary). Sina returns full history with no
        start/end params, so trim to the requested window after fetch.
        """

        symbol = _sina_symbol(code)
        for attempt in range(self._max_retries + 1):
            try:
                import akshare as ak  # noqa: PLC0415

                df: pd.DataFrame = ak.fund_etf_hist_sina(symbol=symbol)
                if df is None or df.empty:
                    logger.warning("akshare sina: empty data", code=code, symbol=symbol)
                    return None

                df["date"] = pd.to_datetime(df["date"])
                df = df.set_index("date").sort_index()

                start_ts = pd.Timestamp(start)
                end_ts = pd.Timestamp(end)
                df = df[(df.index >= start_ts) & (df.index <= end_ts)]

                standard_cols = [c for c in ["open", "high", "low", "close", "volume"] if c in df.columns]
                df = df[standard_cols]

                if df.empty:
                    logger.warning("akshare sina: no rows in window", code=code, symbol=symbol)
                    return None
                return df
            except Exception as exc:
                logger.warning(
                    "akshare sina: fetch failed",
                    code=code,
                    symbol=symbol,
                    attempt=attempt,
                    error=str(exc),
                )
        return None

    def _fetch_from_eastmoney(
        self,
        code: str,
        start: _dt.date,
        end: _dt.date,
    ) -> pd.DataFrame | None:
        """Pull from eastmoney (fallback). Supports qfq adjustment natively
        and accepts start/end in the request, so no post-trim is needed.
        """

        start_str = start.strftime("%Y%m%d")
        end_str = end.strftime("%Y%m%d")

        for attempt in range(self._max_retries + 1):
            try:
                import akshare as ak  # noqa: PLC0415

                df: pd.DataFrame = ak.fund_etf_hist_em(
                    symbol=code,
                    period="daily",
                    start_date=start_str,
                    end_date=end_str,
                    adjust="qfq",  # forward adjusted
                )
                if df is None or df.empty:
                    logger.warning("akshare eastmoney: empty data", code=code)
                    return None

                # Normalize: eastmoney returns columns like '日期','开盘','收盘','最高','最低','成交量'.
                col_map: dict[str, str] = {}
                for col in df.columns:
                    col_lower = col.lower()
                    if "日期" in col or "date" in col_lower:
                        col_map[col] = "date"
                    elif "开盘" in col or "open" in col_lower:
                        col_map[col] = "open"
                    elif "收盘" in col or "close" in col_lower:
                        col_map[col] = "close"
                    elif "最高" in col or "high" in col_lower:
                        col_map[col] = "high"
                    elif "最低" in col or "low" in col_lower:
                        col_map[col] = "low"
                    elif "成交量" in col or "volume" in col_lower:
                        col_map[col] = "volume"
                df = df.rename(columns=col_map)

                if "date" not in df.columns:
                    logger.warning(
                        "akshare eastmoney: missing date column",
                        code=code,
                        columns=list(df.columns),
                    )
                    return None

                df["date"] = pd.to_datetime(df["date"])
                df = df.set_index("date").sort_index()

                standard_cols = [c for c in ["open", "high", "low", "close", "volume"] if c in df.columns]
                df = df[standard_cols]
                return df
            except Exception as exc:
                logger.warning(
                    "akshare eastmoney: fetch failed",
                    code=code,
                    attempt=attempt,
                    error=str(exc),
                )
        return None

    def get_current_price(self, code: str) -> float | None:
        """Return latest closing price for an A-share ETF.

        Args:
            code: 6-digit ETF code.

        Returns:
            Most recent close price, or None on failure.
        """
        df = self.get_etf_ohlcv(code, years=1)
        if df is None or df.empty:
            return None
        return float(df["close"].iloc[-1])
