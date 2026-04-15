"""Datasource routing for OHLCV requests.

Picks the right datasource client based on the asset code shape. This is
the only place that should know which client serves which asset family.
Endpoints and pipelines must call ``fetch_ohlcv`` instead of instantiating
``YahooFinanceClient`` / ``StooqClient`` / ``AKShareClient`` directly.

Routing rule (matches ``backend/db/seed/asset_catalog.sql:data_source`` and
mirrors the Go-side rule in ``backend/internal/datasource/fetcher.go``):

* 6-digit purely-numeric code -> AKShare (CNY)
  Catalog ETF segments: 51xxxx / 56xxxx / 588xxx (SSE), 159xxx / 160xxx (SZSE)
* anything else -> Yahoo Finance, Stooq fallback (USD)
  Includes ``AAPL``, ``GLD``, ``^GSPC``, ``GC=F``, ``000001.SS``...

Adding a catalog entry whose ``data_source`` does not match this rule is a
violation; see ``docs/standards/richson-datasource-routing.md``.
"""

from __future__ import annotations

import pandas as pd
import structlog

logger = structlog.get_logger(__name__)


def is_a_share_code(code: str) -> bool:
    """Return True if ``code`` looks like an A-share ticker.

    A-share ETFs and stocks both use 6-digit purely-numeric codes. The MVP
    catalog only ships ETFs, so a numeric code that happens to be an
    individual stock (e.g. ``600519``) will currently fall through to
    AKShare's ETF endpoint and return None -> 502. That's an acceptable
    failure mode until we extend AKShareClient to cover stocks.
    """
    return len(code) == 6 and code.isdigit()


def resolve_currency(code: str) -> str:
    """Pick currency based on the routing rule.

    Mirrors ``is_a_share_code`` so the answer never drifts from the
    datasource selection.
    """
    return "CNY" if is_a_share_code(code) else "USD"


def fetch_ohlcv(code: str) -> pd.DataFrame | None:
    """Fetch OHLCV from the datasource appropriate for ``code``.

    Returns:
        DataFrame with columns ``open|high|low|close|volume`` indexed by
        date (ascending), or None when no datasource produced data.
    """
    if is_a_share_code(code):
        from richson.datasources.akshare_client import AKShareClient  # noqa: PLC0415

        df = AKShareClient().get_etf_ohlcv(code)
        if df is None or df.empty:
            logger.warning("routing: akshare returned no data", code=code)
            return None
        return df

    from richson.datasources.stooq import StooqClient  # noqa: PLC0415
    from richson.datasources.yahoo import YahooFinanceClient  # noqa: PLC0415

    df = YahooFinanceClient().get_ohlcv(code)
    if df is not None and not df.empty:
        return df

    df = StooqClient().get_ohlcv(code)
    if df is None or df.empty:
        logger.warning("routing: yahoo and stooq both empty", code=code)
        return None
    return df
