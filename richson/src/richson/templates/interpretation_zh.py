"""Chinese degradation templates for market interpretation.

These templates generate structured Chinese text from quantitative scores only,
without any LLM calls. Used when the interpretation agent fails or is unavailable.

Generated output is marked with generated_by='l1_only' by the pipeline.
"""

from __future__ import annotations

from dataclasses import dataclass

# ---------------------------------------------------------------------------
# Signal level -> label mapping
# ---------------------------------------------------------------------------

_SIGNAL_LABELS: dict[str, str] = {
    "strong_bullish": "强烈看多",
    "moderate_bullish": "温和看多",
    "neutral": "中性观望",
    "moderate_bearish": "温和看空",
    "strong_bearish": "强烈看空",
}

_SIGNAL_VERBS: dict[str, str] = {
    "strong_bullish": "配置信号强烈",
    "moderate_bullish": "配置信号积极",
    "neutral": "方向信号中性",
    "moderate_bearish": "配置信号偏弱",
    "strong_bearish": "配置信号较差",
}

_SCORE_RANGE_LABEL: dict[tuple[float, float], str] = {
    (80, 100): "高位",
    (60, 80): "中高位",
    (40, 60): "中位",
    (20, 40): "中低位",
    (0, 20): "低位",
}

_DIMENSION_NAMES: dict[str, str] = {
    "D1": "宏观利率",
    "D2": "美元流动性",
    "D3": "结构性需求",
    "D4": "技术位置",
}


def _score_to_range_label(score: float) -> str:
    for (low, high), label in _SCORE_RANGE_LABEL.items():
        if low <= score < high:
            return label
    return "高位" if score >= 80 else "低位"


def _format_score_label(score: float) -> str:
    if score >= 70:
        return "偏强"
    elif score >= 55:
        return "中性偏强"
    elif score >= 45:
        return "中性"
    elif score >= 30:
        return "中性偏弱"
    else:
        return "偏弱"


def _strongest_dimension(d_scores: dict[str, float | None]) -> tuple[str, float]:
    """Return the dimension with the highest score (name_zh, score)."""
    valid = {k: v for k, v in d_scores.items() if v is not None}
    if not valid:
        return ("综合", 50.0)
    best_key = max(valid, key=lambda k: valid[k])  # type: ignore[arg-type]
    return (_DIMENSION_NAMES.get(best_key, best_key), valid[best_key])  # type: ignore[return-value]


def _weakest_dimension(d_scores: dict[str, float | None]) -> tuple[str, float]:
    """Return the dimension with the lowest score (name_zh, score)."""
    valid = {k: v for k, v in d_scores.items() if v is not None}
    if not valid:
        return ("综合", 50.0)
    worst_key = min(valid, key=lambda k: valid[k])  # type: ignore[arg-type]
    return (_DIMENSION_NAMES.get(worst_key, worst_key), valid[worst_key])  # type: ignore[return-value]


# ---------------------------------------------------------------------------
# Template output dataclass
# ---------------------------------------------------------------------------


@dataclass
class InterpretationTemplateResult:
    """Output of a degradation template render."""

    market_interpretation: str
    risk_factors: list[str]
    regime_summary: str
    major_change_recap: str | None
    change_summary: str | None


# ---------------------------------------------------------------------------
# Main render function
# ---------------------------------------------------------------------------


