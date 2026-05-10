from collections.abc import Callable

from kick_logs.application.dto.users import AdminUserDTO, admin_user_to_dto
from kick_logs.application.ports.unit_of_work import UnitOfWork


class ListAdminUsersUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(self) -> list[AdminUserDTO]:
        async with self._unit_of_work_factory() as unit_of_work:
            users = await unit_of_work.users.list_active()

        return [admin_user_to_dto(user) for user in users]
