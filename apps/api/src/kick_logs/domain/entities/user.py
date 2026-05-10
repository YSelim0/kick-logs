from dataclasses import dataclass
from datetime import UTC, datetime

from kick_logs.domain.exceptions import DomainError
from kick_logs.domain.value_objects.roles import UserRole


@dataclass(slots=True)
class User:
    email: str
    password_hash: str
    role: UserRole
    id: int | None = None
    is_active: bool = True
    created_at: datetime | None = None
    updated_at: datetime | None = None

    def __post_init__(self) -> None:
        normalized_email = self.email.strip().lower()
        if not normalized_email:
            raise DomainError("User email is required.")
        if not self.password_hash:
            raise DomainError("Password hash is required.")
        self.email = normalized_email

    @classmethod
    def create_admin(cls, email: str, password_hash: str) -> "User":
        now = datetime.now(UTC)
        return cls(
            email=email,
            password_hash=password_hash,
            role=UserRole.ADMIN,
            created_at=now,
            updated_at=now,
        )

    def deactivate(self) -> None:
        self.is_active = False
        self.updated_at = datetime.now(UTC)
