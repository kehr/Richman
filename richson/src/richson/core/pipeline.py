"""L1 -> L2 -> L3 analysis pipeline orchestration.

Coordinates the three-layer analysis for a given asset:
- Layer 1: Quantitative scoring (always runs)
- Layer 2: LLM research per dimension (D1/D2/D3) - skipped on failure
- Layer 3: Interpretation + text generation - falls back to templates on failure

G1.3 (asyncio blocking): pandas computation is offloaded via asyncio.to_thread.
G1.5 (LLM budget): daily token spending tracked; exceeds budget -> l1_only mode.
G1.6 (locale params): locale is passed to template and execution agent.
"""

from __future__ import annotations

import asyncio
import json
import time
import uuid
from datetime import UTC, datetime
from decimal import Decimal
from typing import Any

import structlog

from richson.core.adjustment import (
    LLMAdjustmentEvent,
    apply_adjustment_to_score,
    compute_adjustment,
)
from richson.core.confidence import (
    check_fred_freshness,
    compute_confidence,
    compute_data_coverage_label,
)
from richson.core.conflict import check_llm_anomaly, detect_conflict
from richson.core.drawdown import compute_drawdown_reference
from richson.core.scoring import compute_overall_score, signal_level_from_score

logger = structlog.get_logger()

# ---------------------------------------------------------------------------
# Risk preference constants (TRD SS15)
# ---------------------------------------------------------------------------

RISK_PARAMS: dict[str, dict[str, float]] = {
    "conservative": {
        "max_single_add": 2.0,
        "stop_loss_atr_multi": 1.5,
        "concentration_blue": 10.0,
        "score_threshold_add": 70.0,
    },
    "moderate": {
        "max_single_add": 5.0,
        "stop_loss_atr_multi": 2.0,
        "concentration_blue": 15.0,
        "score_threshold_add": 60.0,
    },
    "aggressive": {
        "max_single_add": 8.0,
        "stop_loss_atr_multi": 3.0,
        "concentration_blue": 20.0,
        "score_threshold_add": 55.0,
    },
}

# Demo plan fixed parameters (TRD SS5.3)
_DEMO_POSITION_RATIO = 10.0
_DEMO_COST_PRICE_FACTOR = 0.95
_DEMO_RISK_PREFERENCE = "moderate"
_DEMO_PEER_EXPOSURE = 10.0


# ---------------------------------------------------------------------------
# LLM agent runner
# ---------------------------------------------------------------------------


def _resolve_model(llm_config: Any) -> Any:
    """Resolve LLM config to ADK model string or LiteLlm instance."""
    provider = llm_config.provider
    model = llm_config.model
    api_key = llm_config.api_key or ""

    if provider == "gemini":
        return model

    try:
        from google.adk.models.lite_llm import LiteLlm  # noqa: PLC0415

        if provider == "claude":
            prefix = "anthropic/"
        elif provider == "openai":
            prefix = "openai/"
        elif provider == "openai_compatible":
            api_base = llm_config.api_base
            return LiteLlm(model=f"openai/{model}", api_base=api_base, api_key=api_key)
        else:
            prefix = ""

        full_model = f"{prefix}{model}" if prefix else model
        return LiteLlm(model=full_model, api_key=api_key)
    except Exception:
        # ADK not installed or provider unknown - return model string as fallback
        return model


async def _run_agent(agent: Any, user_input: str, timeout_s: float = 60.0) -> dict | None:
    """Execute an ADK agent via InMemoryRunner and return parsed output.

    Returns None on timeout or error (caller handles degradation).
    """
    try:
        from google.adk.runners import InMemoryRunner  # noqa: PLC0415
        from google.genai import types as genai_types  # noqa: PLC0415

        runner = InMemoryRunner(agent=agent, app_name="richson")
        session_id = str(uuid.uuid4())
        # InMemoryRunner.run_async is a sync generator in some versions;
        # wrap in to_thread to avoid blocking the event loop
        content = genai_types.Content(
            role="user",
            parts=[genai_types.Part(text=user_input)],
        )

        async def _collect() -> dict | None:
            output_text = None
            async for event in runner.run_async(
                user_id="pipeline",
                session_id=session_id,
                new_message=content,
            ):
                if hasattr(event, "content") and event.content:
                    for part in event.content.parts:
                        if hasattr(part, "text") and part.text:
                            output_text = part.text
            if output_text is None:
                return None
            return json.loads(output_text)

        result = await asyncio.wait_for(_collect(), timeout=timeout_s)
        return result
    except TimeoutError:
        logger.warning("agent_run_timeout", timeout_s=timeout_s)
        return None
    except Exception as exc:
        logger.warning("agent_run_error", error=str(exc))
        return None


# ---------------------------------------------------------------------------
# Layer 1: quantitative scoring
# ---------------------------------------------------------------------------


