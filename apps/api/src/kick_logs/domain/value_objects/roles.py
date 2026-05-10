from enum import StrEnum


class UserRole(StrEnum):
    ADMIN = "admin"
    SUPER_ADMIN = "super_admin"

    @property
    def can_manage_admins(self) -> bool:
        return self is UserRole.SUPER_ADMIN
