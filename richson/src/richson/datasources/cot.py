"""CFTC Commitments of Traders (COT) data wrapper.

Fetches managed money (speculative) net positions for gold futures.
COT data is published weekly (Fridays) by the CFTC.

Data source options:
1. CFTC public download: https://www.cftc.gov/dea/options/financial_lof.htm
2. Quandl/Nasdaq Data Link (legacy CFTC series)
3. Direct CFTC file download (CSV)

We use the CFTC public CSV download as primary source with a 24-hour cache.

COT Contract Name for COMEX Gold: GOLD - COMMODITY EXCHANGE INC.
"""

from __future__ import annotations

import io
import zipfile
from typing import Any

import httpx
import pandas as pd
import structlog

from richson.datasources.cache import cache_get, cache_set

logger = structlog.get_logger(__name__)

# CFTC combined futures-and-options CSV (current year)
_CFTC_COMBINED_URL = (
    "https://www.cftc.gov/files/dea/history/com_disagg_futures_only_2024.zip"
)
# Fallback: disaggregated legacy URL pattern
_CFTC_LEGACY_URL = "https://www.cftc.gov/dea/history/fut_disagg_txt_2024.zip"

_GOLD_CONTRACT_NAME = "GOLD - COMMODITY EXCHANGE INC."
_HISTORY_WEEKS = 104  # 2 years of weekly data


class COTClient:
    """CFTC COT data wrapper for gold futures managed money positions.

    Args:
        timeout: HTTP request timeout in seconds.
        max_retries: retries on transient errors.
    """

    def __init__(self, timeout: int = 30, max_retries: int = 2) -> None:
        self._timeout = timeout
        self._max_retries = max_retries

    def _download_cftc_csv(self, url: str) -> pd.DataFrame | None:
        """Download and parse a CFTC ZIP file containing CSV data.

        Args:
            url: CFTC ZIP download URL.

        Returns:
            Parsed DataFrame or None on failure.
        """
        for attempt in range(self._max_retries + 1):
            try:
                with httpx.Client(timeout=self._timeout) as client:
                    resp = client.get(url)
                    resp.raise_for_status()
                    z = zipfile.ZipFile(io.BytesIO(resp.content))
                    # CSV file is usually the first file in the archive
                    csv_name = [n for n in z.namelist() if n.endswith(".csv")]
                    if not csv_name:
                        csv_name = z.namelist()
                    df = pd.read_csv(
                        io.StringIO(z.read(csv_name[0]).decode("utf-8", errors="replace")),
                        low_memory=False,
                    )
                    return df
            except Exception as exc:
                logger.warning(
                    "cot: download failed",
                    url=url,
                    attempt=attempt,
                    error=str(exc),
                )
        return None

    def get_gold_managed_money(self) -> pd.DataFrame | None:
        """Fetch weekly gold futures managed money net positions.

        Returns:
            DataFrame with columns:
            - date (index, datetime)
            - mm_long: managed money long contracts
            - mm_short: managed money short contracts
            - mm_net: net (long - short)
            - open_interest: total open interest
            Sorted ascending by date. None on failure.
        """
        cache_key = "gold_managed_money"
        cached = cache_get("cot", cache_key)
        if cached is not None:
            return cached  # type: ignore[return-value]

        import datetime

        # Try multiple years (current + prior)
        year = datetime.date.today().year
        urls = [
            f"https://www.cftc.gov/files/dea/history/fut_disagg_txt_{year}.zip",
            f"https://www.cftc.gov/files/dea/history/fut_disagg_txt_{year - 1}.zip",
        ]

        frames: list[pd.DataFrame] = []
        for url in urls:
            df_raw = self._download_cftc_csv(url)
            if df_raw is not None:
                frames.append(df_raw)

        if not frames:
            logger.warning("cot: all CFTC downloads failed")
            return None

        combined = pd.concat(frames, ignore_index=True)

        # Filter for gold contract
        name_col = next(
            (c for c in combined.columns if "market_and_exchange_names" in c.lower() or "name" in c.lower()),
            None,
        )
        date_col = next(
            (c for c in combined.columns if "report_date" in c.lower() or "date" in c.lower()),
            None,
        )
        if name_col is None or date_col is None:
            logger.warning("cot: unexpected column names", columns=list(combined.columns[:10]))
            return None

        gold = combined[combined[name_col].str.upper().str.contains("GOLD", na=False)].copy()
        if gold.empty:
            logger.warning("cot: no gold rows found")
            return None

        # Extract managed money columns (disaggregated COT format)
        def _find_col(df: pd.DataFrame, keywords: list[str]) -> Any:
            for col in df.columns:
                if all(kw.lower() in col.lower() for kw in keywords):
                    return col
            return None

        mm_long_col = _find_col(gold, ["money", "long"])
        mm_short_col = _find_col(gold, ["money", "short"])
        oi_col = _find_col(gold, ["open", "interest"])

        if mm_long_col is None or mm_short_col is None:
            logger.warning("cot: managed money columns not found", columns=list(gold.columns[:20]))
            return None

        result = pd.DataFrame(
            {
                "date": pd.to_datetime(gold[date_col], errors="coerce"),
                "mm_long": pd.to_numeric(gold[mm_long_col], errors="coerce"),
                "mm_short": pd.to_numeric(gold[mm_short_col], errors="coerce"),
            }
        )
        if oi_col:
            result["open_interest"] = pd.to_numeric(gold[oi_col], errors="coerce")

        result = result.dropna(subset=["date", "mm_long", "mm_short"])
        result["mm_net"] = result["mm_long"] - result["mm_short"]
        result = result.set_index("date").sort_index()

        # Keep only recent history
        cutoff = result.index.max() - pd.DateOffset(weeks=_HISTORY_WEEKS)
        result = result[result.index >= cutoff]

        cache_set("cot", cache_key, result)
        return result

    def get_latest_mm_net(self) -> float | None:
        """Return the most recent managed money net position.

        Returns:
            Net contracts (positive = net long, negative = net short),
            or None if data unavailable.
        """
        df = self.get_gold_managed_money()
        if df is None or df.empty:
            return None
        return float(df["mm_net"].iloc[-1])