def render_interpretation_zh(
    overall_score: float,
    signal_level: str,
    d1_score: float | None = None,
    d2_score: float | None = None,
    d3_score: float | None = None,
    d4_score: float | None = None,
    prev_score: float | None = None,
    score_delta: float | None = None,
    asset_code: str = "黄金",
) -> InterpretationTemplateResult:
    """Render Chinese market interpretation from quantitative scores.

    Generates structured analysis text without LLM. Suitable for degraded mode
    when the interpretation agent is unavailable.

    Args:
        overall_score: Composite score 0-100.
        signal_level: Signal level string (e.g., "moderate_bullish").
        d1_score: D1 Macro Rates score, or None if unavailable.
        d2_score: D2 Dollar/Liquidity score, or None if unavailable.
        d3_score: D3 Structural Demand score, or None if unavailable.
        d4_score: D4 Technical Position score, or None if unavailable.
        prev_score: Previous overall score for change tracking.
        score_delta: Score change from previous analysis.
        asset_code: Asset display name for text generation.

    Returns:
        InterpretationTemplateResult with all required interpretation fields.
    """
    signal_label = _SIGNAL_LABELS.get(signal_level, "中性")
    signal_verb = _SIGNAL_VERBS.get(signal_level, "方向信号中性")
    score_position = _score_to_range_label(overall_score)

    d_scores: dict[str, float | None] = {
        "D1": d1_score,
        "D2": d2_score,
        "D3": d3_score,
        "D4": d4_score,
    }
    strongest_name, strongest_score = _strongest_dimension(d_scores)
    weakest_name, weakest_score = _weakest_dimension(d_scores)

    # Build market_interpretation
    if overall_score >= 60:
        interpretation = (
            f"量化模型显示{asset_code}综合评分{overall_score:.0f}分，{signal_label}，"
            f"{strongest_name}维度支撑最强（{strongest_score:.0f}分），{signal_verb}。"
        )
    elif overall_score >= 40:
        interpretation = (
            f"量化模型显示{asset_code}综合评分{overall_score:.0f}分，处于{score_position}，"
            f"多空信号均衡，{signal_verb}，建议保持观望。"
        )
    else:
        interpretation = (
            f"量化模型显示{asset_code}综合评分{overall_score:.0f}分，{signal_label}，"
            f"{weakest_name}维度拖累（{weakest_score:.0f}分），当前不建议加仓。"
        )

    # Build risk_factors
    risk_factors: list[str] = []

    if d1_score is not None and d1_score < 45:
        risk_factors.append("实际利率维持高位，对黄金压制未消")
    elif d1_score is not None and d1_score > 65:
        risk_factors.append("若联储转鹰，降息预期逆转将打压金价")
    else:
        risk_factors.append("宏观利率方向尚未明朗，需关注美联储动向")

    if d2_score is not None and d2_score < 45:
        risk_factors.append("美元阶段性走强可能压制黄金上行空间")
    elif d2_score is not None and d2_score > 65:
        risk_factors.append("美元反弹风险将削弱黄金相对吸引力")
    else:
        risk_factors.append("全球流动性变化需持续跟踪")

    if d4_score is not None and d4_score < 40:
        risk_factors.append("技术面超卖修复未完成，短期回调风险犹存")

    # Keep exactly 2-3 risk factors
    risk_factors = risk_factors[:3]
    if len(risk_factors) < 2:
        risk_factors.append("市场波动加大，建议控制仓位")

    # Build regime_summary
    if overall_score >= 65:
        regime_summary = f"当前宏观环境整体有利于{asset_code}资产配置，量化信号维持看多区间。"
    elif overall_score >= 50:
        regime_summary = f"当前宏观体制中性，{asset_code}缺乏明确方向催化，建议等待信号明朗。"
    else:
        regime_summary = f"当前宏观环境对{asset_code}不利，量化评分处于看空区间，建议谨慎。"

    # Build major_change_recap (only when |delta| > 20)
    major_change_recap: str | None = None
    if score_delta is not None and abs(score_delta) > 20:
        direction = "大幅上升" if score_delta > 0 else "大幅下降"
        major_change_recap = (
            f"本期{asset_code}综合评分较上期{direction}{abs(score_delta):.0f}分，"
            f"跨越一个信号级别。量化模型数据驱动更新，建议重新评估仓位配置。"
            f"模板生成摘要仅供参考，详细研判需待 LLM 分析恢复。"
        )

    # Build change_summary
    change_summary: str | None = None
    if prev_score is not None and score_delta is not None:
        # We don't have per-dimension deltas in the template context,
        # so we generate a simplified overall change summary
        direction_str = f"+{score_delta:.0f}" if score_delta > 0 else f"{score_delta:.0f}"
        change_summary = f"综合{direction_str}(量化更新)"

    return InterpretationTemplateResult(
        market_interpretation=interpretation,
        risk_factors=risk_factors,
        regime_summary=regime_summary,
        major_change_recap=major_change_recap,
        change_summary=change_summary,
    )
