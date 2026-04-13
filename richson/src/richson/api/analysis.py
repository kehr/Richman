"""Synchronous analysis endpoints (Mode B and C).

POST /analyze/holding    - holding-level personalized execution plan
POST /analyze/demo-plan  - demo execution plan for unauthenticated users
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends, HTTPException, Request

from richson.api.auth import require_api_key
from richson.schemas.analysis import AnalyzeHoldingRequest, DemoPlanRequest

router = APIRouter(prefix="/analyze", dependencies=[Depends(require_api_key)])
logger = structlog.get_logger()


@router.post("/holding")
async def analyze_holding(
    body: AnalyzeHoldingRequest,
    request: Request,
) -> dict:
    """Generate personalized execution plan for a user holding (Mode B).

    Synchronous endpoint; richman expects a response within 30 seconds.
    Raises 404 if the referenced asset analysis does not exist.
    Raises 502 if the execution agent fails.
    """
    from richson.core.pipeline import run_holding_analysis  # noqa: PLC0415
    from richson.main import get_session_factory  # noqa: PLC0415

    session_factory = get_session_factory()

    log = logger.bind(
        asset_code=body.asset_code,
        asset_analysis_id=body.asset_analysis_id,
        request_id=str(body.request_id) if body.request_id else None,
    )
    log.info("holding_analysis_start")

    holding_info = {
        "holdingId": body.holding.holding_id,
        "costPrice": float(body.holding.cost_price),
        "positionRatio": body.holding.position_ratio,
        "quantity": body.holding.quantity,
    }

    try:
        result = await run_holding_analysis(
            asset_code=body.asset_code,
            asset_analysis_id=body.asset_analysis_id,
            holding_info=holding_info,
            risk_preference=body.risk_preference,
            peer_exposure=body.peer_exposure,
            language=body.language,
            llm_config=body.llm_config,
            session_factory=session_factory,
        )
    except ValueError as exc:
        raise HTTPException(status_code=404, detail={
            "error": {
                "code": "ASSET_NOT_FOUND",
                "message": str(exc),
                "details": [],
            }
        }) from exc
    except RuntimeError as exc:
        log.error("holding_analysis_failed", error=str(exc))
        raise HTTPException(status_code=502, detail={
            "error": {
                "code": "LLM_INVALID_RESPONSE",
                "message": "Execution agent failed to generate plan",
                "details": [],
            }
        }) from exc

    log.info("holding_analysis_complete")

    # Map snake_case result to camelCase API response
    return {"data": _camel_case_plan(result)}


@router.post("/demo-plan")
async def analyze_demo_plan(
    body: DemoPlanRequest,
    request: Request,
) -> dict:
    """Generate a demo execution plan using fixed assumption parameters (Mode C).

    Uses the latest rs_asset_analyses record for the asset. Raises 404 if no analysis exists.
    """
    from richson.core.pipeline import run_demo_plan  # noqa: PLC0415
    from richson.main import get_session_factory  # noqa: PLC0415

    session_factory = get_session_factory()

    log = logger.bind(
        asset_code=body.asset_code,
        request_id=str(body.request_id) if body.request_id else None,
    )
    log.info("demo_plan_start")

    try:
        result = await run_demo_plan(
            asset_code=body.asset_code,
            language=body.language,
            llm_config=body.llm_config,
            session_factory=session_factory,
        )
    except ValueError as exc:
        raise HTTPException(status_code=404, detail={
            "error": {
                "code": "ASSET_NOT_FOUND",
                "message": str(exc),
                "details": [],
            }
        }) from exc
    except RuntimeError as exc:
        log.error("demo_plan_failed", error=str(exc))
        raise HTTPException(status_code=502, detail={
            "error": {
                "code": "LLM_INVALID_RESPONSE",
                "message": "Execution agent failed to generate demo plan",
                "details": [],
            }
        }) from exc

    log.info("demo_plan_complete")
    return {"data": _camel_case_plan(result)}


def _camel_case_plan(plan: dict) -> dict:
    """Convert snake_case execution plan keys to camelCase for API response."""
    out = {}
    key_map = {
        "action": "action",
        "action_label": "actionLabel",
        "default_action": "defaultAction",
        "current_position": "currentPosition",
        "target_position": "targetPosition",
        "stop_loss": "stopLoss",
        "take_profit": "takeProfit",
        "valid_days": "validDays",
        "no_trigger_note": "noTriggerNote",
        "concentration_level": "concentrationLevel",
        "concentration_message": "concentrationMessage",
        "is_demo_plan": "isDemoPlan",
        "scenarios": "scenarios",
    }
    for snake, camel in key_map.items():
        if snake in plan:
            out[camel] = plan[snake]

    # Normalize scenarios
    scenarios = plan.get("scenarios") or []
    normalized_scenarios = []
    for s in scenarios:
        normalized_scenarios.append({
            "condition": s.get("condition"),
            "action": s.get("action"),
            "lotCount": s.get("lot_count") if "lot_count" in s else s.get("lotCount"),
            "rationale": s.get("rationale"),
            "priority": s.get("priority"),
            "exclusionGroup": s.get("exclusion_group") if "exclusion_group" in s else s.get("exclusionGroup"),
        })
    out["scenarios"] = normalized_scenarios
    return out
