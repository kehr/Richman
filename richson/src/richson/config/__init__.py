"""Application configuration loaded from environment variables via pydantic-settings."""

from __future__ import annotations

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    # Server
    app_env: str = "dev"
    host: str = "0.0.0.0"
    port: int = 8001
    log_level: str = "info"
    workers: int = 1

    # Database
    database_url: str = Field(
        default="postgresql+asyncpg://richson_user:password@localhost:5432/richman",
    )

    # Internal auth
    internal_api_key: str = Field(default="change-me-in-production")

    # LLM
    platform_llm_api_key: str = Field(default="")
    default_llm_provider: str = "claude"
    default_llm_model: str = "claude-sonnet-4-20250514"
    daily_llm_budget_usd: float = 10.0

    # External data sources
    fred_api_key: str = Field(default="")

    # CORS
    cors_allowed_origins: str = "http://localhost:3000"

    @property
    def cors_origins_list(self) -> list[str]:
        """Parse comma-separated origins into a list."""
        return [o.strip() for o in self.cors_allowed_origins.split(",") if o.strip()]


# Module-level singleton; imported by other modules
settings = Settings()
