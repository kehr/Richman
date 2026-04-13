"""English degradation templates for market interpretation.

Mirrors interpretation_zh.py but generates English text. Used when the
interpretation agent fails or is unavailable and language='en' is requested.

Generated output is marked with generated_by='l1_only' by the pipeline.
"""

from __future__ import annotations

from richson.templates.interpretation_zh import InterpretationTemplateResult

# ---------------------------------------------------------------------------
# Signal level -> label mapping (English)
# ---------------------------------------------------------------------------

_SIGNAL_LABELS: dict[str, str] = {
    "strong_bullish": "strongly bullish",
    "moderate_bullish": "moderately bullish",
    "neutral": "neutral",
    "moderate_bearish": "moderately bearish",
    "strong_bearish": "strongly bearish",
}

_SIGNAL_VERBS: dict[str, str] = {
    "strong_bullish": "signaling strong allocation opportunity",
    "moderate_bullish": "signaling positive allocation bias",
    "neutral": "showing balanced signals",
    "moderate_bearish": "signaling caution on allocation",
    "strong_bearish": "signaling poor near-term setup",
}

_SCORE_RANGE_LABEL: dict[tuple[float, float], str] = {
    (80, 100): "high range",
    (60, 80): "upper-mid range",
    (40, 60): "mid range",
    (20, 40): "lower-mid range",
    (0, 20): "low range",
}

_DIMENSION_NAMES: dict[str, str] = {
    "D1": "Macro Rates",
    "D2": "Dollar/Liquidity",
    "D3": "Structural Demand",
    "D4": "Technical Position",
}


def _score_to_range_label(score: float) -> str:
    for (low, high), label in _SCORE_RANGE_LABEL.items():
        if low <= score < high:
            return label
    return "high range" if score >= 80 else "low range"


def _strongest_dimension(d_scores: dict[str, float | None]) -> tuple[str, float]:
    valid = {k: v for k, v in d_scores.items() if v is not None}
    if not valid:
        return ("overall", 50.0)
    best_key = max(valid, key=lambda k: valid[k])  # type: ignore[arg-type]
    return (_DIMENSION_NAMES.get(best_key, best_key), valid[best_key])  # type: ignore[return-value]


def _weakest_dimension(d_scores: dict[str, float | None]) -> tuple[str, float]:
    valid = {k: v for k, v in d_scores.items() if v is not None}
    if not valid:
        return ("overall", 50.0)
    worst_key = min(valid, key=lambda k: valid[k])  # type: ignore[arg-type]
    return (_DIMENSION_NAMES.get(worst_key, worst_key), valid[worst_key])  # type: ignore[return-value]


# ---------------------------------------------------------------------------
# Main render function
# ---------------------------------------------------------------------------


def render_interpretation_en(
    overall_score: float,
    signal_level: str,
    d1_score: float | None = None,
    d2_score: float | None = None,
    d3_score: float | None = None,
    d4_score: float | None = None,
    prev_score: float | None = None,
    score_delta: float | None = None,
    asset_code: str = "Gold",
) -> InterpretationTemplateResult:
    """Render English market interpretation from quantitative scores.

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
    signal_label = _SIGNAL_LABELS.get(signal_level, "neutral")
    signal_verb = _SIGNAL_VERBS.get(signal_level, "showing balanced signals")
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
            f"Quantitative model scores {asset_code} at {overall_score:.0f}/100, "
            f"{signal_label}. {strongest_name} provides the strongest support "
            f"({strongest_score:.0f}/100), {signal_verb}."
        )
    elif overall_score >= 40:
        interpretation = (
            f"Quantitative model scores {asset_code} at {overall_score:.0f}/100, "
            f"in the {score_position}. Bullish and bearish signals are balanced; "
            f"{signal_verb}. Wait for clearer directional confirmation."
        )
    else:
        interpretation = (
            f"Quantitative model scores {asset_code} at {overall_score:.0f}/100, "
            f"{signal_label}. {weakest_name} is the primary drag "
            f"({weakest_score:.0f}/100). Adding positions is not recommended."
        )

    # Build risk_factors
    risk_factors: list[str] = []

    if d1_score is not None and d1_score < 45:
        risk_factors.append("Elevated real rates continue to weigh on gold")
    elif d1_score is not None and d1_score > 65:
        risk_factors.append("Fed hawkish pivot would reverse rate-cut expectations")
    else:
        risk_factors.append("Macro rate direction unclear; monitor Fed communications")

    if d2_score is not None and d2_score < 45:
        risk_factors.append("Dollar strength limits gold's near-term upside")
    elif d2_score is not None and d2_score > 65:
        risk_factors.append("Dollar rebound risk could reduce gold's relative appeal")
    else:
        risk_factors.append("Global liquidity conditions require continued monitoring")

    if d4_score is not None and d4_score < 40:
        risk_factors.append("Technical oversold condition: short-term pullback risk remains")

    risk_factors = risk_factors[:3]
    if len(risk_factors) < 2:
        risk_factors.append("Elevated market volatility: manage position size accordingly")

    # Build regime_summary
    if overall_score >= 65:
        regime_summary = (
            f"The current macro environment is broadly supportive for {asset_code} "
            f"allocation; quantitative signals remain in bullish territory."
        )
    elif overall_score >= 50:
        regime_summary = (
            f"Current macro regime is neutral; {asset_code} lacks a clear directional "
            f"catalyst. Wait for signals to clarify before increasing exposure."
        )
    else:
        regime_summary = (
            f"Current macro environment is unfavorable for {asset_code}; "
            f"quantitative scores are in bearish territory. Exercise caution."
        )

    # Build major_change_recap (only when |delta| > 20)
    major_change_recap: str | None = None
    if score_delta is not None and abs(score_delta) > 20:
        direction = "risen" if score_delta > 0 else "fallen"
        major_change_recap = (
            f"{asset_code} composite score has {direction} sharply by {abs(score_delta):.0f} points "
            f"vs. the prior analysis, crossing a signal threshold. "
            f"The quantitative model has updated based on new data inputs; "
            f"consider reassessing position sizing. "
            f"This summary is template-generated — detailed LLM analysis unavailable."
        )

    # Build change_summary
    change_summary: str | None = None
    if prev_score is not None and score_delta is not None:
        direction_str = f"+{score_delta:.0f}" if score_delta > 0 else f"{score_delta:.0f}"
        change_summary = f"overall{direction_str}(quant update)"

    return InterpretationTemplateResult(
        market_interpretation=interpretation,
        risk_factors=risk_factors,
        regime_summary=regime_summary,
        major_change_recap=major_change_recap,
        change_summary=change_summary,
    )
