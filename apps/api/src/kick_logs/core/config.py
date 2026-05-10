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

    jwt_secret_key: str = Field(default="change-me-for-local-development-secret", repr=False)
    jwt_algorithm: str = "HS256"
    jwt_expires_minutes: int = 60 * 24 * 7
    jwt_cookie_name: str = "kick_logs_session"
    jwt_cookie_secure: bool = False
    jwt_cookie_samesite: str = "lax"
    seed_super_admin_on_startup: bool = True

    default_super_admin_email: str = "admin@kicklogs.local"
    default_super_admin_password: str = Field(default="admin123", repr=False)

    kick_pusher_url: str = (
        "wss://ws-us2.pusher.com/app/32cbd69e4b950bf97679"
        "?protocol=7&client=js&version=8.4.0-rc2&flash=false"
    )
    listener_reconnect_initial_delay_seconds: float = 1.0
    listener_reconnect_max_delay_seconds: float = 30.0
    listener_reconnect_multiplier: float = 2.0


@lru_cache
def get_settings() -> Settings:
    return Settings()
