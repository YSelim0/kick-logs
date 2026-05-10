from typing import Annotated

from fastapi import APIRouter, Depends, HTTPException, status

from kick_logs.application.exceptions import DuplicateUserEmailError, PermissionDeniedError
from kick_logs.application.use_cases.users import CreateAdminUserUseCase, ListAdminUsersUseCase
from kick_logs.domain.entities.user import User
from kick_logs.infrastructure.auth import PasslibPasswordHasher
from kick_logs.presentation.http.dependencies import (
    UnitOfWorkFactory,
    get_password_hasher,
    get_unit_of_work_factory,
    require_admin,
    require_super_admin,
)
from kick_logs.presentation.http.schemas.users import AdminUserResponse, CreateAdminUserRequest

router = APIRouter(prefix="/admin/users", tags=["admin-users"])

AdminUserDep = Annotated[User, Depends(require_admin)]
SuperAdminUserDep = Annotated[User, Depends(require_super_admin)]
UnitOfWorkFactoryDep = Annotated[UnitOfWorkFactory, Depends(get_unit_of_work_factory)]
PasswordHasherDep = Annotated[PasslibPasswordHasher, Depends(get_password_hasher)]


@router.get("", response_model=list[AdminUserResponse])
async def list_admin_users(
    _current_user: AdminUserDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
) -> list[AdminUserResponse]:
    users = await ListAdminUsersUseCase(unit_of_work_factory).execute()
    return [AdminUserResponse.from_dto(user) for user in users]


@router.post("", response_model=AdminUserResponse, status_code=status.HTTP_201_CREATED)
async def create_admin_user(
    payload: CreateAdminUserRequest,
    current_user: SuperAdminUserDep,
    unit_of_work_factory: UnitOfWorkFactoryDep,
    password_hasher: PasswordHasherDep,
) -> AdminUserResponse:
    try:
        created_user = await CreateAdminUserUseCase(
            unit_of_work_factory,
            password_hasher,
        ).execute(current_user, payload.email, payload.password)
    except DuplicateUserEmailError as exc:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="User email already exists.",
        ) from exc
    except PermissionDeniedError as exc:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Super admin role required.",
        ) from exc

    return AdminUserResponse.from_dto(created_user)
