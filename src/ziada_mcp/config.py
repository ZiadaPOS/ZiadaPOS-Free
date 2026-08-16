"""Environment configuration for the ZiadaPOS MCP server."""
from functools import lru_cache

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    # ── ZiadaPOS API ──────────────────────────────────────────────────────
    ziada_api_url: str = Field(
        default="http://localhost:8080/api/v1",
        description="Base URL of the ZiadaPOS Go API (including /api/v1).",
    )

    # Single service account — the shop owner the agent acts for.
    ziada_identifier: str = Field(
        default="",
        description="Owner login identifier: phone or email.",
    )
    ziada_password: str = Field(
        default="",
        description="Owner login password.",
    )
    ziada_store_id: str = Field(
        default="",
        description="Optional active store UUID. Empty = use the owner's default store.",
    )

    # ── Ngamia AI gateway (OpenAI-compatible) ─────────────────────────────
    ngamia_base_url: str = Field(
        default="https://api.ngamia.cc/v1",
        description="Ngamia (OpenAI-compatible) base URL.",
    )
    ngamia_api_key: str = Field(
        default="",
        description="Ngamia API key for the embedded agent.",
    )
    ngamia_model: str = Field(
        default="openai/gpt-4o-mini",
        description="Model used by the embedded agent.",
    )
    agent_max_steps: int = Field(
        default=8,
        description="Maximum tool-call rounds the agent may make per question.",
    )
    agent_timeout_seconds: int = Field(
        default=120,
        description="Timeout for a single LLM call.",
    )

    # ── MCP server ────────────────────────────────────────────────────────
    mcp_host: str = Field(default="0.0.0.0")
    mcp_port: int = Field(default=8081)
    mcp_transport: str = Field(
        default="http",
        description='Transport: "http" (streamable-http) or "stdio".',
    )
    log_level: str = Field(default="INFO")


@lru_cache
def get_settings() -> Settings:
    return Settings()
