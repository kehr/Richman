"""In-process TTL cache layer for datasource responses.

Key format: ``<source>:<indicator>`` e.g. ``fred:DFII10``.
Each source has a dedicated TTLCache instance with its own TTL setting so that
cache lifetimes can be configured independently without sharing state.
"""

from __future__ import annotations

import threading
from typing import Any

from cachetools import TTLCache

# ---------------------------------------------------------------------------
# Per-source TTL caches (thread-safe via lock)
# ---------------------------------------------------------------------------

# TTLs in seconds
_FRED_TTL = 3600          # 1 hour
_YAHOO_PRICE_TTL = 300    # 5 minutes
_YAHOO_ETF_TTL = 3600     # 1 hour
_AKSHARE_TTL = 300        # 5 minutes
_POLYMARKET_TTL = 900     # 15 minutes
_COT_TTL = 86400          # 24 hours
_WGC_TTL = 2592000        # 30 days (~1 month)

_CACHE_MAXSIZE = 512  # max entries per cache

_fred_cache: TTLCache[str, Any] = TTLCache(maxsize=_CACHE_MAXSIZE, ttl=_FRED_TTL)
_yahoo_price_cache: TTLCache[str, Any] = TTLCache(maxsize=_CACHE_MAXSIZE, ttl=_YAHOO_PRICE_TTL)
_yahoo_etf_cache: TTLCache[str, Any] = TTLCache(maxsize=_CACHE_MAXSIZE, ttl=_YAHOO_ETF_TTL)
_akshare_cache: TTLCache[str, Any] = TTLCache(maxsize=_CACHE_MAXSIZE, ttl=_AKSHARE_TTL)
_polymarket_cache: TTLCache[str, Any] = TTLCache(maxsize=_CACHE_MAXSIZE, ttl=_POLYMARKET_TTL)
_cot_cache: TTLCache[str, Any] = TTLCache(maxsize=_CACHE_MAXSIZE, ttl=_COT_TTL)
_wgc_cache: TTLCache[str, Any] = TTLCache(maxsize=_CACHE_MAXSIZE, ttl=_WGC_TTL)

_lock = threading.Lock()

_SOURCE_CACHES: dict[str, TTLCache[str, Any]] = {
    "fred": _fred_cache,
    "yahoo_price": _yahoo_price_cache,
    "yahoo_etf": _yahoo_etf_cache,
    "akshare": _akshare_cache,
    "polymarket": _polymarket_cache,
    "cot": _cot_cache,
    "wgc": _wgc_cache,
}


def cache_get(source: str, key: str) -> Any | None:
    """Retrieve a cached value.

    Args:
        source: data source name (``fred``, ``yahoo_price``, etc.)
        key: indicator or asset identifier

    Returns:
        Cached value or ``None`` if not found / expired.
    """
    cache = _SOURCE_CACHES.get(source)
    if cache is None:
        return None
    with _lock:
        return cache.get(key)


def cache_set(source: str, key: str, value: Any) -> None:
    """Store a value in the cache.

    Args:
        source: data source name
        key: indicator or asset identifier
        value: value to store
    """
    cache = _SOURCE_CACHES.get(source)
    if cache is None:
        return
    with _lock:
        cache[key] = value


def cache_clear(source: str | None = None) -> None:
    """Clear one or all caches.

    Args:
        source: if given, clear only that source's cache; otherwise clear all.
    """
    with _lock:
        if source is not None:
            c = _SOURCE_CACHES.get(source)
            if c is not None:
                c.clear()
        else:
            for c in _SOURCE_CACHES.values():
                c.clear()
