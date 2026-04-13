"""Asyncio-based event polling scheduler for richson internal tasks.

Responsibilities (TRD SS9.3, SS13.1, SS13.2):
1. Polymarket event probability monitoring (hourly) - detects >= 20% probability shifts
   and writes alert records to rs_event_alerts.
2. rs_asset_analyses data cleanup (weekly, Sunday UTC 03:00) - soft-deletes records
   beyond the retention window.
3. rs_analysis_jobs expired job cleanup (every 10 min) - handled by richman cron;
   richson scheduler only handles its own events table.

Scheduler runs as a background asyncio task started in the FastAPI lifespan.
All intervals are in seconds.
"""

from __future__ import annotations

import asyncio
from datetime import UTC, datetime, timedelta
from typing import Any

import structlog

logger = structlog.get_logger()

# Polling intervals (seconds)
_POLYMARKET_POLL_INTERVAL = 3600   # 1 hour
_DATA_CLEANUP_INTERVAL = 86400 * 7  # 1 week (check daily, act weekly)

# Data retention windows (TRD SS13.1)
_RETENTION_FULL_DAYS = 365           # keep all records for 1 year
_RETENTION_WEEKLY_DAYS = 365 * 5    # 1-5 years: keep weekly (Monday records)

# Previous Polymarket snapshot for delta comparison
_prev_poly_snapshot: list[dict[str, Any]] = []


async def poll_polymarket_events(session_factory: Any) -> None:
    """Fetch Polymarket market probabilities and detect significant changes.

    Writes alert records to rs_event_alerts when delta >= 20% threshold.
    """
    from richson.core.event_monitor import (  # noqa: PLC0415
        build_event_snapshot,
        detect_probability_changes,
    )
    from richson.datasources.polymarket import PolymarketClient  # noqa: PLC0415
    from richson.db import repository as repo  # noqa: PLC0415

    global _prev_poly_snapshot

    poly_client = PolymarketClient()

    try:
        markets = await asyncio.to_thread(poly_client.get_gold_relevant_markets)
    except Exception as exc:
        logger.warning("polymarket_poll_failed", error=str(exc))
        return

    if not _prev_poly_snapshot:
        # First run: just capture snapshot, no comparison
        _prev_poly_snapshot = build_event_snapshot(markets)
        logger.info("polymarket_snapshot_initialized", market_count=len(markets))
        return

    alerts = detect_probability_changes(markets, _prev_poly_snapshot)
    _prev_poly_snapshot = build_event_snapshot(markets)

    if not alerts:
        logger.debug("polymarket_no_significant_changes")
        return

    logger.info("polymarket_alerts_detected", count=len(alerts))

    async with session_factory() as sess:
        for alert_data in alerts:
            try:
                await repo.create_event_alert(sess, alert_data)
            except Exception as exc:
                # Unique index violation = already alerted; ignore
                logger.debug("event_alert_duplicate", slug=alert_data.get("event_slug"), error=str(exc))
        await sess.commit()


async def cleanup_old_analyses(session_factory: Any) -> None:
    """Soft-delete rs_asset_analyses records beyond retention window (TRD SS13.1).

    Retention policy:
    - < 365 days: keep all
    - 365 days - 5 years: keep Monday records only (weekly)
    - > 5 years: soft-delete all
    """
    from sqlalchemy import select, update  # noqa: PLC0415

    from richson.db.models import AssetAnalysis  # noqa: PLC0415

    now = datetime.now(tz=UTC)
    cutoff_full = now - timedelta(days=_RETENTION_FULL_DAYS)
    cutoff_weekly = now - timedelta(days=_RETENTION_WEEKLY_DAYS)

    logger.info("cleanup_start", cutoff_full=str(cutoff_full), cutoff_weekly=str(cutoff_weekly))

    async with session_factory() as sess:
        # Soft-delete all records older than 5 years
        stmt_5y = (
            update(AssetAnalysis)
            .where(
                AssetAnalysis.analyzed_at < cutoff_weekly,
                AssetAnalysis.is_deleted == 0,
            )
            .values(is_deleted=1, modifier="scheduler")
        )
        result_5y = await sess.execute(stmt_5y)

        # For 1-5 year range, keep only Monday records (weekday() == 0)
        # Fetch non-Monday records in this range to soft-delete them
        stmt_select = select(AssetAnalysis).where(
            AssetAnalysis.analyzed_at >= cutoff_weekly,
            AssetAnalysis.analyzed_at < cutoff_full,
            AssetAnalysis.is_deleted == 0,
        )
        result = await sess.execute(stmt_select)
        records_in_range = result.scalars().all()

        delete_ids = [
            r.asset_analysis_id
            for r in records_in_range
            if r.analyzed_at.weekday() != 0  # 0 = Monday
        ]

        if delete_ids:
            stmt_range = (
                update(AssetAnalysis)
                .where(AssetAnalysis.asset_analysis_id.in_(delete_ids))
                .values(is_deleted=1, modifier="scheduler")
            )
            await sess.execute(stmt_range)

        await sess.commit()

    logger.info(
        "cleanup_complete",
        deleted_5y_plus=result_5y.rowcount,
        deleted_1_5y_non_monday=len(delete_ids),
    )


async def _run_with_interval(
    name: str,
    interval_s: int,
    coro_fn: Any,
    session_factory: Any,
) -> None:
    """Run a coroutine on a fixed interval, logging errors without crashing."""
    while True:
        try:
            await coro_fn(session_factory)
        except Exception as exc:
            logger.error("scheduler_task_error", task=name, error=str(exc))
        await asyncio.sleep(interval_s)


async def start_scheduler(session_factory: Any) -> list[asyncio.Task]:
    """Start all scheduler tasks and return the task handles.

    Called from FastAPI lifespan. The returned tasks are cancelled on shutdown.
    """
    logger.info("scheduler_starting")

    tasks = [
        asyncio.create_task(
            _run_with_interval(
                "polymarket_events",
                _POLYMARKET_POLL_INTERVAL,
                poll_polymarket_events,
                session_factory,
            ),
            name="polymarket_events",
        ),
        asyncio.create_task(
            _run_with_interval(
                "cleanup_analyses",
                _DATA_CLEANUP_INTERVAL,
                cleanup_old_analyses,
                session_factory,
            ),
            name="cleanup_analyses",
        ),
    ]

    logger.info("scheduler_started", task_count=len(tasks))
    return tasks
