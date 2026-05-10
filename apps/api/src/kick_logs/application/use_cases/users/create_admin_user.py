from collections.abc import Callable

from kick_logs.application.dto.users import AdminUserDTO, admin_user_to_dto
from kick_logs.application.exceptions import DuplicateUserEmailError, PermissionDeniedError
from kick_logs.application.ports.password_hasher import PasswordHasher
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.domain.entities.user import User


class CreateAdminUserUseCase:
    def __init__(
        self,
        unit_of_work_factory: Callable[[], UnitOfWork],
        password_hasher: PasswordHasher,
    ) -> None:
        self._unit_of_work_factory = unit_of_work_factory
        self._password_hasher = password_hasher

    async def execute(self, current_user: User, email: str, password: str) -> AdminUserDTO:
        if not current_user.role.can_manage_admins:
            raise PermissionDeniedError("Only super admins can create admin users.")

        async with self._unit_of_work_factory() as unit_of_work:
            existing_user = await unit_of_work.users.get_by_email(email)
            if existing_user is not None:
                raise DuplicateUserEmailError("User email already exists.")

            created_user = await unit_of_work.users.add(
                User.create_admin(
                    email=email,
                    password_hash=self._password_hasher.hash(password),
                )
            )
            await unit_of_work.commit()

        return admin_user_to_dto(created_user)
