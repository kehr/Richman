"""FastAPI application factory for richson service."""

from __future__ import annotations

import time
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

import structlog
from fastapi import FastAPI, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from richson.config import settings
from richson.logging_config import configure_logging

# Configure structlog before any logger is used
configure_logging(settings.log_level)

logger = structlog.get_logger()

# ---------------------------------------------------------------------------
# Database engine (module-level, created once at startup)
# ---------------------------------------------------------------------------

_engine = None
_session_factory: async_sessionmaker[AsyncSession] | None = None
_scheduler_tasks: list = []


def get_engine():  # type: ignore[return]
    return _engine


def get_session_factory() -> async_sessionmaker[AsyncSession]:
    if _session_factory is None:
        raise RuntimeError("Database session factory not initialized")
    return _session_factory


# ---------------------------------------------------------------------------
# Lifespan: DB pool setup, scheduler start, and teardown
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    global _engine, _session_factory, _scheduler_tasks

    logger.info("startup: initializing database pool", url=settings.database_url.split("@")[-1])
    _engine = create_async_engine(
        settings.database_url,
        pool_size=5,
        max_overflow=10,
        pool_pre_ping=True,
        echo=settings.log_level.lower() == "debug",
    )
    _session_factory = async_sessionmaker(
        bind=_engine,
        class_=AsyncSession,
        expire_on_commit=False,
    )
    logger.info("startup: database pool ready")

    # Start background scheduler
    try:
        from richson.tasks.scheduler import start_scheduler  # noqa: PLC0415
        _scheduler_tasks = await start_scheduler(_session_factory)
        logger.info("startup: scheduler started")
    except Exception as exc:
        logger.warning("startup: scheduler failed to start", error=str(exc))

    yield

    # Shutdown: cancel scheduler tasks
    for task in _scheduler_tasks:
        task.cancel()
    if _scheduler_tasks:
        import asyncio  # noqa: PLC0415
        await asyncio.gather(*_scheduler_tasks, return_exceptions=True)
        logger.info("shutdown: scheduler tasks cancelled")

    logger.info("shutdown: disposing database pool")
    await _engine.dispose()
    logger.info("shutdown: complete")


# ---------------------------------------------------------------------------
# Application factory
# ---------------------------------------------------------------------------


def create_app() -> FastAPI:
    app = FastAPI(
        title="richson",
        version="0.1.0",
        description="Quantitative computation and LLM orchestration service",
        lifespan=lifespan,
        # Internal service only; disable interactive docs in production
        docs_url="/docs",
        redoc_url=None,
    )

    # CORS - only allow configured origins
    if settings.cors_origins_list:
        app.add_middleware(
            CORSMiddleware,
            allow_origins=settings.cors_origins_list,
            allow_credentials=True,
            allow_methods=["*"],
            allow_headers=["*"],
        )

    # Request ID tracing middleware (TRD SS4.3)
    from richson.middleware.tracing import RequestTracingMiddleware  # noqa: PLC0415
    app.add_middleware(RequestTracingMiddleware)

    # Request logging middleware (keep existing pattern, enhanced with request_id)
    @app.middleware("http")
    async def log_requests(request: Request, call_next: object) -> Response:
        start = time.perf_counter()
        response: Response = await call_next(request)  # type: ignore[operator]
        duration_ms = round((time.perf_counter() - start) * 1000)
        logger.info(
            "http_request",
            method=request.method,
            path=request.url.path,
            status_code=response.status_code,
            duration_ms=duration_ms,
        )
        return response

    # ---------------------------------------------------------------------------
    # Register routers
    # ---------------------------------------------------------------------------

    # Health check - no auth required (TRD SS5.4)
    from richson.api.health import router as health_router  # noqa: PLC0415
    app.include_router(health_router)

    # Job management (Mode A: async asset analysis)
    from richson.api.jobs import router as jobs_router  # noqa: PLC0415
    app.include_router(jobs_router)

    # Synchronous analysis (Mode B: holding, Mode C: demo-plan)
    from richson.api.analysis import router as analysis_router  # noqa: PLC0415
    app.include_router(analysis_router)

    # Market data (Mode C: regime, OHLCV)
    from richson.api.market import router as market_router  # noqa: PLC0415
    app.include_router(market_router)

    # Asset score history
    from richson.api.assets import router as assets_router  # noqa: PLC0415
    app.include_router(assets_router)

    # Event radar
    from richson.api.events import router as events_router  # noqa: PLC0415
    app.include_router(events_router)

    # Content generation (weekly insight)
    from richson.api.content import router as content_router  # noqa: PLC0415
    app.include_router(content_router)

    return app


app = create_app()
