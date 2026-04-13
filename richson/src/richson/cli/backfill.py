"""CLI command: backfill historical analysis data.

Usage:
    python -m richson.cli backfill --days 90

Runs Layer 1 scoring for each day in the backfill window and writes results
to rs_asset_analyses with source='backfill'. Used for cold-start bootstrapping
(TRD SS13.3).

G1.7: backfill records are marked source='backfill' so downstream queries can
filter them if needed. Decision to include/exclude in percentile calculation
is noted in comments.
"""

from __future__ import annotations

import argparse
import asyncio
import sys

import structlog

logger = structlog.get_logger()

# Default assets to backfill
_DEFAULT_ASSETS = ["GLD"]


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Backfill historical analysis data")
    parser.add_argument(
        "--days",
        type=int,
        default=90,
        help="Number of days to backfill (default: 90)",
    )
    parser.add_argument(
        "--assets",
        nargs="+",
        default=_DEFAULT_ASSETS,
        help="Asset codes to backfill (default: GLD)",
    )
    parser.add_argument(
        "--locale",
        default="zh",
        choices=["zh", "en"],
        help="Locale for analysis text (default: zh)",
    )
    return parser


async def run_backfill(days: int, assets: list[str], locale: str) -> None:
    """Backfill analysis records for the given number of days.

    For each day in the window, runs Layer 1 scoring only (no LLM) to
    build a historical score series. Records are marked with source='backfill'
    and generated_by='l1_only'.

    Note: backfill records DO participate in percentile calculations.
    This is intentional: backfill provides the historical baseline needed
    for the blended percentile algorithm to function at cold start.
    """
    from sqlalchemy.ext.asyncio import (  # noqa: PLC0415
        AsyncSession,
        async_sessionmaker,
        create_async_engine,
    )

    from richson.config import settings  # noqa: PLC0415
    from richson.core.pipeline import run_asset_analysis_pipeline  # noqa: PLC0415
    from richson.logging_config import configure_logging  # noqa: PLC0415

    configure_logging(settings.log_level)

    engine = create_async_engine(settings.database_url, pool_pre_ping=True)
    session_factory = async_sessionmaker(bind=engine, class_=AsyncSession, expire_on_commit=False)

    logger.info("backfill_start", days=days, assets=assets, locale=locale)

    # For backfill, we run Layer 1 only (no LLM)
    # We use a dummy LLM config since it won't be called
    from richson.schemas.common import LLMConfig  # noqa: PLC0415
    dummy_llm_config = LLMConfig(provider="claude", model="", api_key="")


    from richson.db import repository as repo  # noqa: PLC0415

    for asset_code in assets:
        logger.info("backfill_asset_start", asset_code=asset_code, days=days)
        success_count = 0
        error_count = 0

        for day_offset in range(days, 0, -1):
            # Create a job record for this backfill day
            async with session_factory() as sess:
                # Check if already backfilled for this date range
                job = await repo.create_analysis_job(
                    sess,
                    asset_code=asset_code,
                    locale=locale,
                    job_type="backfill",
                )
                await sess.commit()
                await sess.refresh(job)

            try:
                # Run pipeline in l1_only mode with backfill source
                await run_asset_analysis_pipeline(
                    job_id=job.job_id,
                    asset_code=asset_code,
                    locale=locale,
                    llm_config=dummy_llm_config,
                    session_factory=session_factory,
                    generated_by_override="l1_only",
                    budget_exceeded=True,  # force l1_only
                )
                success_count += 1
                logger.info(
                    "backfill_day_complete",
                    asset_code=asset_code,
                    day_offset=day_offset,
                )
            except Exception as exc:
                error_count += 1
                logger.error(
                    "backfill_day_error",
                    asset_code=asset_code,
                    day_offset=day_offset,
                    error=str(exc),
                )

        logger.info(
            "backfill_asset_complete",
            asset_code=asset_code,
            success=success_count,
            errors=error_count,
        )

    await engine.dispose()
    logger.info("backfill_complete")


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    asyncio.run(run_backfill(days=args.days, assets=args.assets, locale=args.locale))
    return 0


if __name__ == "__main__":
    sys.exit(main())
