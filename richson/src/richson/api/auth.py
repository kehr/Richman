"""API key authentication middleware (TRD SS4.2).

All endpoints except GET /health require Authorization: Bearer {INTERNAL_API_KEY}.
G1.9: API key must not be logged in plaintext.
"""

from __future__ import annotations

import structlog
from fastapi import Header, HTTPException, status
from fastapi.security import HTTPBearer

from richson.config import settings

logger = structlog.get_logger()

_bearer_scheme = HTTPBearer(auto_error=False)


async def require_api_key(
    authorization: str | None = Header(default=None),
) -> None:
    """FastAPI dependency that validates the INTERNAL_API_KEY bearer token.

    Raises 401 if the token is missing or invalid.
    G1.9: only logs whether auth succeeded/failed, never logs the key value.
    """
    configured_key = settings.internal_api_key

    if not authorization:
        logger.warning("auth_missing_header")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail={
                "error": {
                    "code": "UNAUTHORIZED",
                    "message": "Missing Authorization header",
                    "details": [],
                }
            },
            headers={"WWW-Authenticate": "Bearer"},
        )

    # Expected format: "Bearer <key>"
    parts = authorization.split(" ", 1)
    if len(parts) != 2 or parts[0].lower() != "bearer":
        logger.warning("auth_malformed_header")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail={
                "error": {
                    "code": "UNAUTHORIZED",
                    "message": "Malformed Authorization header; expected 'Bearer <key>'",
                    "details": [],
                }
            },
            headers={"WWW-Authenticate": "Bearer"},
        )

    token = parts[1]
    if token != configured_key:
        logger.warning("auth_invalid_key")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail={
                "error": {
                    "code": "UNAUTHORIZED",
                    "message": "Invalid API key",
                    "details": [],
                }
            },
            headers={"WWW-Authenticate": "Bearer"},
        )