async def run_layer1_gold(asset_code: str, llm_config: Any) -> dict[str, Any]:
    """Run Layer 1 quantitative scoring for gold assets.

    All data fetching is done in asyncio.to_thread to avoid blocking (G1.3).

    Returns a dict with keys:
    - sub_indicators: list of sub-indicator dicts per dimension
    - dimension_results: {d1, d2, d3, d4} -> {sub_indicators, base_score, ...}
    - price: current price float
    - ohlcv_df: pandas DataFrame
    - t10y2y_series: pandas Series
    - vix_df: pandas DataFrame
    - fred_last_date: date | None
    - polymarket_available: bool
    - rate_cut_probability: float | None
    """
    from richson.config import settings  # noqa: PLC0415
    from richson.core.indicators.d1_macro_rates import compute_d1_indicators  # noqa: PLC0415
    from richson.core.indicators.d2_dollar_liquidity import compute_d2_indicators  # noqa: PLC0415
    from richson.core.indicators.d3_structural_demand import compute_d3_indicators  # noqa: PLC0415
    from richson.core.indicators.d4_technical_position import compute_d4_indicators  # noqa: PLC0415
    from richson.datasources.cot import COTClient  # noqa: PLC0415
    from richson.datasources.fred import FREDClient  # noqa: PLC0415
    from richson.datasources.polymarket import PolymarketClient  # noqa: PLC0415
    from richson.datasources.stooq import StooqClient  # noqa: PLC0415
    from richson.datasources.wgc import WGCClient  # noqa: PLC0415
    from richson.datasources.yahoo import YahooFinanceClient  # noqa: PLC0415

    fred_client = FREDClient(api_key=settings.fred_api_key)
    yahoo_client = YahooFinanceClient()
    stooq_client = StooqClient()
    poly_client = PolymarketClient()
    cot_client = COTClient()
    wgc_client = WGCClient()

    # Fetch data concurrently in thread pool (pandas/IO is blocking)
    def _fetch_all() -> dict[str, Any]:
        all_fred = fred_client.get_all_series()
        fedfunds = all_fred.get("FEDFUNDS")
        t10y2y = all_fred.get("T10Y2Y")
        dfii10 = all_fred.get("DFII10")
        dgs10 = all_fred.get("DGS10")
        m2sl = all_fred.get("M2SL")

        # Gold price - yahoo primary, stooq fallback
        ohlcv = yahoo_client.get_ohlcv(asset_code)
        if ohlcv is None or (hasattr(ohlcv, "empty") and ohlcv.empty):
            ohlcv = stooq_client.get_ohlcv(asset_code)

        vix_ohlcv = yahoo_client.get_vix()
        dxy_ohlcv = yahoo_client.get_dxy()

        # COT data for D3
        cot_data = cot_client.get_gold_managed_money()

        # WGC data for D3
        try:
            wgc_data = wgc_client.get_quarterly_data()
        except Exception:
            wgc_data = None

        # Current gold price for AISC margin calc
        current_price = None
        if ohlcv is not None and not (hasattr(ohlcv, "empty") and ohlcv.empty):
            close_col = "Close" if "Close" in ohlcv.columns else "close"
            if close_col in ohlcv.columns:
                current_price = float(ohlcv[close_col].dropna().iloc[-1])

        return {
            "fedfunds": fedfunds,
            "t10y2y": t10y2y,
            "dfii10": dfii10,
            "dgs10": dgs10,
            "m2sl": m2sl,
            "ohlcv": ohlcv,
            "vix_ohlcv": vix_ohlcv,
            "dxy_ohlcv": dxy_ohlcv,
            "cot_data": cot_data,
            "wgc_data": wgc_data,
            "current_price": current_price,
        }

    def _fetch_polymarket() -> tuple[float | None, float | None, bool]:
        try:
            rate_cut_prob = poly_client.get_rate_cut_probability()
            geo_risk_prob = poly_client.get_geopolitical_risk_index()
            return rate_cut_prob, geo_risk_prob, True
        except Exception:
            return None, None, False

    data, (rate_cut_prob, geo_risk_prob, poly_ok) = await asyncio.gather(
        asyncio.to_thread(_fetch_all),
        asyncio.to_thread(_fetch_polymarket),
    )

    fedfunds = data["fedfunds"]
    t10y2y = data["t10y2y"]
    dfii10 = data["dfii10"]
    dgs10 = data["dgs10"]
    m2sl = data["m2sl"]
    ohlcv = data["ohlcv"]
    vix_ohlcv = data["vix_ohlcv"]
    dxy_ohlcv = data["dxy_ohlcv"]
    cot_data = data["cot_data"]
    wgc_data = data["wgc_data"]
    current_price = data["current_price"]

    # Compute dimension indicators in thread pool
    def _compute_dims() -> dict[str, Any]:
        d1 = compute_d1_indicators(fedfunds, t10y2y, dfii10, dgs10, rate_cut_prob)
        d2 = compute_d2_indicators(dxy_ohlcv, m2sl, None)  # TGA not fetched in MVP
        d3 = compute_d3_indicators(cot_data, wgc_data, ohlcv, current_price, None, geo_risk_prob)
        d4 = compute_d4_indicators(ohlcv)
        return {"d1": d1, "d2": d2, "d3": d3, "d4": d4}

    dim_results = await asyncio.to_thread(_compute_dims)

    # Determine FRED freshness
    fred_last_date = None
    if t10y2y is not None and not t10y2y.empty:
        last_idx = t10y2y.dropna().index
        if len(last_idx):
            fred_last_date = last_idx[-1].date()

    # Current price (already computed during fetch; use directly)
    price = current_price

    return {
        "dimension_results": dim_results,
        "price": price,
        "ohlcv": ohlcv,
        "t10y2y": t10y2y,
        "vix_ohlcv": vix_ohlcv,
        "fred_last_date": fred_last_date,
        "polymarket_available": poly_ok,
        "rate_cut_probability": rate_cut_prob,
    }


