from pydantic import BaseModel, Field

from kick_logs.application.dto.auth import AuthSessionDTO
from kick_logs.presentation.http.schemas.users import AdminUserResponse


class LoginRequest(BaseModel):
    email: str = Field(min_length=3, max_length=320)
    password: str = Field(min_length=1, max_length=256)


class AuthResponse(BaseModel):
    user: AdminUserResponse

    @classmethod
    def from_dto(cls, session: AuthSessionDTO) -> "AuthResponse":
        return cls(user=AdminUserResponse.from_dto(session.user))
