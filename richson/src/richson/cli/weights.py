"""CLI command: update dimension weights.

Usage:
    python -m richson.cli update-weights \\
        --asset-type gold \\
        --d1 0.30 --d2 0.25 --d3 0.25 --d4 0.20 \\
        --version gold_v1.1

Validates that:
- Weights sum to 1.0 (within float tolerance)
- Each weight change is within +/-10% of the previous value

Inserts new rs_dimension_definitions rows with the new model_version.
(TRD SS6.2, SS19)
"""

from __future__ import annotations

import argparse
import asyncio
import sys

import structlog

logger = structlog.get_logger()

_DIMENSION_NAMES: dict[str, tuple[str, str]] = {
    "d1": ("宏观利率", "Macro Rates"),
    "d2": ("美元流动性", "Dollar Liquidity"),
    "d3": ("结构性需求", "Structural Demand"),
    "d4": ("技术位置", "Technical Position"),
}

_MAX_WEIGHT_CHANGE = 0.10  # +/-10% per dimension


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Update dimension weights for a model version")
    parser.add_argument("--asset-type", default="gold", help="Asset type (default: gold)")
    parser.add_argument("--d1", type=float, required=True, help="D1 weight (0.0-1.0)")
    parser.add_argument("--d2", type=float, required=True, help="D2 weight (0.0-1.0)")
    parser.add_argument("--d3", type=float, required=True, help="D3 weight (0.0-1.0)")
    parser.add_argument("--d4", type=float, required=True, help="D4 weight (0.0-1.0)")
    parser.add_argument("--version", required=True, help="New model version (e.g. gold_v1.1)")
    return parser


async def run_update_weights(
    asset_type: str,
    d1: float,
    d2: float,
    d3: float,
    d4: float,
    version: str,
) -> None:
    """Validate and apply new dimension weights.

    Reads current weights from the latest version in rs_dimension_definitions,
    validates changes, and inserts new records.
    """
    from sqlalchemy.ext.asyncio import (  # noqa: PLC0415
        AsyncSession,
        async_sessionmaker,
        create_async_engine,
    )

    from richson.config import settings  # noqa: PLC0415
    from richson.db.models import DimensionDefinition  # noqa: PLC0415
    from richson.logging_config import configure_logging  # noqa: PLC0415

    configure_logging(settings.log_level, settings.app_env)

    new_weights = {"d1": d1, "d2": d2, "d3": d3, "d4": d4}

    # Validate sum
    total = sum(new_weights.values())
    if abs(total - 1.0) > 0.001:
        logger.error("weight_sum_invalid", total=total)
        print(f"Error: weights must sum to 1.0, got {total:.4f}")
        sys.exit(1)

    engine = create_async_engine(settings.database_url, pool_pre_ping=True)
    session_factory = async_sessionmaker(bind=engine, class_=AsyncSession, expire_on_commit=False)

    # Find current version
    async with session_factory() as sess:
        from sqlalchemy import select  # noqa: PLC0415
        stmt = (
            select(DimensionDefinition)
            .where(
                DimensionDefinition.asset_type == asset_type,
                DimensionDefinition.is_deleted == 0,
            )
            .order_by(DimensionDefinition.created_at.desc())
        )
        result = await sess.execute(stmt)
        existing = result.scalars().all()

    # Group by version to find the latest
    current_weights: dict[str, float] = {}
    if existing:
        # Get most recent version's weights
        latest_version = existing[0].model_version
        for row in existing:
            if row.model_version == latest_version:
                dim_key = row.dimension.lower()
                current_weights[dim_key] = float(row.weight)

    # Validate per-dimension change <= 10%
    if current_weights:
        for dim, new_w in new_weights.items():
            old_w = current_weights.get(dim)
            if old_w is not None:
                change = abs(new_w - old_w)
                if change > _MAX_WEIGHT_CHANGE + 0.001:
                    logger.error(
                        "weight_change_too_large",
                        dimension=dim,
                        old=old_w,
                        new=new_w,
                        change=change,
                    )
                    print(
                        f"Error: {dim} weight change {change:.3f} exceeds maximum {_MAX_WEIGHT_CHANGE}. "
                        f"For major restructuring, increment major version."
                    )
                    sys.exit(1)

    # Insert new dimension definitions
    async with session_factory() as sess:
        for order, (dim_key, (name_zh, name_en)) in enumerate(
            _DIMENSION_NAMES.items(), start=1
        ):
            new_def = DimensionDefinition(
                asset_type=asset_type,
                dimension=dim_key.upper(),
                name_zh=name_zh,
                name_en=name_en,
                weight=new_weights[dim_key],
                display_order=order,
                model_version=version,
                description_zh=f"{name_zh}维度权重配置",
                description_en=f"{name_en} dimension weight configuration",
            )
            sess.add(new_def)
        await sess.commit()

    logger.info(
        "weights_updated",
        asset_type=asset_type,
        version=version,
        d1=d1,
        d2=d2,
        d3=d3,
        d4=d4,
    )
    print(f"Successfully updated weights to version {version}:")
    for dim, w in new_weights.items():
        print(f"  {dim.upper()}: {w:.2f}")

    await engine.dispose()


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    asyncio.run(
        run_update_weights(
            asset_type=args.asset_type,
            d1=args.d1,
            d2=args.d2,
            d3=args.d3,
            d4=args.d4,
            version=args.version,
        )
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
