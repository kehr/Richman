"""Datasource layer: external data fetching with caching.

Each client wraps one external data source and returns pandas DataFrames
or primitive values. All clients handle errors gracefully (return None on
failure) and use the shared TTL cache layer.
"""

from richson.datasources.akshare_client import AKShareClient
from richson.datasources.cache import cache_clear, cache_get, cache_set
from richson.datasources.cot import COTClient
from richson.datasources.fred import FREDClient
from richson.datasources.polymarket import PolymarketClient
from richson.datasources.routing import fetch_ohlcv, is_a_share_code, resolve_currency
from richson.datasources.stooq import StooqClient
from richson.datasources.wgc import WGCClient
from richson.datasources.yahoo import YahooFinanceClient

__all__ = [
    "AKShareClient",
    "COTClient",
    "FREDClient",
    "PolymarketClient",
    "StooqClient",
    "WGCClient",
    "YahooFinanceClient",
    "cache_clear",
    "cache_get",
    "cache_set",
    "fetch_ohlcv",
    "is_a_share_code",
    "resolve_currency",
]
