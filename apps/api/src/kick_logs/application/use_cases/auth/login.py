from collections.abc import Callable

from kick_logs.application.dto.auth import AuthSessionDTO
from kick_logs.application.dto.users import admin_user_to_dto
from kick_logs.application.exceptions import AuthenticationFailedError
from kick_logs.application.ports.password_hasher import PasswordHasher
from kick_logs.application.ports.token_service import TokenService
from kick_logs.application.ports.unit_of_work import UnitOfWork


class LoginUseCase:
    def __init__(
        self,
        unit_of_work_factory: Callable[[], UnitOfWork],
        password_hasher: PasswordHasher,
        token_service: TokenService,
    ) -> None:
        self._unit_of_work_factory = unit_of_work_factory
        self._password_hasher = password_hasher
        self._token_service = token_service

    async def execute(self, email: str, password: str) -> AuthSessionDTO:
        async with self._unit_of_work_factory() as unit_of_work:
            user = await unit_of_work.users.get_by_email(email)

        if user is None or not user.is_active:
            raise AuthenticationFailedError("Invalid email or password.")

        if not self._password_hasher.verify(password, user.password_hash):
            raise AuthenticationFailedError("Invalid email or password.")

        if user.id is None:
            raise AuthenticationFailedError("Invalid user state.")

        return AuthSessionDTO(
            access_token=self._token_service.create_access_token(user.id),
            user=admin_user_to_dto(user),
        )
