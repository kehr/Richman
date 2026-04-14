"""AKShare wrapper for A-share market data.

Provides daily OHLCV data for A-share ETFs (e.g. 518880 gold ETF).
AKShare has a large API surface; only the subset needed by richson is wrapped.

Key symbols:
- 518880: Huaan Gold ETF (A-share, trading on Shanghai Exchange)
- 159934: Bosera Gold ETF (Shenzhen Exchange)
"""

from __future__ import annotations

import pandas as pd
import structlog

from richson.datasources.cache import cache_get, cache_set

logger = structlog.get_logger(__name__)

_HISTORY_YEARS = 5


class AKShareClient:
    """AKShare data wrapper for A-share ETF price data.

    Args:
        timeout: implied via akshare internals; passed as hint only.
        max_retries: retries on transient errors.
    """

    def __init__(self, timeout: int = 10, max_retries: int = 2) -> None:
        self._timeout = timeout
        self._max_retries = max_retries

    def get_etf_ohlcv(self, code: str, years: int = _HISTORY_YEARS) -> pd.DataFrame | None:
        """Fetch daily OHLCV for an A-share ETF.

        Args:
            code: 6-digit A-share ETF code (e.g. ``518880``).
            years: history length in years.

        Returns:
            DataFrame[open, high, low, close, volume] sorted ascending by date,
            or None on failure.
        """
        import datetime

        cache_key = f"etf:{code}:{years}y"
        cached = cache_get("akshare", cache_key)
        if cached is not None:
            return cached  # type: ignore[return-value]

        end = datetime.date.today()
        start = end - datetime.timedelta(days=years * 365 + 30)

        for attempt in range(self._max_retries + 1):
            try:
                import akshare as ak  # noqa: PLC0415

                df: pd.DataFrame = ak.fund_etf_hist_em(
                    symbol=code,
                    period="daily",
                    start_date=start.strftime("%Y%m%d"),
                    end_date=end.strftime("%Y%m%d"),
                    adjust="qfq",  # forward adjusted
                )
                if df is None or df.empty:
                    logger.warning("akshare: empty data", code=code)
                    return None

                # Normalize: AKShare returns columns like '日期','开盘','收盘','最高','最低','成交量'
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
                    logger.warning("akshare: missing date column", code=code, columns=list(df.columns))
                    return None

                df["date"] = pd.to_datetime(df["date"])
                df = df.set_index("date").sort_index()

                # Retain only standard OHLCV columns
                standard_cols = [c for c in ["open", "high", "low", "close", "volume"] if c in df.columns]
                df = df[standard_cols]

                cache_set("akshare", cache_key, df)
                return df
            except Exception as exc:
                logger.warning(
                    "akshare: fetch failed",
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