def _build_dimension_scores(dim_results: dict[str, Any]) -> dict[str, float | None]:
    """Extract base scores from dimension results."""
    return {
        "d1": dim_results["d1"].get("base_score"),
        "d2": dim_results["d2"].get("base_score"),
        "d3": dim_results["d3"].get("base_score"),
        "d4": dim_results["d4"].get("base_score"),
    }


# ---------------------------------------------------------------------------
# Layer 2: LLM research per dimension
# ---------------------------------------------------------------------------


async def run_layer2(
    dimension: str,
    base_score: float,
    llm_config: Any,
    timeout_s: float = 60.0,
) -> dict[str, Any] | None:
    """Run Layer 2 research agent for a single dimension (D1/D2/D3).

    Returns structured research result dict, or None on failure.
    """
    try:
        from richson.agents.research_agent import (  # noqa: PLC0415
            ResearchResult,
            create_research_agent,
        )

        agent = create_research_agent(dimension)  # type: ignore[arg-type]
        # Inject model
        model = _resolve_model(llm_config)
        agent.model = model  # type: ignore[attr-defined]

        user_input = (
            f"Dimension: {dimension}\n"
            f"Current quantitative base score: {base_score:.1f}/100\n"
            f"Research current macroeconomic conditions affecting this dimension for gold."
        )

        result_dict = await _run_agent(agent, user_input, timeout_s=timeout_s)
        if result_dict is None:
            return None

        # Validate via pydantic schema
        result = ResearchResult.model_validate(result_dict)
        result = result.validate_major_magnitude()
        return result.model_dump()
    except Exception as exc:
        logger.warning("layer2_error", dimension=dimension, error=str(exc))
        return None


# ---------------------------------------------------------------------------
# Layer 3: interpretation + execution plan
# ---------------------------------------------------------------------------


async def run_layer3_interpretation(
    overall_score: float,
    signal_level: str,
    dimension_scores: dict[str, float | None],
    research_summaries: dict[str, Any],
    locale: str,
    llm_config: Any,
    prev_analysis: Any | None = None,
    timeout_s: float = 60.0,
) -> dict[str, Any] | None:
    """Run Layer 3 interpretation agent.

    Returns dict with market_interpretation, risk_factors, regime_summary, etc.
    Returns None on failure (caller uses template fallback).
    """
    try:
        from richson.agents.interpretation_agent import create_interpretation_agent  # noqa: PLC0415

        agent = create_interpretation_agent()
        model = _resolve_model(llm_config)
        agent.model = model  # type: ignore[attr-defined]

        prev_score = None
        if prev_analysis is not None:
            prev_score = float(prev_analysis.overall_score)

        score_delta = None
        if prev_score is not None:
            score_delta = round(overall_score - prev_score, 2)

        context = {
            "overall_score": overall_score,
            "signal_level": signal_level,
            "locale": locale,
            "dimension_scores": dimension_scores,
            "research_summaries": research_summaries,
            "prev_score": prev_score,
            "score_delta": score_delta,
        }

        user_input = f"Generate market interpretation. Context: {json.dumps(context, ensure_ascii=False)}"
        result_dict = await _run_agent(agent, user_input, timeout_s=timeout_s)
        return result_dict
    except Exception as exc:
        logger.warning("layer3_interpretation_error", error=str(exc))
        return None


