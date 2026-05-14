import logging
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from kick_logs.core.config import Settings, get_settings
from kick_logs.core.logging import configure_logging
from kick_logs.infrastructure.auth import PasslibPasswordHasher
from kick_logs.infrastructure.database import SqlAlchemyUnitOfWork, create_session_factory
from kick_logs.infrastructure.seed import seed_super_admin
from kick_logs.presentation.http.routes.admin_channels import router as admin_channels_router
from kick_logs.presentation.http.routes.admin_operations import router as admin_operations_router
from kick_logs.presentation.http.routes.admin_users import router as admin_users_router
from kick_logs.presentation.http.routes.analytics import router as analytics_router
from kick_logs.presentation.http.routes.auth import router as auth_router
from kick_logs.presentation.http.routes.channel_profiles import router as channel_profiles_router
from kick_logs.presentation.http.routes.health import router as health_router
from kick_logs.presentation.http.routes.messages import router as messages_router
from kick_logs.presentation.http.routes.user_profiles import router as user_profiles_router

logger = logging.getLogger(__name__)


def build_lifespan(
    settings: Settings,
    should_seed_super_admin: bool,
):
    @asynccontextmanager
    async def lifespan(_app: FastAPI) -> AsyncIterator[None]:
        if should_seed_super_admin:
            session_factory = create_session_factory()
            await seed_super_admin(
                lambda: SqlAlchemyUnitOfWork(session_factory),
                PasslibPasswordHasher(),
                settings,
            )
            logger.info("Default super admin seed checked.")
        yield

    return lifespan


def create_app(
    settings: Settings | None = None,
    seed_super_admin_on_startup: bool | None = None,
) -> FastAPI:
    resolved_settings = settings or get_settings()
    configure_logging(resolved_settings.log_level)
    should_seed = (
        resolved_settings.seed_super_admin_on_startup
        if seed_super_admin_on_startup is None
        else seed_super_admin_on_startup
    )

    app = FastAPI(
        title=resolved_settings.app_name,
        lifespan=build_lifespan(resolved_settings, should_seed),
    )
    cors_origins = parse_cors_origins(resolved_settings.backend_cors_origins)
    if cors_origins:
        app.add_middleware(
            CORSMiddleware,
            allow_origins=cors_origins,
            allow_credentials=True,
            allow_methods=["*"],
            allow_headers=["*"],
        )

    app.include_router(health_router)
    app.include_router(auth_router)
    app.include_router(analytics_router)
    app.include_router(messages_router)
    app.include_router(user_profiles_router)
    app.include_router(channel_profiles_router)
    app.include_router(admin_channels_router)
    app.include_router(admin_operations_router)
    app.include_router(admin_users_router)
    return app


def parse_cors_origins(value: str) -> list[str]:
    return [origin.strip() for origin in value.split(",") if origin.strip()]
