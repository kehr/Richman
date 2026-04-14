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
from dataclasses import dataclass
from typing import Any

import httpx
import pandas as pd
import structlog

from richson.config.event_metadata import FRED_RELEASE_METADATA
from richson.datasources.cache import cache_get, cache_set

logger = structlog.get_logger(__name__)

# Series to fetch in bulk and their cache keys
SERIES_IDS = ["FEDFUNDS", "T10Y2Y", "DFII10", "DGS10", "M2SL"]

# FRED REST endpoint for the release calendar.
_FRED_RELEASES_DATES_URL = "https://api.stlouisfed.org/fred/releases/dates"


@dataclass(frozen=True)
class FREDReleaseDate:
    """A single upcoming FRED release date."""

    release_id: int
    release_name: str
    date: str  # ISO YYYY-MM-DD

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

    def get_upcoming_releases(self, window_days: int = 7) -> list[FREDReleaseDate]:
        """Fetch upcoming FRED release dates within the next N days.

        Returns:
            Sorted list of FREDReleaseDate (ascending by date). Empty list when
            the FRED key is disabled, the network call fails, or no releases
            fall in the window.
        """
        # Disabled short-circuit: skip the network round-trip and the retry
        # loop. The startup warning already told the operator what to fix.
        if self._disabled or not self._api_key:
            return []

        cache_key = f"upcoming_releases:{window_days}"
        cached = cache_get("fred", cache_key)
        if cached is not None:
            return list(cached)

        today = datetime.date.today()
        horizon = today + datetime.timedelta(days=window_days)
        params: dict[str, Any] = {
            "api_key": self._api_key,
            "file_type": "json",
            # The FRED default `false` filters out future dates because they
            # have no observations yet; we must explicitly opt in to keep them.
            "include_release_dates_with_no_data": "true",
            "realtime_start": today.isoformat(),
            "realtime_end": horizon.isoformat(),
            "order_by": "release_date",
            "sort_order": "asc",
            "limit": 1000,
            "offset": 0,
        }

        payload: dict[str, Any] | None = None
        for attempt in range(self._max_retries + 1):
            try:
                with httpx.Client(timeout=self._timeout) as client:
                    resp = client.get(_FRED_RELEASES_DATES_URL, params=params)
                    resp.raise_for_status()
                    payload = resp.json()
                    break
            except Exception as exc:
                logger.warning(
                    "fred releases fetch failed",
                    error=str(exc),
                    attempt=attempt,
                )
        if payload is None:
            return []

        raw_dates: list[dict[str, Any]] = payload.get("release_dates") or []
        seen: set[tuple[int, str]] = set()
        results: list[FREDReleaseDate] = []
        for item in raw_dates:
            try:
                release_id = int(item["release_id"])
            except (KeyError, TypeError, ValueError):
                continue
            if release_id not in FRED_RELEASE_METADATA:
                continue
            release_date = str(item.get("date") or "")
            if not release_date:
                continue
            key = (release_id, release_date)
            if key in seen:
                continue
            seen.add(key)
            results.append(
                FREDReleaseDate(
                    release_id=release_id,
                    release_name=str(item.get("release_name") or ""),
                    date=release_date,
                )
            )

        results.sort(key=lambda r: r.date)
        cache_set("fred", cache_key, results)
        return results

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