async def run_layer3_execution(
    asset_code: str,
    overall_score: float,
    dimension_scores: dict[str, float | None],
    holding_info: dict[str, Any],
    risk_preference: str,
    peer_exposure: float,
    support_levels: list[float],
    resistance_levels: list[float],
    language: str,
    llm_config: Any,
    timeout_s: float = 60.0,
) -> dict[str, Any] | None:
    """Run Layer 3 execution agent to generate execution plan.

    Returns dict matching ExecutionPlanOutput schema, or None on failure.
    """
    try:
        from richson.agents.execution_agent import create_execution_agent  # noqa: PLC0415

        agent = create_execution_agent()
        model = _resolve_model(llm_config)
        agent.model = model  # type: ignore[attr-defined]

        risk_params = RISK_PARAMS.get(risk_preference, RISK_PARAMS["moderate"])

        context = {
            "asset_code": asset_code,
            "overall_score": overall_score,
            "signal_level": signal_level_from_score(overall_score),
            "dimension_scores": dimension_scores,
            "holding": holding_info,
            "risk_preference": risk_preference,
            "risk_params": risk_params,
            "peer_exposure": peer_exposure,
            "support_levels": support_levels,
            "resistance_levels": resistance_levels,
            "language": language,
        }

        user_input = f"Generate execution plan. Context: {json.dumps(context, ensure_ascii=False)}"
        result_dict = await _run_agent(agent, user_input, timeout_s=timeout_s)
        return result_dict
    except Exception as exc:
        logger.warning("layer3_execution_error", error=str(exc))
        return None


# ---------------------------------------------------------------------------
# Support/resistance helper
# ---------------------------------------------------------------------------


def _get_support_resistance(ohlcv: Any) -> tuple[list[float], list[float]]:
    """Extract support/resistance levels from OHLCV data."""
    try:
        from richson.core.support_resistance import compute_support_resistance  # noqa: PLC0415

        if ohlcv is None or (hasattr(ohlcv, "empty") and ohlcv.empty):
            return [], []
        # Normalize column names to lowercase
        df = ohlcv.copy()
        df.columns = [c.lower() for c in df.columns]
        close_series = df["close"].dropna()
        if close_series.empty:
            return [], []
        sma200 = float(close_series.rolling(200).mean().iloc[-1]) if len(close_series) >= 200 else float(close_series.mean())
        result = compute_support_resistance(df, sma200)
        return result.get("support_levels", []), result.get("resistance_levels", [])
    except Exception:
        return [], []


# ---------------------------------------------------------------------------
# Full asset analysis pipeline (Mode A)
# ---------------------------------------------------------------------------


