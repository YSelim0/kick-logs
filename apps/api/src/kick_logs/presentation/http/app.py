from fastapi import FastAPI

from kick_logs.core.config import Settings, get_settings
from kick_logs.core.logging import configure_logging
from kick_logs.presentation.http.routes.health import router as health_router


def create_app(settings: Settings | None = None) -> FastAPI:
    resolved_settings = settings or get_settings()
    configure_logging(resolved_settings.log_level)

    app = FastAPI(title=resolved_settings.app_name)
    app.include_router(health_router)
    return app
