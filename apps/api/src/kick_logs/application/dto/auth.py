from dataclasses import dataclass

from kick_logs.application.dto.users import AdminUserDTO


@dataclass(frozen=True, slots=True)
class AuthSessionDTO:
    access_token: str
    user: AdminUserDTO
