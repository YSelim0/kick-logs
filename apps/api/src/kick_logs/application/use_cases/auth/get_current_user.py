from collections.abc import Callable

from kick_logs.application.exceptions import UserNotFoundError
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.domain.entities.user import User


class GetCurrentUserUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(self, user_id: int) -> User:
        async with self._unit_of_work_factory() as unit_of_work:
            user = await unit_of_work.users.get_by_id(user_id)

        if user is None or not user.is_active:
            raise UserNotFoundError("Current user was not found.")

        return user
