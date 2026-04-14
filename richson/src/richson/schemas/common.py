"""Common Pydantic types shared across richson API schemas."""

from __future__ import annotations

from typing import Any

from pydantic import BaseModel, Field


class ErrorDetail(BaseModel):
    code: str
    message: str
    details: list[Any] = Field(default_factory=list)


class ErrorResponse(BaseModel):
    error: ErrorDetail


class DataResponse[T](BaseModel):
    """Generic success envelope: {"data": <T>}."""

    data: T


class PaginationMeta(BaseModel):
    page: int = Field(ge=1)
    page_size: int = Field(ge=1, le=200)
    total: int = Field(ge=0)
    total_pages: int = Field(ge=0)


class LLMConfig(BaseModel):
    """LLM provider configuration passed per-request from richman.

    The api_key field is marked ``repr=False`` so Pydantic's automatic
    ``__repr__`` / ``str()`` output elides it (richson SS21.9): logging a
    LLMConfig instance — or any model that embeds one — will not leak the
    provider secret. The HTTP request-logging middleware (see main.py) only
    records method / path / status / duration and never reads the body, so
    the raw "apiKey" JSON field also stays out of structured logs.
    """

    provider: str = Field(
        description="LLM provider: claude | openai | openai_compatible | gemini"
    )
    model: str
    api_key: str = Field(default="", alias="apiKey", repr=False)
    api_base: str | None = Field(
        default=None, alias="apiBase", description="Required for openai_compatible"
    )

    model_config = {"populate_by_name": True}
