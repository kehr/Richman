"""FRED (Federal Reserve Economic Data) API wrapper.

Fetches macro-economic series used by the quant engine:
- FEDFUNDS  : effective federal funds rate
- T10Y2Y    : 10-year minus 2-year treasury spread (yield curve)
- DFII10    : 10-year TIPS yield (real yield)
- DGS10     : 10-year treasury yield (nominal)
- M2SL      : M2 money supply (seasonally adjusted)

All fetch methods return a pandas Series indexed by date (datetime64[ns]).
On error or missing data, methods return None rather than raising.
"""

from __future__ import annotations

import datetime
import re
from typing import Any

import pandas as pd
import structlog

from richson.datasources.cache import cache_get, cache_set

logger = structlog.get_logger(__name__)

# Series to fetch in bulk and their cache keys
SERIES_IDS = ["FEDFUNDS", "T10Y2Y", "DFII10", "DGS10", "M2SL"]

# Lookback window for percentile calculation (5 years + buffer)
_HISTORY_YEARS = 6

# FRED requires a 32-character lowercase alphanumeric API key. Anything else is
# either an empty default or a placeholder the operator never replaced (e.g.
# "...", "change-me"). We detect these early so the scheduler does not waste
# cycles retrying every series against a server that will always reject us.
_FRED_KEY_PATTERN = re.compile(r"^[a-z0-9]{32}$")


def _is_valid_fred_api_key(key: str) -> bool:
    """Return True when the key has the shape FRED's server will accept.

    This is a shape check, not a liveness check — the server is still the
    authority on whether the key is registered. It exists solely to short-
    circuit obvious placeholders at startup.
    """
    return bool(key) and _FRED_KEY_PATTERN.fullmatch(key) is not None


class FREDClient:
    """Wrapper around the fredapi library.

    Args:
        api_key: FRED API key; if empty string, uses fredapi default (env var).
        timeout: HTTP request timeout in seconds.
        max_retries: number of retries on transient errors.
    """

    def __init__(self, api_key: str = "", timeout: int = 10, max_retries: int = 2) -> None:
        self._api_key = api_key
        self._timeout = timeout
        self._max_retries = max_retries
        self._fred: Any = None  # lazy init
        # A non-empty but malformed key (e.g. the default "..." placeholder)
        # will make every FRED call fail with "Bad Request" and flood the logs.
        # Treat shape-invalid keys as "disabled" and log once so the operator
        # sees a single actionable message instead of per-series retries.
        self._disabled: bool = bool(api_key) and not _is_valid_fred_api_key(api_key)
        if self._disabled:
            logger.warning(
                "fred: api key looks like a placeholder, disabling fetches",
                hint=(
                    "Set FRED_API_KEY to a 32-character lowercase alphanumeric "
                    "key (register at https://fred.stlouisfed.org/docs/api/api_key.html)"
                ),
            )

    def _get_client(self) -> Any:
        """Lazily initialise the fredapi.Fred client."""
        if self._fred is None:
            from fredapi import Fred  # noqa: PLC0415

            kwargs: dict[str, Any] = {}
            if self._api_key:
                kwargs["api_key"] = self._api_key
            self._fred = Fred(**kwargs)
        return self._fred

    def _fetch_series(self, series_id: str) -> pd.Series | None:
        """Fetch a single FRED series with caching and retry.

        Args:
            series_id: FRED series identifier.

        Returns:
            Pandas Series indexed by date, or None on failure.
        """
        cached = cache_get("fred", series_id)
        if cached is not None:
            return cached  # type: ignore[return-value]

        # If the key is obviously malformed, skip the network round-trip and
        # the retry loop. The one-time startup warning already told the
        # operator what to fix; spamming "fred fetch failed" per series for
        # every scheduled run buys nothing.
        if self._disabled:
            return None

        end_date = datetime.date.today()
        start_date = end_date - datetime.timedelta(days=_HISTORY_YEARS * 365)

        for attempt in range(self._max_retries + 1):
            try:
                client = self._get_client()
                series: pd.Series = client.get_series(
                    series_id,
                    observation_start=start_date.isoformat(),
                    observation_end=end_date.isoformat(),
                )
                series = series.dropna()
                if series.empty:
                    logger.warning("fred: empty series", series=series_id)
                    return None
                cache_set("fred", series_id, series)
                return series
            except Exception as exc:
                logger.warning(
                    "fred: fetch failed",
                    series=series_id,
                    attempt=attempt,
                    error=str(exc),
                )
        return None

    def get_fed_funds_rate(self) -> pd.Series | None:
        """Effective federal funds rate (FEDFUNDS)."""
        return self._fetch_series("FEDFUNDS")

    def get_yield_curve(self) -> pd.Series | None:
        """10Y minus 2Y treasury spread (T10Y2Y). Negative = inverted."""
        return self._fetch_series("T10Y2Y")

    def get_real_yield_10y(self) -> pd.Series | None:
        """10-year TIPS yield / real yield (DFII10)."""
        return self._fetch_series("DFII10")

    def get_nominal_yield_10y(self) -> pd.Series | None:
        """10-year treasury nominal yield (DGS10)."""
        return self._fetch_series("DGS10")

    def get_m2(self) -> pd.Series | None:
        """M2 money supply seasonally adjusted (M2SL), in billions USD."""
        return self._fetch_series("M2SL")

    def get_all_series(self) -> dict[str, pd.Series]:
        """Fetch all required FRED series in one call.

        Returns:
            Dict mapping series_id to pd.Series. Missing series are omitted.
        """
        result: dict[str, pd.Series] = {}
        for sid in SERIES_IDS:
            s = self._fetch_series(sid)
            if s is not None:
                result[sid] = s
        return result

    def get_data_freshness(self, series_id: str) -> datetime.date | None:
        """Return the date of the most recent observation for a series.

        Used by confidence.py to check whether FRED data is stale (> 3 days).
        """
        series = self._fetch_series(series_id)
        if series is None or series.empty:
            return None
        last_index = series.index[-1]
        if isinstance(last_index, pd.Timestamp):
            return last_index.date()
        return None  # type: ignore[return-value]
