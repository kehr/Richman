"""Polymarket prediction market API wrapper.

Fetches event probabilities for geopolitical and macroeconomic events
that are relevant to gold price dynamics (D1 and D3 dimensions).

Polymarket public API endpoints:
- GET https://clob.polymarket.com/markets  (paginated market list)
- GET https://clob.polymarket.com/markets/{condition_id}  (single market)

We filter for markets tagged with keywords relevant to gold:
- Federal Reserve rate decisions (D1)
- Geopolitical risk events (D3)

Results are cached for 15 minutes.
"""

from __future__ import annotations

from typing import Any

import httpx
import structlog

from richson.datasources.cache import cache_get, cache_set

logger = structlog.get_logger(__name__)

_CLOB_BASE = "https://clob.polymarket.com"
_GAMMA_BASE = "https://gamma-api.polymarket.com"

# Keywords to filter relevant markets
_GOLD_RELEVANT_TAGS = [
    "fed",
    "federal reserve",
    "rate cut",
    "interest rate",
    "inflation",
    "recession",
    "gold",
    "geopolitical",
    "war",
    "sanctions",
]


class PolymarketClient:
    """Polymarket public API wrapper.

    Args:
        timeout: HTTP request timeout in seconds.
        max_retries: retries on transient errors.
    """

    def __init__(self, timeout: int = 10, max_retries: int = 1) -> None:
        self._timeout = timeout
        self._max_retries = max_retries

    def _get(self, url: str, params: dict[str, Any] | None = None) -> Any | None:
        """Execute an HTTP GET with retry.

        Returns:
            Parsed JSON response or None on failure.
        """
        for attempt in range(self._max_retries + 1):
            try:
                with httpx.Client(timeout=self._timeout) as client:
                    resp = client.get(url, params=params)
                    resp.raise_for_status()
                    return resp.json()
            except Exception as exc:
                logger.warning(
                    "polymarket: request failed",
                    url=url,
                    attempt=attempt,
                    error=str(exc),
                )
        return None

    def get_active_markets(self, limit: int = 100) -> list[dict[str, Any]]:
        """Fetch active Polymarket markets via Gamma API.

        Args:
            limit: max number of markets to fetch.

        Returns:
            List of market dicts with fields: id, question, slug,
            outcomePrices, volume24hr, etc. Empty list on failure.
        """
        cache_key = f"active_markets:{limit}"
        cached = cache_get("polymarket", cache_key)
        if cached is not None:
            return cached  # type: ignore[return-value]

        data = self._get(
            f"{_GAMMA_BASE}/markets",
            params={"active": "true", "limit": limit, "order": "volume24hr"},
        )
        if data is None:
            return []

        markets: list[dict[str, Any]] = data if isinstance(data, list) else data.get("data", [])
        cache_set("polymarket", cache_key, markets)
        return markets

    def get_gold_relevant_markets(self) -> list[dict[str, Any]]:
        """Return markets relevant to gold price prediction.

        Filters active markets by keyword matching on question text and tags.

        Returns:
            Filtered list of market dicts with probability information.
        """
        cache_key = "gold_relevant"
        cached = cache_get("polymarket", cache_key)
        if cached is not None:
            return cached  # type: ignore[return-value]

        all_markets = self.get_active_markets(limit=200)
        relevant: list[dict[str, Any]] = []

        for market in all_markets:
            question: str = (market.get("question") or "").lower()
            tags: list[str] = [t.lower() for t in (market.get("tags") or [])]
            combined = question + " " + " ".join(tags)

            if any(kw in combined for kw in _GOLD_RELEVANT_TAGS):
                # Extract probability for "Yes" outcome if available
                outcome_prices = market.get("outcomePrices") or []
                if isinstance(outcome_prices, list) and outcome_prices:
                    try:
                        yes_prob = float(outcome_prices[0])
                    except (ValueError, TypeError):
                        yes_prob = 0.5
                else:
                    yes_prob = 0.5

                relevant.append(
                    {
                        "slug": market.get("slug", ""),
                        "question": market.get("question", ""),
                        "yes_probability": yes_prob,
                        "volume24hr": market.get("volume24hr", 0),
                        "end_date": market.get("endDate"),
                        "raw": market,
                    }
                )

        relevant.sort(key=lambda m: m.get("volume24hr", 0), reverse=True)
        cache_set("polymarket", cache_key, relevant)
        return relevant

    def get_rate_cut_probability(self) -> float | None:
        """Estimate Fed rate cut probability from Polymarket.

        Searches for the highest-volume rate cut market and returns its
        Yes probability (0.0 - 1.0).

        Returns:
            Float probability or None if no relevant market found.
        """
        cache_key = "rate_cut_prob"
        cached = cache_get("polymarket", cache_key)
        if cached is not None:
            return cached  # type: ignore[return-value]

        markets = self.get_gold_relevant_markets()
        rate_cut_keywords = ["rate cut", "fed cut", "cuts rates", "lower rates", "25bp", "50bp"]

        for market in markets:
            question = market.get("question", "").lower()
            if any(kw in question for kw in rate_cut_keywords):
                prob = market.get("yes_probability")
                if prob is not None:
                    cache_set("polymarket", cache_key, prob)
                    return prob
        return None

    def get_geopolitical_risk_index(self) -> float | None:
        """Compute a composite geopolitical risk index from multiple markets.

        Averages Yes probabilities of geopolitical risk markets (war, sanctions,
        major conflict). Returns 0.0-1.0 where higher = more risk.

        Returns:
            Float risk index or None if insufficient data.
        """
        cache_key = "geo_risk_index"
        cached = cache_get("polymarket", cache_key)
        if cached is not None:
            return cached  # type: ignore[return-value]

        geo_keywords = ["war", "conflict", "sanctions", "attack", "invasion"]
        markets = self.get_gold_relevant_markets()

        probs = []
        for market in markets:
            question = market.get("question", "").lower()
            if any(kw in question for kw in geo_keywords):
                prob = market.get("yes_probability")
                if prob is not None:
                    probs.append(float(prob))

        if not probs:
            return None

        index = sum(probs) / len(probs)
        cache_set("polymarket", cache_key, index)
        return index
