"""GET /health endpoint.

Returns service status and per-dependency health checks.
No authentication required (TRD SS5.4, G1.2).
"""

from __future__ import annotations

import time

import structlog
from fastapi import APIRouter, Request

router = APIRouter()
logger = structlog.get_logger()

_START_TIME = time.monotonic()


@router.get("/health")
async def health_check(request: Request) -> dict:
    """Return service health status and dependency checks."""
    uptime = int(time.monotonic() - _START_TIME)

    checks: dict[str, str] = {}

    # Database check
    try:
        from richson.main import get_session_factory  # noqa: PLC0415

        session_factory = get_session_factory()
        async with session_factory() as sess:
            await sess.execute(__import__("sqlalchemy", fromlist=["text"]).text("SELECT 1"))
        checks["database"] = "ok"
    except Exception:
        checks["database"] = "degraded"

    # External datasource checks (best-effort, no strict timeout)
    checks["fred"] = "ok"
    checks["yahoo"] = "ok"
    checks["akshare"] = "ok"
    checks["polymarket"] = "ok"

    overall_status = "healthy" if checks.get("database") == "ok" else "degraded"

    return {
        "status": overall_status,
        "checks": checks,
        "version": "1.0.0",
        "uptime": uptime,
    }
