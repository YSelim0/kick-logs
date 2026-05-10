from collections.abc import Callable
from functools import lru_cache
from typing import Annotated

from fastapi import Depends, HTTPException, Request, status

from kick_logs.application.exceptions import UserNotFoundError
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.application.use_cases.auth import GetCurrentUserUseCase
from kick_logs.core.config import Settings, get_settings
from kick_logs.domain.entities.user import User
from kick_logs.infrastructure.auth import JwtTokenService, PasslibPasswordHasher
from kick_logs.infrastructure.database import SqlAlchemyUnitOfWork, create_session_factory
from kick_logs.infrastructure.database.unit_of_work import SessionFactory
from kick_logs.infrastructure.kick import KickWebChannelResolver

UnitOfWorkFactory = Callable[[], UnitOfWork]


def get_settings_dependency() -> Settings:
    return get_settings()


@lru_cache
def get_default_session_factory() -> SessionFactory:
    return create_session_factory()


def get_unit_of_work_factory() -> UnitOfWorkFactory:
    session_factory = get_default_session_factory()
    return lambda: SqlAlchemyUnitOfWork(session_factory)


@lru_cache
def get_password_hasher() -> PasslibPasswordHasher:
    return PasslibPasswordHasher()


@lru_cache
def get_token_service() -> JwtTokenService:
    return JwtTokenService(get_settings())


@lru_cache
def get_channel_resolver() -> KickWebChannelResolver:
    return KickWebChannelResolver()


SettingsDep = Annotated[Settings, Depends(get_settings_dependency)]
TokenServiceDep = Annotated[JwtTokenService, Depends(get_token_service)]
UnitOfWorkFactoryDep = Annotated[UnitOfWorkFactory, Depends(get_unit_of_work_factory)]


async def get_current_user(
    request: Request,
    settings: SettingsDep,
    token_service: TokenServiceDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
) -> User:
    token = request.cookies.get(settings.jwt_cookie_name)
    if not token:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authentication required.",
        )

    user_id = token_service.get_user_id(token)
    if user_id is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid session.",
        )

    try:
        return await GetCurrentUserUseCase(unit_of_work_factory).execute(user_id)
    except UserNotFoundError as exc:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid session.",
        ) from exc


CurrentUserDep = Annotated[User, Depends(get_current_user)]


def require_admin(current_user: CurrentUserDep) -> User:
    return current_user


def require_super_admin(current_user: CurrentUserDep) -> User:
    if not current_user.role.can_manage_admins:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Super admin role required.",
        )
    return current_user
