from dataclasses import dataclass

from kick_logs.domain.entities.user import User
from kick_logs.domain.value_objects.roles import UserRole


@dataclass(frozen=True, slots=True)
class AdminUserDTO:
    id: int
    email: str
    role: UserRole
    is_active: bool


def admin_user_to_dto(user: User) -> AdminUserDTO:
    if user.id is None:
        raise ValueError("User id is required for API responses.")
    return AdminUserDTO(
        id=user.id,
        email=user.email,
        role=user.role,
        is_active=user.is_active,
    )
