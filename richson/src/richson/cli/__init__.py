"""richson CLI entry points.

Sub-commands:
    backfill    - historical data backfill for cold-start (TRD SS13.3)
    backtest    - model validation over historical data (TRD SS18)
    update-weights - adjust dimension weights (TRD SS19)
"""

from __future__ import annotations

import argparse
import sys


def main(argv: list[str] | None = None) -> int:
    """Main CLI dispatcher.

    Usage:
        python -m richson.cli <command> [options]

    Commands:
        backfill        Backfill historical analysis records
        backtest        Run model direction accuracy backtest
        update-weights  Update dimension weights for a new model version
    """
    parser = argparse.ArgumentParser(
        prog="richson.cli",
        description="richson CLI tools",
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    # backfill sub-command
    from richson.cli.backfill import build_parser as backfill_parser  # noqa: PLC0415
    from richson.cli.backfill import main as backfill_main
    backfill_sp = subparsers.add_parser("backfill", help="Backfill historical analysis data")
    for action in backfill_parser()._actions:
        if action.dest != "help":
            backfill_sp._add_action(action)

    # backtest sub-command
    from richson.cli.backtest import build_parser as backtest_parser  # noqa: PLC0415
    from richson.cli.backtest import main as backtest_main
    backtest_sp = subparsers.add_parser("backtest", help="Run model validation backtest")
    for action in backtest_parser()._actions:
        if action.dest != "help":
            backtest_sp._add_action(action)

    # update-weights sub-command
    from richson.cli.weights import build_parser as weights_parser  # noqa: PLC0415
    from richson.cli.weights import main as weights_main
    weights_sp = subparsers.add_parser("update-weights", help="Update dimension weights")
    for action in weights_parser()._actions:
        if action.dest != "help":
            weights_sp._add_action(action)

    args = parser.parse_args(argv)

    if args.command == "backfill":
        return backfill_main(argv[1:] if argv else sys.argv[2:])
    elif args.command == "backtest":
        return backtest_main(argv[1:] if argv else sys.argv[2:])
    elif args.command == "update-weights":
        return weights_main(argv[1:] if argv else sys.argv[2:])

    parser.print_help()
    return 1


if __name__ == "__main__":
    sys.exit(main())
