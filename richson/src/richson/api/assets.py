"""Asset data endpoints.

GET /assets/{assetCode}/score-history - historical score series for trend-line rendering
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends, Query

from richson.api.auth import require_api_key
from richson.db import repository as repo

router = APIRouter(prefix="/assets", dependencies=[Depends(require_api_key)])
logger = structlog.get_logger()

_VALID_DAYS = (30, 90, 180, 240)


@router.get("/{asset_code}/score-history")
async def get_score_history(
    asset_code: str,
    days: int = Query(default=90),
) -> dict:
    """Return historical composite + dimension scores for an asset.

    Query params:
        days: 30 | 90 | 180 | 240 (default 90). Invalid values clamp to nearest valid.

    Response conforms to TRD SS5.3 ScoreHistoryData schema.
    """
    from richson.main import get_session_factory  # noqa: PLC0415

    # Clamp days to valid set (G1.8 bounds validation)
    if days not in _VALID_DAYS:
        # Pick closest valid value
        days = min(_VALID_DAYS, key=lambda d: abs(d - days))

    session_factory = get_session_factory()
    async with session_factory() as sess:
        records = await repo.get_score_history(sess, asset_code, days=days)

    if not records:
        return {
            "data": {
                "assetCode": asset_code,
                "scores": [],
                "versionChanges": [],
            }
        }

    scores = []
    version_changes = []
    prev_version = None

    for record in records:
        date_str = record.analyzed_at.strftime("%Y-%m-%d")
        scores.append({
            "date": date_str,
            "overallScore": float(record.overall_score),
            "d1Score": float(record.d1_score) if record.d1_score is not None else None,
            "d2Score": float(record.d2_score) if record.d2_score is not None else None,
            "d3Score": float(record.d3_score) if record.d3_score is not None else None,
            "d4Score": float(record.d4_score) if record.d4_score is not None else None,
            "modelVersion": record.model_version,
        })

        if prev_version and record.model_version != prev_version:
            version_changes.append({
                "date": date_str,
                "fromVersion": prev_version,
                "toVersion": record.model_version,
                "note": f"Model updated from {prev_version} to {record.model_version}",
            })
        prev_version = record.model_version

    return {
        "data": {
            "assetCode": asset_code,
            "scores": scores,
            "versionChanges": version_changes,
        }
    }
