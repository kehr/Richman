"""CLI command: model validation backtest.

Usage:
    python -m richson.cli backtest --start 2020-01-01 --end 2025-12-31

Runs Layer 1 scoring over a 5-year historical window and computes direction
accuracy (score > 60 -> gold up, score < 40 -> gold down) against actual
price outcomes 1-3 months later. Output written to docs/validation/.

This is validation-only: results are NOT written to production tables.
(TRD SS18)
"""

from __future__ import annotations

import argparse
import asyncio
import csv
import sys
from datetime import date, timedelta
from pathlib import Path

import structlog

logger = structlog.get_logger()


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Backtest model direction accuracy")
    parser.add_argument(
        "--start",
        type=str,
        default="2020-01-01",
        help="Backtest start date (YYYY-MM-DD)",
    )
    parser.add_argument(
        "--end",
        type=str,
        default="2025-12-31",
        help="Backtest end date (YYYY-MM-DD)",
    )
    parser.add_argument(
        "--asset",
        default="GLD",
        help="Asset code to backtest (default: GLD)",
    )
    parser.add_argument(
        "--horizon-days",
        type=int,
        default=90,
        help="Forward horizon in days for direction accuracy (default: 90)",
    )
    parser.add_argument(
        "--output-dir",
        type=str,
        default="docs/validation",
        help="Directory to write validation results",
    )
    return parser