async def run_asset_analysis_pipeline(
    job_id: uuid.UUID,
    asset_code: str,
    locale: str,
    llm_config: Any,
    session_factory: Any,
    request_id: uuid.UUID | None = None,
    model_version: str = "gold_v1.0",
    generated_by_override: str | None = None,
    budget_exceeded: bool = False,
) -> None:
    """Full L1->L2->L3 pipeline for a single asset (async background task).

    Updates rs_analysis_jobs progress throughout and persists final result
    to rs_asset_analyses + rs_asset_analysis_dimensions.

    Args:
        job_id: UUID of the rs_analysis_jobs record.
        asset_code: asset ticker symbol.
        locale: language locale for L3 text output.
        llm_config: LLM provider configuration.
        session_factory: SQLAlchemy async_sessionmaker.
        request_id: optional upstream request UUID for tracing.
        model_version: model version string for this analysis.
        generated_by_override: force 'l1_only' or 'backfill' mode.
        budget_exceeded: if True, skip L2/L3 (budget cap).
    """
    from richson.db import repository as repo  # noqa: PLC0415

    log = logger.bind(
        job_id=str(job_id),
        asset_code=asset_code,
        request_id=str(request_id) if request_id else None,
    )

    # Step tracking helpers
    steps: list[dict] = [
        {"name": "data_fetch", "status": "pending", "durationMs": None},
        {"name": "layer1_scoring", "status": "pending", "durationMs": None},
        {"name": "layer2_d1", "status": "pending", "durationMs": None},
        {"name": "layer2_d2", "status": "pending", "durationMs": None},
        {"name": "layer2_d3", "status": "pending", "durationMs": None},
        {"name": "layer3_interpretation", "status": "pending", "durationMs": None},
        {"name": "persist", "status": "pending", "durationMs": None},
    ]

    def step_index(name: str) -> int:
        for i, s in enumerate(steps):
            if s["name"] == name:
                return i
        return -1

    async def update_step(name: str, status: str, duration_ms: int | None = None) -> None:
        idx = step_index(name)
        if idx >= 0:
            steps[idx]["status"] = status
            if duration_ms is not None:
                steps[idx]["durationMs"] = duration_ms
        progress = sum(1 for s in steps if s["status"] == "completed") / len(steps)
        async with session_factory() as sess:
            await repo.update_job_status(
                sess,
                job_id=job_id,
                status="running",
                current_step=name,
                progress=progress,
                steps=steps,
            )
            await sess.commit()

    try:
        # Mark job running
        async with session_factory() as sess:
            await repo.update_job_status(sess, job_id=job_id, status="running", steps=steps)
            await sess.commit()

        # ----------------------------------------------------------------
        # Step 1: data_fetch + Layer 1 scoring
        # ----------------------------------------------------------------
        log.info("pipeline_step_start", step="data_fetch")
        await update_step("data_fetch", "running")
        t0 = time.perf_counter()

        l1_data = await run_layer1_gold(asset_code, llm_config)

        fetch_ms = int((time.perf_counter() - t0) * 1000)
        await update_step("data_fetch", "completed", fetch_ms)
        log.info("pipeline_step_done", step="data_fetch", duration_ms=fetch_ms)

        # ----------------------------------------------------------------
        # Layer 1 scoring
        # ----------------------------------------------------------------
        await update_step("layer1_scoring", "running")
        t0 = time.perf_counter()

        dim_results = l1_data["dimension_results"]
        base_scores = _build_dimension_scores(dim_results)

        # Compute overall score from base scores
        available_base = {k: v for k, v in base_scores.items() if v is not None}
        if len(available_base) < 2:
            log.error("insufficient_dimensions", available=list(available_base.keys()))
            async with session_factory() as sess:
                await repo.update_job_status(
                    sess,
                    job_id=job_id,
                    status="failed",
                    error_code="INSUFFICIENT_HISTORY",
                    error_message="Less than 2 dimensions available",
                )
                await sess.commit()
            return

        dimension_weights = {"d1": 0.30, "d2": 0.25, "d3": 0.25, "d4": 0.20}
        overall_base = compute_overall_score(base_scores, dimension_weights)

        score_ms = int((time.perf_counter() - t0) * 1000)
        await update_step("layer1_scoring", "completed", score_ms)
        log.info("pipeline_step_done", step="layer1_scoring", duration_ms=score_ms, overall_base=overall_base)

        # ----------------------------------------------------------------
        # Layer 2: LLM research (skip if budget exceeded or l1_only mode)
        # ----------------------------------------------------------------
        llm_skipped = budget_exceeded or generated_by_override == "l1_only"
        llm_adjustments: dict[str, float] = {}
        research_results: dict[str, Any] = {}

        for dim in ["D1", "D2", "D3"]:
            step_name = f"layer2_{dim.lower()}"
            if llm_skipped:
                await update_step(step_name, "skipped")
                continue

            await update_step(step_name, "running")
            t0 = time.perf_counter()
            log.info("pipeline_step_start", step=step_name)

            base_score_for_dim = base_scores.get(dim.lower(), 50.0) or 50.0
            research = await run_layer2(dim, base_score_for_dim, llm_config)
            dim_ms = int((time.perf_counter() - t0) * 1000)

            if research is None:
                await update_step(step_name, "failed", dim_ms)
                log.warning("layer2_failed", dimension=dim, duration_ms=dim_ms)
                llm_skipped = True  # subsequent dims also skipped
            else:
                await update_step(step_name, "completed", dim_ms)
                log.info("pipeline_step_done", step=step_name, duration_ms=dim_ms)
                research_results[dim] = research
                # Map LLM judgment to numeric adjustment
                judgment = research.get("judgment", {})
                events = [
                    LLMAdjustmentEvent(
                        dimension=dim,
                        direction=judgment.get("direction", "neutral"),
                        magnitude=judgment.get("magnitude", "minor"),
                        confidence=judgment.get("confidence", "low"),
                        source_count=len(research.get("events", [])),
                    )
                ]
                adj_map = compute_adjustment(events)
                llm_adjustments[dim] = adj_map.get(dim, 0.0)

        # Apply LLM adjustments to build final dimension scores
        final_scores: dict[str, float | None] = {}
        llm_adjustment_values: dict[str, float] = {}
        for dim_key in ["d1", "d2", "d3", "d4"]:
            base = base_scores.get(dim_key)
            if base is None:
                final_scores[dim_key] = None
                llm_adjustment_values[dim_key] = 0.0
                continue
            adj = llm_adjustments.get(dim_key.upper(), 0.0)
            final_scores[dim_key] = apply_adjustment_to_score(base, adj)
            llm_adjustment_values[dim_key] = adj

        # Recompute overall with adjusted scores
        overall_score = compute_overall_score(final_scores, dimension_weights)
        signal_level = signal_level_from_score(overall_score)

        # ----------------------------------------------------------------
        # Confidence + conflict + data coverage
        # ----------------------------------------------------------------
        fred_fresh = check_fred_freshness(l1_data["fred_last_date"])
        data_completeness = {
            "fred_fresh": fred_fresh,
            "polymarket": l1_data["polymarket_available"],
        }
        confidence, band_low, band_high = compute_confidence(
            final_scores, data_completeness, not llm_skipped
        )
        data_coverage = compute_data_coverage_label(
            final_scores, fred_fresh, l1_data["polymarket_available"]
        )

        available_final = {k: v for k, v in final_scores.items() if v is not None}
        conflict_type, conflict_message = detect_conflict(available_final)

        # LLM anomaly flags (runtime only, not persisted; TRD SS8.2)
        # Computed here for logging; would be surfaced in job detail API in a future iteration
        for _dim in ["d1", "d2", "d3"]:
            if check_llm_anomaly(_dim, llm_adjustment_values.get(_dim, 0.0)):
                log.warning("llm_anomaly_flag", dimension=_dim, adjustment=llm_adjustment_values.get(_dim, 0.0))

        # ----------------------------------------------------------------
        # Previous analysis for change tracking
        # ----------------------------------------------------------------
        async with session_factory() as sess:
            prev_analysis = await repo.get_latest_asset_analysis(sess, asset_code, locale)

        score_delta = None
        prev_analysis_id = None
        if prev_analysis is not None:
            prev_analysis_id = prev_analysis.asset_analysis_id
            score_delta = round(overall_score - float(prev_analysis.overall_score), 2)

        # ----------------------------------------------------------------
        # Layer 3: interpretation
        # ----------------------------------------------------------------
        await update_step("layer3_interpretation", "running")
        t0 = time.perf_counter()
        log.info("pipeline_step_start", step="layer3_interpretation")

        interpretation_result = None
        if not llm_skipped:
            interpretation_result = await run_layer3_interpretation(
                overall_score=overall_score,
                signal_level=signal_level,
                dimension_scores=final_scores,
                research_summaries=research_results,
                locale=locale,
                llm_config=llm_config,
                prev_analysis=prev_analysis,
            )

        if interpretation_result is None:
            # Fallback to template
            if locale == "en":
                from richson.templates.interpretation_en import (
                    render_interpretation_en,  # noqa: PLC0415
                )
                tmpl = render_interpretation_en(
                    overall_score=overall_score,
                    signal_level=signal_level,
                    d1_score=final_scores.get("d1"),
                    d2_score=final_scores.get("d2"),
                    d3_score=final_scores.get("d3"),
                    d4_score=final_scores.get("d4"),
                    prev_score=float(prev_analysis.overall_score) if prev_analysis else None,
                    score_delta=score_delta,
                    asset_code=asset_code,
                )
            else:
                from richson.templates.interpretation_zh import (
                    render_interpretation_zh,  # noqa: PLC0415
                )
                tmpl = render_interpretation_zh(
                    overall_score=overall_score,
                    signal_level=signal_level,
                    d1_score=final_scores.get("d1"),
                    d2_score=final_scores.get("d2"),
                    d3_score=final_scores.get("d3"),
                    d4_score=final_scores.get("d4"),
                    prev_score=float(prev_analysis.overall_score) if prev_analysis else None,
                    score_delta=score_delta,
                    asset_code=asset_code,
                )
            market_interpretation = tmpl.market_interpretation
            risk_factors = tmpl.risk_factors
            regime_summary = tmpl.regime_summary
            change_summary = tmpl.change_summary
            major_change_recap = tmpl.major_change_recap
            generated_by = "l1_only"
        else:
            market_interpretation = interpretation_result.get("market_interpretation", "")
            risk_factors = interpretation_result.get("risk_factors", [])
            regime_summary = interpretation_result.get("regime_summary", "")
            change_summary = interpretation_result.get("change_summary")
            major_change_recap = interpretation_result.get("major_change_recap")
            generated_by = generated_by_override or "full"

        interp_ms = int((time.perf_counter() - t0) * 1000)
        await update_step("layer3_interpretation", "completed", interp_ms)
        log.info("pipeline_step_done", step="layer3_interpretation", duration_ms=interp_ms)

        # ----------------------------------------------------------------
        # Drawdown reference
        # ----------------------------------------------------------------
        import contextlib  # noqa: PLC0415

        ohlcv = l1_data["ohlcv"]
        drawdown_ref = None
        if ohlcv is not None and not (hasattr(ohlcv, "empty") and ohlcv.empty):
            with contextlib.suppress(Exception):
                drawdown_ref = await asyncio.to_thread(compute_drawdown_reference, ohlcv)

        # ----------------------------------------------------------------
        # Support/resistance levels
        # ----------------------------------------------------------------
        support_levels, resistance_levels = _get_support_resistance(ohlcv)

        # ----------------------------------------------------------------
        # Persist results (step: persist)
        # ----------------------------------------------------------------
        await update_step("persist", "running")
        t0 = time.perf_counter()
        log.info("pipeline_step_start", step="persist")

        analysis_metadata: dict[str, Any] = {}
        if drawdown_ref:
            analysis_metadata["drawdown_reference"] = drawdown_ref
        if support_levels:
            analysis_metadata["support_levels"] = support_levels
        if resistance_levels:
            analysis_metadata["resistance_levels"] = resistance_levels

        analysis_data: dict[str, Any] = {
            "asset_code": asset_code,
            "locale": locale,
            "overall_score": Decimal(str(round(overall_score, 2))),
            "signal_level": signal_level,
            "confidence": Decimal(str(round(confidence, 2))),
            "confidence_band_low": Decimal(str(round(band_low, 2))),
            "confidence_band_high": Decimal(str(round(band_high, 2))),
            "model_version": model_version,
            "market_interpretation": market_interpretation,
            "risk_factors": risk_factors,
            "regime_summary": regime_summary,
            # dimension scores
            "d1_score": Decimal(str(round(final_scores["d1"], 2))) if final_scores.get("d1") is not None else None,
            "d1_base_score": Decimal(str(round(base_scores["d1"], 2))) if base_scores.get("d1") is not None else None,
            "d1_llm_adjustment": Decimal(str(round(llm_adjustment_values.get("d1", 0.0), 2))),
            "d2_score": Decimal(str(round(final_scores["d2"], 2))) if final_scores.get("d2") is not None else None,
            "d2_base_score": Decimal(str(round(base_scores["d2"], 2))) if base_scores.get("d2") is not None else None,
            "d2_llm_adjustment": Decimal(str(round(llm_adjustment_values.get("d2", 0.0), 2))),
            "d3_score": Decimal(str(round(final_scores["d3"], 2))) if final_scores.get("d3") is not None else None,
            "d3_base_score": Decimal(str(round(base_scores["d3"], 2))) if base_scores.get("d3") is not None else None,
            "d3_llm_adjustment": Decimal(str(round(llm_adjustment_values.get("d3", 0.0), 2))),
            "d4_score": Decimal(str(round(final_scores["d4"], 2))) if final_scores.get("d4") is not None else None,
            "d4_base_score": Decimal(str(round(base_scores["d4"], 2))) if base_scores.get("d4") is not None else None,
            # weights
            "d1_weight": Decimal("0.30"),
            "d2_weight": Decimal("0.25"),
            "d3_weight": Decimal("0.25"),
            "d4_weight": Decimal("0.20"),
            # degradation markers
            "llm_skipped": llm_skipped,
            "data_coverage": data_coverage,
            # conflict
            "conflict_type": conflict_type,
            "conflict_message": conflict_message,
            # change tracking
            "prev_analysis_id": prev_analysis_id,
            "score_delta": Decimal(str(round(score_delta, 2))) if score_delta is not None else None,
            "change_summary": change_summary,
            "major_change_recap": major_change_recap,
            # price
            "price_at_analysis": Decimal(str(round(l1_data["price"], 4))) if l1_data.get("price") else None,
            "data_snapshot_at": datetime.now(tz=UTC),
            # metadata
            "analysis_metadata": analysis_metadata,
            "generated_by": generated_by,
            "source": "scheduled",
            "job_id": job_id,
        }

        async with session_factory() as sess:
            analysis = await repo.create_asset_analysis(sess, analysis_data)
            asset_analysis_id = analysis.asset_analysis_id

            # Persist dimension sub-indicators
            all_sub_indicators: list[dict[str, Any]] = []
            for dim_key in ["d1", "d2", "d3", "d4"]:
                dim_data = dim_results.get(dim_key, {})
                for sub in dim_data.get("sub_indicators", []):
                    all_sub_indicators.append({
                        "dimension": dim_key.upper(),
                        "sub_indicator": sub["name"],
                        "raw_value": Decimal(str(sub["raw_value"])) if sub.get("raw_value") is not None else None,
                        "percentile_1y": Decimal(str(sub["percentile_1y"])) if sub.get("percentile_1y") is not None else None,
                        "percentile_5y": Decimal(str(sub["percentile_5y"])) if sub.get("percentile_5y") is not None else None,
                        "blended_percentile": Decimal(str(sub["blended_percentile"])) if sub.get("blended_percentile") is not None else None,
                        "normalized_score": Decimal(str(sub["normalized_score"])) if sub.get("normalized_score") is not None else None,
                        "weight_in_dimension": Decimal(str(sub["weight_in_dimension"])) if sub.get("weight_in_dimension") is not None else None,
                        "data_source": sub.get("data_source"),
                        "data_as_of": sub.get("data_as_of"),
                    })
            if all_sub_indicators:
                await repo.bulk_create_dimensions(sess, asset_analysis_id, all_sub_indicators)

            await repo.update_job_status(
                sess,
                job_id=job_id,
                status="completed",
                progress=1.0,
                steps=steps,
                asset_analysis_id=asset_analysis_id,
            )
            await sess.commit()

        persist_ms = int((time.perf_counter() - t0) * 1000)
        await update_step("persist", "completed", persist_ms)
        log.info(
            "pipeline_complete",
            asset_analysis_id=asset_analysis_id,
            overall_score=overall_score,
            generated_by=generated_by,
            duration_ms=persist_ms,
        )

    except Exception as exc:
        log.exception("pipeline_error", error=str(exc))
        try:
            async with session_factory() as sess:
                await repo.update_job_status(
                    sess,
                    job_id=job_id,
                    status="failed",
                    error_code="PIPELINE_ERROR",
                    error_message=str(exc),
                    steps=steps,
                )
                await sess.commit()
        except Exception:
            pass


