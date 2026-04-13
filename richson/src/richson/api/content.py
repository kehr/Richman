"""Content generation endpoints.

POST /content/weekly-insight - generate weekly investment insight text via LLM
"""

from __future__ import annotations

import json
import uuid
from datetime import UTC, datetime

import structlog
from fastapi import APIRouter, Depends, Request
from pydantic import BaseModel, Field

from richson.api.auth import require_api_key
from richson.schemas.common import LLMConfig

router = APIRouter(prefix="/content", dependencies=[Depends(require_api_key)])
logger = structlog.get_logger()


class WeeklyInsightBody(BaseModel):
    locale: str = Field(default="zh", pattern="^(zh|en)$")
    llm_config: LLMConfig = Field(alias="llmConfig")
    request_id: uuid.UUID | None = Field(default=None, alias="requestId")

    model_config = {"populate_by_name": True}


@router.post("/weekly-insight")
async def generate_weekly_insight(
    body: WeeklyInsightBody,
    request: Request,
) -> dict:
    """Generate weekly investment insight text (LLM-driven).

    Called by richman cron every Monday. Synchronous; expects response within 30s.

    Returns:
        weeklyReview: last week's summary
        weeklyOutlook: this week's key themes
        educationTopic: educational content
        locale: requested locale
    """
    from richson.core.pipeline import _resolve_model, _run_agent  # noqa: PLC0415
    from richson.db import repository as repo  # noqa: PLC0415
    from richson.main import get_session_factory  # noqa: PLC0415

    session_factory = get_session_factory()
    log = logger.bind(locale=body.locale, request_id=str(body.request_id) if body.request_id else None)
    log.info("weekly_insight_start")

    # Load latest gold analysis as context
    async with session_factory() as sess:
        latest = await repo.get_latest_asset_analysis(sess, "GLD", body.locale)

    context = {
        "locale": body.locale,
        "weekOf": datetime.now(tz=UTC).strftime("%Y-%m-%d"),
        "latestAnalysis": None,
    }
    if latest is not None:
        context["latestAnalysis"] = {
            "overallScore": float(latest.overall_score),
            "signalLevel": latest.signal_level,
            "scoreDelta": float(latest.score_delta) if latest.score_delta is not None else None,
            "changeSummary": latest.change_summary,
            "marketInterpretation": latest.market_interpretation,
        }

    # Build LLM prompt
    if body.locale == "zh":
        prompt = (
            f"请基于以下黄金分析数据，生成本周投研洞察内容（约300-500字）。\n\n"
            f"数据上下文：{json.dumps(context, ensure_ascii=False)}\n\n"
            "请输出JSON格式，包含以下字段：\n"
            "- weeklyReview: 上周黄金市场回顾（2-3句话）\n"
            "- weeklyOutlook: 本周关键主题和关注点（2-3句话）\n"
            "- educationTopic: 投资教育内容，介绍一个黄金相关概念（3-4句话）"
        )
    else:
        prompt = (
            f"Based on the following gold analysis data, generate a weekly investment insight (300-500 words).\n\n"
            f"Context: {json.dumps(context, ensure_ascii=False)}\n\n"
            "Output JSON with these fields:\n"
            "- weeklyReview: last week's gold market review (2-3 sentences)\n"
            "- weeklyOutlook: this week's key themes and watchpoints (2-3 sentences)\n"
            "- educationTopic: educational content on a gold-related concept (3-4 sentences)"
        )

    # Create a simple LLM agent for content generation
    result = None
    try:
        from google.adk.agents import Agent  # noqa: PLC0415

        model = _resolve_model(body.llm_config)
        agent = Agent(
            name="weekly_insight_agent",
            model=model,
            instruction="You are a professional gold investment analyst. Generate concise, professional insights.",
            tools=[],
        )
        result = await _run_agent(agent, prompt, timeout_s=60.0)
    except Exception as exc:
        log.error("weekly_insight_llm_error", error=str(exc))

    if result is None:
        # Fallback to template content
        result = _fallback_weekly_insight(context, body.locale)

    log.info("weekly_insight_complete")

    return {
        "data": {
            "weeklyReview": result.get("weeklyReview", result.get("weekly_review", "")),
            "weeklyOutlook": result.get("weeklyOutlook", result.get("weekly_outlook", "")),
            "educationTopic": result.get("educationTopic", result.get("education_topic", "")),
            "locale": body.locale,
        }
    }


def _fallback_weekly_insight(context: dict, locale: str) -> dict:
    """Template-based weekly insight fallback when LLM is unavailable."""
    analysis = context.get("latestAnalysis") or {}
    score = analysis.get("overallScore", 50.0)
    signal = analysis.get("signalLevel", "neutral")

    if locale == "zh":
        signal_label = {
            "strong_bullish": "强烈看多",
            "moderate_bullish": "温和看多",
            "neutral": "中性",
            "moderate_bearish": "温和看空",
            "strong_bearish": "强烈看空",
        }.get(signal, "中性")
        return {
            "weeklyReview": (
                f"上周黄金市场综合评分{score:.0f}分，信号方向{signal_label}。"
                "量化模型综合宏观利率、美元流动性、结构性需求和技术位置四个维度评估。"
            ),
            "weeklyOutlook": (
                "本周关注美联储官员讲话及宏观数据发布对黄金走势的影响。"
                "建议依据最新量化评分调整仓位，维持动态再平衡策略。"
            ),
            "educationTopic": (
                "黄金与实际利率的反向关系：当TIPS收益率（剔除通胀后的实际利率）下降时，"
                "持有黄金的机会成本降低，通常对黄金价格形成支撑。"
                "这是量化模型D1维度最重要的子指标之一。"
            ),
        }
    else:
        return {
            "weeklyReview": (
                f"Last week's gold quantitative score was {score:.0f}/100 with a {signal} signal. "
                "The model evaluates macro rates, dollar liquidity, structural demand, and technical position."
            ),
            "weeklyOutlook": (
                "This week, monitor Fed official speeches and macro data releases for gold trend signals. "
                "Adjust positions according to the latest quantitative score and maintain dynamic rebalancing."
            ),
            "educationTopic": (
                "Gold's inverse relationship with real interest rates: when TIPS yields (real rates after inflation) decline, "
                "the opportunity cost of holding gold decreases, typically supporting gold prices. "
                "This is the most important sub-indicator in the D1 dimension of our quantitative model."
            ),
        }
