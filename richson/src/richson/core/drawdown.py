"""Max drawdown calculation (TRD SS7.7).

Computes:
1. Current bull-run start date (based on 20% drawdown from prior peak)
2. Max drawdown within the current bull-run
3. Historical average max drawdown across all completed bull-runs

Input: OHLCV DataFrame.
Output: dict with drawdown metrics for rs_asset_analyses.analysis_metadata.
"""

from __future__ import annotations

import numpy as np
import pandas as pd
import structlog

logger = structlog.get_logger(__name__)

_BULL_RUN_START_THRESHOLD = -0.20  # -20% drawdown from peak = bull-run start marker


def compute_drawdown_reference(ohlcv: pd.DataFrame) -> dict:
    """Compute current and historical drawdown statistics (TRD SS7.7).

    Algorithm:
    1. Identify current bull-run start: most recent point where the drawdown
       from any prior peak exceeded 20%, or series start if no such event exists.
    2. Compute max drawdown within current bull-run:
       peak = rolling max of close; drawdown = (close - peak) / peak.
    3. Compute historical average max drawdown across all completed bull-runs.

    Args:
        ohlcv: daily OHLCV DataFrame with at least a ``close`` column.
               Minimum 60 rows recommended; at least 2 rows required.

    Returns:
        Dict with keys:
        - ``currentBullRunStart`` (str | None): ISO date string
        - ``maxDrawdown`` (float | None): max drawdown in current bull-run, e.g. -0.085
        - ``maxDrawdownDate`` (str | None): ISO date of worst drawdown
        - ``historicalAvgDrawdown`` (float | None): average across completed runs
    """
    if ohlcv is None or len(ohlcv) < 2:
        return {
            "currentBullRunStart": None,
            "maxDrawdown": None,
            "maxDrawdownDate": None,
            "historicalAvgDrawdown": None,
        }

    close = ohlcv["close"].dropna().sort_index()
    if len(close) < 2:
        return {
            "currentBullRunStart": None,
            "maxDrawdown": None,
            "maxDrawdownDate": None,
            "historicalAvgDrawdown": None,
        }

    # --- Identify all bull-run boundaries (20% drawdowns) ---
    rolling_peak = close.cummax()
    drawdown_series = (close - rolling_peak) / rolling_peak

    # Find indices where drawdown first crosses -20% threshold
    # (start of a new bear market / bull-run break)
    bear_crossings = drawdown_series[drawdown_series <= _BULL_RUN_START_THRESHOLD]

    if bear_crossings.empty:
        # No significant drawdown in series: entire history is one bull run
        bull_run_start_idx = close.index[0]
    else:
        # Current bull run starts after the last bear crossing
        # The recovery begins at the local low after the last -20% crossing
        last_bear_cross = bear_crossings.index[-1]
        # Find local minimum after last bear crossing
        post_bear = close[close.index >= last_bear_cross]
        bull_run_start_idx = last_bear_cross if post_bear.empty else post_bear.idxmin()

    # --- Compute max drawdown within current bull run ---
    current_run = close[close.index >= bull_run_start_idx]

    if len(current_run) < 2:
        max_dd = None
        max_dd_date = None
    else:
        run_peak = current_run.cummax()
        run_drawdown = (current_run - run_peak) / run_peak
        max_dd = float(run_drawdown.min())
        max_dd_date_idx = run_drawdown.idxmin()
        max_dd_date = (
            max_dd_date_idx.strftime("%Y-%m-%d")
            if isinstance(max_dd_date_idx, pd.Timestamp)
            else str(max_dd_date_idx)
        )

    # --- Historical average max drawdown across completed bull runs ---
    hist_avg = _compute_historical_avg_drawdown(close, bear_crossings)

    bull_run_start_str = (
        bull_run_start_idx.strftime("%Y-%m-%d")
        if isinstance(bull_run_start_idx, pd.Timestamp)
        else str(bull_run_start_idx)
    )

    return {
        "currentBullRunStart": bull_run_start_str,
        "maxDrawdown": round(max_dd, 4) if max_dd is not None else None,
        "maxDrawdownDate": max_dd_date,
        "historicalAvgDrawdown": round(hist_avg, 4) if hist_avg is not None else None,
    }


def _compute_historical_avg_drawdown(
    close: pd.Series,
    bear_crossings: pd.Series,
) -> float | None:
    """Compute average max drawdown across all completed bull-run periods.

    A completed bull run is a segment between two consecutive bear-crossing events.

    Args:
        close: full close price series.
        bear_crossings: sub-series where drawdown <= -20%.

    Returns:
        Average max drawdown float (negative), or None if fewer than 2 bear crossings.
    """
    if len(bear_crossings) < 1:
        return None

    # Identify run segment start points: series start + each bear-crossing low
    segment_starts: list[pd.Timestamp] = [close.index[0]]

    for crossing_idx in bear_crossings.index:
        post = close[close.index >= crossing_idx]
        if not post.empty:
            segment_starts.append(post.idxmin())

    if len(segment_starts) < 2:
        return None

    max_drawdowns: list[float] = []
    for i in range(len(segment_starts) - 1):
        seg = close[
            (close.index >= segment_starts[i]) & (close.index < segment_starts[i + 1])
        ]
        if len(seg) < 5:
            continue
        seg_peak = seg.cummax()
        seg_dd = (seg - seg_peak) / seg_peak
        max_drawdowns.append(float(seg_dd.min()))

    if not max_drawdowns:
        return None

    return float(np.mean(max_drawdowns))