# ---------------------------------------------------------------------------
# Holding analysis pipeline (Mode B, synchronous)
# ---------------------------------------------------------------------------


async def run_holding_analysis(
    asset_code: str,
    asset_analysis_id: int,
    holding_info: dict[str, Any],
    risk_preference: str,
    peer_exposure: float,
    language: str,
    llm_config: Any,
    session_factory: Any,
) -> dict[str, Any]:
    """Synchronous holding analysis: load existing asset analysis and generate execution plan.

    Raises ValueError if asset analysis not found.
    Raises RuntimeError if execution agent fails.
    """
    from richson.db import repository as repo  # noqa: PLC0415

    async with session_factory() as sess:
        analysis = await repo.get_asset_analysis_by_id(sess, asset_analysis_id)

    if analysis is None:
        raise ValueError(f"Asset analysis {asset_analysis_id} not found")

    overall_score = float(analysis.overall_score)
    final_scores = {
        "d1": float(analysis.d1_score) if analysis.d1_score is not None else None,
        "d2": float(analysis.d2_score) if analysis.d2_score is not None else None,
        "d3": float(analysis.d3_score) if analysis.d3_score is not None else None,
        "d4": float(analysis.d4_score) if analysis.d4_score is not None else None,
    }

    # Extract support/resistance from analysis_metadata
    metadata = analysis.analysis_metadata or {}
    support_levels = metadata.get("support_levels", [])
    resistance_levels = metadata.get("resistance_levels", [])

    execution_result = await run_layer3_execution(
        asset_code=asset_code,
        overall_score=overall_score,
        dimension_scores=final_scores,
        holding_info=holding_info,
        risk_preference=risk_preference,
        peer_exposure=peer_exposure,
        support_levels=support_levels,
        resistance_levels=resistance_levels,
        language=language,
        llm_config=llm_config,
    )

    if execution_result is None:
        raise RuntimeError("Execution agent failed to generate plan")

    # Clamp validDays (G1.8)
    valid_days = max(1, min(90, execution_result.get("valid_days", 7)))
    execution_result["valid_days"] = valid_days

    return execution_result


