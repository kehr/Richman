"""Request ID tracing middleware (TRD SS4.3).

Reads X-Request-ID header from incoming requests and propagates it to:
- structlog context for all subsequent log entries in this request
- Response headers (echoed back to the caller)

If no X-Request-ID is present, a new UUID is generated.
"""

from __future__ import annotations

import uuid

import structlog
from fastapi import Request, Response
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.types import ASGIApp


class RequestTracingMiddleware(BaseHTTPMiddleware):
    """Inject X-Request-ID into structlog context and response headers."""

    def __init__(self, app: ASGIApp) -> None:
        super().__init__(app)

    async def dispatch(self, request: Request, call_next: object) -> Response:
        request_id = request.headers.get("x-request-id") or str(uuid.uuid4())

        # Bind request_id to structlog context for this request
        structlog.contextvars.clear_contextvars()
        structlog.contextvars.bind_contextvars(request_id=request_id)

        response: Response = await call_next(request)  # type: ignore[operator]
        response.headers["x-request-id"] = request_id
        return response
