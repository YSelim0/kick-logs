from pydantic import BaseModel, Field

from kick_logs.application.dto.users import AdminUserDTO
from kick_logs.domain.value_objects.roles import UserRole


class AdminUserResponse(BaseModel):
    id: int
    email: str
    role: UserRole
    is_active: bool

    @classmethod
    def from_dto(cls, user: AdminUserDTO) -> "AdminUserResponse":
        return cls(
            id=user.id,
            email=user.email,
            role=user.role,
            is_active=user.is_active,
        )


class CreateAdminUserRequest(BaseModel):
    email: str = Field(min_length=3, max_length=320)
    password: str = Field(min_length=8, max_length=256)
