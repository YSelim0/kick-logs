from collections.abc import Callable

from kick_logs.application.dto.channels import ChannelDTO, channel_to_dto
from kick_logs.application.exceptions import ChannelNotFoundError
from kick_logs.application.ports.unit_of_work import UnitOfWork


class RemoveChannelUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(self, channel_id: int) -> ChannelDTO:
        async with self._unit_of_work_factory() as unit_of_work:
            channel = await unit_of_work.channels.get_by_id(channel_id)
            if channel is None:
                raise ChannelNotFoundError("Channel not found.")

            channel.disable()
            channel = await unit_of_work.channels.update(channel)
            await unit_of_work.commit()

        return channel_to_dto(channel)
