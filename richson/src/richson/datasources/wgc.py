"""World Gold Council (WGC) data wrapper.

WGC does not provide a public API. Data is sourced via:
1. Attempt: parse WGC open data downloads (CSV/Excel on gold.org)
2. Fallback: read from local seed file config/wgc_quarterly.json

Data captured:
- Central bank net purchases (tonnes, quarterly / annualized)
- AISC (All-In Sustaining Cost) per oz in USD

WGC data updates quarterly. TTL cache is set to 30 days.
The seed file provides manually-maintained fallback values for when
web parsing fails.
"""

from __future__ import annotations

import json
import logging
from pathlib import Path
from typing import Any

import httpx

from richson.datasources.cache import cache_get, cache_set

logger = logging.getLogger(__name__)

# Relative path from this file to the config directory
_SEED_FILE = Path(__file__).parent.parent / "config" / "wgc_quarterly.json"

# WGC statistics page (public download links)
_WGC_DEMAND_URL = "https://www.gold.org/download/ref_8743/goldhub-data.xlsx"


class WGCClient:
    """WGC data wrapper with seed-file fallback.

    Args:
        timeout: HTTP request timeout in seconds.
        seed_file: path to the JSON seed file; defaults to config/wgc_quarterly.json.
    """

    def __init__(
        self,
        timeout: int = 30,
        seed_file: Path | None = None,
    ) -> None:
        self._timeout = timeout
        self._seed_file = seed_file or _SEED_FILE

    def _load_seed(self) -> dict[str, Any]:
        """Load the manually maintained seed file.

        Returns:
            Dict with keys: central_bank_net_tonnes, aisc_usd_per_oz, quarter, updated_at.
            Falls back to safe defaults if file is missing or malformed.
        """
        try:
            if self._seed_file.exists():
                with open(self._seed_file) as f:
                    return json.load(f)  # type: ignore[return-value]
        except Exception as exc:
            logger.warning("wgc: seed file read failed", error=str(exc))
        # Safe defaults based on 2024 estimates
        return {
            "central_bank_net_tonnes": 800,  # approx 2024 annualized
            "aisc_usd_per_oz": 1350,         # approx 2024 industry average
            "quarter": "2024Q3",
            "updated_at": "2025-01-01",
        }

    def get_quarterly_data(self) -> dict[str, Any]:
        """Return the latest available WGC quarterly data.

        Attempts to fetch from WGC website; falls back to seed file on failure.

        Returns:
            Dict with:
            - central_bank_net_tonnes (float): annualized central bank net purchases
            - aisc_usd_per_oz (float): industry average AISC
            - quarter (str): reference quarter, e.g. ``2024Q3``
            - source (str): ``web`` or ``seed``
        """
        cache_key = "quarterly_data"
        cached = cache_get("wgc", cache_key)
        if cached is not None:
            return cached  # type: ignore[return-value]

        result = self._fetch_from_web()
        if result is None:
            seed = self._load_seed()
            result = {**seed, "source": "seed"}
        else:
            result["source"] = "web"

        cache_set("wgc", cache_key, result)
        return result

    def _fetch_from_web(self) -> dict[str, Any] | None:
        """Attempt to parse WGC data from their open data page.

        Returns:
            Parsed data dict or None if unavailable / unparseable.
        """
        # WGC data format changes periodically; we probe a simple JSON endpoint
        # that aggregates gold demand statistics.
        probe_url = "https://www.gold.org/goldhub/data/central-bank-statistics"
        try:
            with httpx.Client(timeout=self._timeout) as client:
                resp = client.get(probe_url, headers={"Accept": "application/json"})
                if resp.status_code == 200:
                    data = resp.json()
                    # Extract latest central bank net purchase if present
                    cb_tonnes = self._extract_cb_net(data)
                    if cb_tonnes is not None:
                        seed = self._load_seed()
                        return {
                            "central_bank_net_tonnes": cb_tonnes,
                            "aisc_usd_per_oz": seed["aisc_usd_per_oz"],
                            "quarter": "latest",
                        }
        except Exception as exc:
            logger.info("wgc: web fetch unavailable, using seed", error=str(exc))
        return None

    @staticmethod
    def _extract_cb_net(data: Any) -> float | None:
        """Extract central bank net purchase from WGC JSON response.

        The WGC API format is not publicly documented and may change.
        This method handles the known formats gracefully.

        Args:
            data: parsed JSON response.

        Returns:
            Net tonnes float or None.
        """
        if isinstance(data, dict):
            for key in ("netPurchases", "net_purchases", "value", "total"):
                val = data.get(key)
                if val is not None:
                    try:
                        return float(val)
                    except (TypeError, ValueError):
                        pass
        if isinstance(data, list) and data:
            first = data[-1]
            if isinstance(first, dict):
                return WGCClient._extract_cb_net(first)
        return None

    def get_aisc(self) -> float:
        """Return the latest AISC (USD per oz).

        Returns:
            AISC value; falls back to seed default if data unavailable.
        """
        data = self.get_quarterly_data()
        return float(data.get("aisc_usd_per_oz", 1350))

    def get_central_bank_net_tonnes(self) -> float:
        """Return latest annualized central bank net purchase (tonnes).

        Returns:
            Tonnes purchased; falls back to seed default if unavailable.
        """
        data = self.get_quarterly_data()
        return float(data.get("central_bank_net_tonnes", 800))
