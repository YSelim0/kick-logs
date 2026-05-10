from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, Response, status

from kick_logs.application.exceptions import AuthenticationFailedError
from kick_logs.application.use_cases.auth import LoginUseCase
from kick_logs.core.config import Settings
from kick_logs.domain.entities.user import User
from kick_logs.infrastructure.auth import JwtTokenService, PasslibPasswordHasher
from kick_logs.presentation.http.dependencies import (
    UnitOfWorkFactory,
    get_current_user,
    get_password_hasher,
    get_settings_dependency,
    get_token_service,
    get_unit_of_work_factory,
)
from kick_logs.presentation.http.schemas.auth import AuthResponse, LoginRequest
from kick_logs.presentation.http.schemas.users import AdminUserResponse

router = APIRouter(prefix="/auth", tags=["auth"])

SettingsDep = Annotated[Settings, Depends(get_settings_dependency)]
UnitOfWorkFactoryDep = Annotated[UnitOfWorkFactory, Depends(get_unit_of_work_factory)]
PasswordHasherDep = Annotated[PasslibPasswordHasher, Depends(get_password_hasher)]
TokenServiceDep = Annotated[JwtTokenService, Depends(get_token_service)]
CurrentUserDep = Annotated[User, Depends(get_current_user)]


@router.post("/login", response_model=AuthResponse)
async def login(
    payload: LoginRequest,
    response: Response,
    settings: SettingsDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
    password_hasher: PasswordHasherDep,
    token_service: TokenServiceDep,
) -> AuthResponse:
    try:
        session = await LoginUseCase(
            unit_of_work_factory,
            password_hasher,
            token_service,
        ).execute(payload.email, payload.password)
    except AuthenticationFailedError as exc:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid credentials.",
        ) from exc

    response.set_cookie(
        key=settings.jwt_cookie_name,
        value=session.access_token,
        max_age=settings.jwt_expires_minutes * 60,
        httponly=True,
        secure=settings.jwt_cookie_secure,
        samesite=settings.jwt_cookie_samesite,
        path="/",
    )
    return AuthResponse.from_dto(session)


@router.post("/logout")
async def logout(
    response: Response,
    settings: SettingsDep,
) -> dict[str, str]:
    response.delete_cookie(key=settings.jwt_cookie_name, path="/")
    return {"status": "ok"}


@router.get("/me", response_model=AdminUserResponse)
async def me(current_user: CurrentUserDep) -> AdminUserResponse:
    return AdminUserResponse(
        id=current_user.id,
        email=current_user.email,
        role=current_user.role,
        is_active=current_user.is_active,
    )
