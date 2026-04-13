"""Common Pydantic types shared across richson API schemas."""

from __future__ import annotations

from typing import Any, Generic, TypeVar

from pydantic import BaseModel, Field

T = TypeVar("T")


class ErrorDetail(BaseModel):
    code: str
    message: str
    details: list[Any] = Field(default_factory=list)


class ErrorResponse(BaseModel):
    error: ErrorDetail


class DataResponse(BaseModel, Generic[T]):
    """Generic success envelope: {"data": <T>}."""

    data: T


class PaginationMeta(BaseModel):
    page: int = Field(ge=1)
    page_size: int = Field(ge=1, le=200)
    total: int = Field(ge=0)
    total_pages: int = Field(ge=0)


class LLMConfig(BaseModel):
    """LLM provider configuration passed per-request from richman."""

    provider: str = Field(
        description="LLM provider: claude | openai | openai_compatible | gemini"
    )
    model: str
    api_key: str = Field(default="")
    api_base: str | None = Field(default=None, description="Required for openai_compatible")
