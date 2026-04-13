"""Dimension indicator calculators for the quant engine.

Each module computes raw indicator values and blended percentile scores
for one dimension. All functions are pure computation:
- Input: pandas DataFrames / dicts / scalar values from datasource layer
- Output: dicts containing raw values, percentile scores, and metadata

Modules:
- d1_macro_rates    : D1 macro/rate indicators
- d2_dollar_liquidity : D2 dollar & liquidity indicators
- d3_structural_demand : D3 structural demand indicators
- d4_technical_position : D4 technical position indicators
"""

from richson.core.indicators.d1_macro_rates import compute_d1_indicators
from richson.core.indicators.d2_dollar_liquidity import compute_d2_indicators
from richson.core.indicators.d3_structural_demand import compute_d3_indicators
from richson.core.indicators.d4_technical_position import compute_d4_indicators

__all__ = [
    "compute_d1_indicators",
    "compute_d2_indicators",
    "compute_d3_indicators",
    "compute_d4_indicators",
]