# ---------------------------------------------------------------------------
# Demo plan pipeline (Mode C, synchronous)
# ---------------------------------------------------------------------------


async def run_demo_plan(
    asset_code: str,
    language: str,
    llm_config: Any,
    session_factory: Any,
) -> dict[str, Any]:
    """Generate a demo execution plan using fixed assumption parameters.

    Uses the latest rs_asset_analyses record. Raises ValueError if no analysis found.
    """
    from richson.db import repository as repo  # noqa: PLC0415

    async with session_factory() as sess:
        analysis = await repo.get_latest_asset_analysis(sess, asset_code, language)

    if analysis is None:
        raise ValueError(f"No analysis found for {asset_code}")

    # Fixed demo parameters (TRD SS5.3)
    price_at = float(analysis.price_at_analysis) if analysis.price_at_analysis else 100.0
    demo_cost = price_at * _DEMO_COST_PRICE_FACTOR

    holding_info = {
        "holdingId": 0,
        "costPrice": demo_cost,
        "positionRatio": _DEMO_POSITION_RATIO,
        "quantity": 1,
    }

    overall_score = float(analysis.overall_score)
    final_scores = {
        "d1": float(analysis.d1_score) if analysis.d1_score is not None else None,
        "d2": float(analysis.d2_score) if analysis.d2_score is not None else None,
        "d3": float(analysis.d3_score) if analysis.d3_score is not None else None,
        "d4": float(analysis.d4_score) if analysis.d4_score is not None else None,
    }
    metadata = analysis.analysis_metadata or {}
    support_levels = metadata.get("support_levels", [])
    resistance_levels = metadata.get("resistance_levels", [])

    execution_result = await run_layer3_execution(
        asset_code=asset_code,
        overall_score=overall_score,
        dimension_scores=final_scores,
        holding_info=holding_info,
        risk_preference=_DEMO_RISK_PREFERENCE,
        peer_exposure=_DEMO_PEER_EXPOSURE,
        support_levels=support_levels,
        resistance_levels=resistance_levels,
        language=language,
        llm_config=llm_config,
    )

    if execution_result is None:
        raise RuntimeError("Execution agent failed to generate demo plan")

    valid_days = max(1, min(90, execution_result.get("valid_days", 7)))
    execution_result["valid_days"] = valid_days
    execution_result["is_demo_plan"] = True
    return execution_result
