"""Quant engine core: pure computation modules for Layer 1 scoring.

Public API:
- scoring     : blended_percentile, weighted_dimension_score, compute_overall_score
- adjustment  : compute_adjustment, apply_adjustment_to_score
- confidence  : compute_confidence, check_fred_freshness
- support_resistance : compute_support_resistance
- regime      : detect_regime
- event_monitor : detect_probability_changes
- drawdown    : compute_drawdown_reference
- conflict    : detect_conflict, check_llm_anomaly
- indicators  : compute_d1_indicators, compute_d2_indicators, compute_d3_indicators,
                compute_d4_indicators
"""

from richson.core.adjustment import apply_adjustment_to_score, compute_adjustment
from richson.core.confidence import compute_confidence
from richson.core.conflict import check_llm_anomaly, detect_conflict
from richson.core.drawdown import compute_drawdown_reference
from richson.core.event_monitor import detect_probability_changes
from richson.core.indicators import (
    compute_d1_indicators,
    compute_d2_indicators,
    compute_d3_indicators,
    compute_d4_indicators,
)
from richson.core.regime import detect_regime
from richson.core.scoring import (
    blended_percentile,
    compute_overall_score,
    signal_level_from_score,
    weighted_dimension_score,
)
from richson.core.support_resistance import compute_support_resistance

__all__ = [
    # scoring
    "blended_percentile",
    "weighted_dimension_score",
    "compute_overall_score",
    "signal_level_from_score",
    # adjustment
    "compute_adjustment",
    "apply_adjustment_to_score",
    # confidence
    "compute_confidence",
    # conflict
    "detect_conflict",
    "check_llm_anomaly",
    # drawdown
    "compute_drawdown_reference",
    # event monitor
    "detect_probability_changes",
    # regime
    "detect_regime",
    # support/resistance
    "compute_support_resistance",
    # indicators
    "compute_d1_indicators",
    "compute_d2_indicators",
    "compute_d3_indicators",
    "compute_d4_indicators",
]