async def run_backtest(
    start: str,
    end: str,
    asset: str,
    horizon_days: int,
    output_dir: str,
) -> None:
    """Run model backtest over the given date range.

    Algorithm:
    1. For each month in the date range, compute L1 score using data up to that date
    2. Look forward horizon_days to compute actual price return
    3. Classify prediction as correct if:
       - score > 60 and price went up, OR
       - score < 40 and price went down
    4. Compute direction accuracy = correct_count / total_count
    """
    from richson.config import settings  # noqa: PLC0415
    from richson.logging_config import configure_logging  # noqa: PLC0415

    configure_logging(settings.log_level, settings.app_env)
    logger.info("backtest_start", start=start, end=end, asset=asset, horizon_days=horizon_days)

    from richson.core.indicators.d1_macro_rates import compute_d1_indicators  # noqa: PLC0415
    from richson.core.indicators.d2_dollar_liquidity import compute_d2_indicators  # noqa: PLC0415
    from richson.core.indicators.d3_structural_demand import compute_d3_indicators  # noqa: PLC0415
    from richson.core.indicators.d4_technical_position import compute_d4_indicators  # noqa: PLC0415
    from richson.core.scoring import compute_overall_score  # noqa: PLC0415
    from richson.datasources.fred import FREDClient  # noqa: PLC0415
    from richson.datasources.yahoo import YahooFinanceClient  # noqa: PLC0415

    yahoo = YahooFinanceClient()
    fred = FREDClient(api_key=settings.fred_api_key)

    # Fetch full historical data
    logger.info("fetching_historical_data")

    def _fetch_all():
        all_fred = fred.get_all_series()
        dxy_ohlcv = yahoo.get_dxy()
        return {
            "ohlcv": yahoo.get_ohlcv(asset),
            "fedfunds": all_fred.get("FEDFUNDS"),
            "t10y2y": all_fred.get("T10Y2Y"),
            "dfii10": all_fred.get("DFII10"),
            "dgs10": all_fred.get("DGS10"),
            "m2sl": all_fred.get("M2SL"),
            "bamlc": all_fred.get("BAMLC0A0CM"),
            "dxy_ohlcv": dxy_ohlcv,
        }

    data = await asyncio.to_thread(_fetch_all)

    start_date = date.fromisoformat(start)
    end_date = date.fromisoformat(end)

    results: list[dict] = []
    current_date = start_date

    dimension_weights = {"d1": 0.30, "d2": 0.25, "d3": 0.25, "d4": 0.20}

    import pandas as pd  # noqa: PLC0415

    # Monthly evaluation (first trading day of each month)
    while current_date <= end_date:
        cutoff_ts = pd.Timestamp(current_date)

        # Filter data to cutoff
        def _compute_scores(cutoff):
            results_inner = {}
            for key in ["ohlcv", "fedfunds", "t10y2y", "dfii10", "dgs10", "m2sl", "bamlc", "dxy_ohlcv"]:
                df_or_series = data[key]
                if df_or_series is None:
                    results_inner[key] = None
                    continue
                filtered = df_or_series[df_or_series.index <= cutoff]
                results_inner[key] = filtered if not filtered.empty else None

            d1 = compute_d1_indicators(
                results_inner.get("fedfunds"),
                results_inner.get("t10y2y"),
                results_inner.get("dfii10"),
                results_inner.get("dgs10"),
                None,  # no Polymarket in backtest
            )
            d2 = compute_d2_indicators(
                results_inner.get("dxy_ohlcv"),
                results_inner.get("m2sl"),
                None,  # TGA not tracked in backtest
            )
            d3 = compute_d3_indicators(
                None,  # cot_data not in backtest (no historical COT slicing)
                None,  # wgc_data not in backtest
                results_inner.get("ohlcv"),
                None,  # current_gold_price
            )
            d4 = compute_d4_indicators(results_inner.get("ohlcv"))

            dim_scores = {
                "d1": d1.get("base_score"),
                "d2": d2.get("base_score"),
                "d3": d3.get("base_score"),
                "d4": d4.get("base_score"),
            }

            try:
                overall = compute_overall_score(dim_scores, dimension_weights)
            except ValueError:
                overall = None

            return overall, dim_scores

        try:
            overall_score, dim_scores = await asyncio.to_thread(_compute_scores, cutoff_ts)
        except Exception as exc:
            logger.warning("backtest_score_error", date=str(current_date), error=str(exc))
            current_date = (current_date.replace(day=1) + timedelta(days=32)).replace(day=1)
            continue

        if overall_score is None:
            current_date = (current_date.replace(day=1) + timedelta(days=32)).replace(day=1)
            continue

        # Compute actual price return over horizon
        ohlcv = data["ohlcv"]
        actual_return = None
        direction_correct = None

        if ohlcv is not None:
            close_col = "Close" if "Close" in ohlcv.columns else "close"
            if close_col in ohlcv.columns:
                closes = ohlcv[close_col].dropna()
                near = closes[closes.index <= cutoff_ts]
                future_cutoff = cutoff_ts + pd.Timedelta(days=horizon_days)
                future = closes[(closes.index > cutoff_ts) & (closes.index <= future_cutoff)]

                if not near.empty and not future.empty:
                    price_now = float(near.iloc[-1])
                    price_future = float(future.iloc[-1])
                    actual_return = (price_future - price_now) / price_now

                    if overall_score > 60:
                        direction_correct = actual_return > 0
                    elif overall_score < 40:
                        direction_correct = actual_return < 0
                    # scores in 40-60 range are neutral; not counted in accuracy

        result_row = {
            "date": str(current_date),
            "overall_score": round(overall_score, 2),
            "d1_score": round(dim_scores.get("d1") or 0, 2),
            "d2_score": round(dim_scores.get("d2") or 0, 2),
            "d3_score": round(dim_scores.get("d3") or 0, 2),
            "d4_score": round(dim_scores.get("d4") or 0, 2),
            "actual_return": round(actual_return, 4) if actual_return is not None else None,
            "direction_correct": direction_correct,
        }
        results.append(result_row)
        logger.info("backtest_month", **result_row)

        # Advance to next month
        current_date = (current_date.replace(day=1) + timedelta(days=32)).replace(day=1)

    # Compute accuracy statistics
    directional_results = [r for r in results if r["direction_correct"] is not None]
    if directional_results:
        accuracy = sum(1 for r in directional_results if r["direction_correct"]) / len(directional_results)
        logger.info(
            "backtest_summary",
            total_months=len(results),
            directional_count=len(directional_results),
            direction_accuracy=round(accuracy, 4),
            minimum_required=0.60,
            passes=accuracy >= 0.60,
        )
    else:
        accuracy = 0.0
        logger.warning("backtest_no_directional_results")

    # Write results to CSV
    out_dir = Path(output_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    out_file = out_dir / f"backtest_{asset}_{start}_{end}.csv"

    with open(out_file, "w", newline="") as f:
        if results:
            writer = csv.DictWriter(f, fieldnames=list(results[0].keys()))
            writer.writeheader()
            writer.writerows(results)

    logger.info("backtest_results_written", path=str(out_file))

    if directional_results and accuracy < 0.60:
        logger.error(
            "backtest_failed_threshold",
            accuracy=round(accuracy, 4),
            minimum=0.60,
            message="Model does not meet 60%+ direction accuracy requirement. Adjust weights before production.",
        )


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    asyncio.run(
        run_backtest(
            start=args.start,
            end=args.end,
            asset=args.asset,
            horizon_days=args.horizon_days,
            output_dir=args.output_dir,
        )
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
