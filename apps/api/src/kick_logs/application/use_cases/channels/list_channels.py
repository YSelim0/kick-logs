from collections.abc import Callable

from kick_logs.application.dto.channels import ChannelDTO, channel_to_dto
from kick_logs.application.ports.unit_of_work import UnitOfWork


class ListChannelsUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(self) -> list[ChannelDTO]:
        async with self._unit_of_work_factory() as unit_of_work:
            channels = await unit_of_work.channels.list_all()

        return [channel_to_dto(channel) for channel in channels]
