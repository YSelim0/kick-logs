from functools import lru_cache

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=("../../.env", ".env"),
        env_file_encoding="utf-8",
        extra="ignore",
    )

    app_name: str = "Kick Logs"
    app_env: str = "local"
    log_level: str = "INFO"
    api_host: str = "0.0.0.0"
    api_port: int = 8000
    backend_cors_origins: str = "http://localhost:3000"

    database_url: str = "postgresql+asyncpg://kick_logs:kick_logs@localhost:5432/kick_logs"
    database_echo: bool = False

    jwt_secret_key: str = Field(default="change-me-for-local-development", repr=False)
    jwt_cookie_name: str = "kick_logs_session"

    default_super_admin_email: str = "admin@kicklogs.local"
    default_super_admin_password: str = Field(default="admin123", repr=False)


@lru_cache
def get_settings() -> Settings:
    return Settings()
