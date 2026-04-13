"""FastAPI application factory for richson service."""

from __future__ import annotations

import time
from contextlib import asynccontextmanager
from typing import AsyncIterator

import structlog
from fastapi import FastAPI, Request, Response
from fastapi.middleware.cors import CORSMiddleware
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine

from richson.config import settings

logger = structlog.get_logger()

# ---------------------------------------------------------------------------
# Database engine (module-level, created once at startup)
# ---------------------------------------------------------------------------

_engine = None
_session_factory: async_sessionmaker[AsyncSession] | None = None


def get_engine():  # type: ignore[return]
    return _engine


def get_session_factory() -> async_sessionmaker[AsyncSession]:
    if _session_factory is None:
        raise RuntimeError("Database session factory not initialized")
    return _session_factory


# ---------------------------------------------------------------------------
# Lifespan: DB pool setup and teardown
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    global _engine, _session_factory

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

    yield

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
        # Disable default OpenAPI auth UI – internal service only
        docs_url="/docs",
        redoc_url=None,
    )

    # CORS – only allow configured origins
    if settings.cors_origins_list:
        app.add_middleware(
            CORSMiddleware,
            allow_origins=settings.cors_origins_list,
            allow_credentials=True,
            allow_methods=["*"],
            allow_headers=["*"],
        )

    # Request logging middleware
    @app.middleware("http")
    async def log_requests(request: Request, call_next: object) -> Response:
        start = time.perf_counter()
        response: Response = await call_next(request)  # type: ignore[operator]
        duration_ms = round((time.perf_counter() - start) * 1000)
        logger.info(
            "http request",
            method=request.method,
            path=request.url.path,
            status_code=response.status_code,
            duration_ms=duration_ms,
            request_id=request.headers.get("x-request-id"),
        )
        return response

    # Business routers will be registered in later steps (Step 6)
    # e.g. app.include_router(health_router) etc.

    return app


app = create_app()
