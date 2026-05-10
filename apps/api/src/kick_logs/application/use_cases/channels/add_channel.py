from collections.abc import Callable

from kick_logs.application.dto.channels import ChannelDTO, channel_to_dto
from kick_logs.application.ports.kick_channel_resolver import KickChannelResolver
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.domain.entities.channel import Channel


class AddChannelUseCase:
    def __init__(
        self,
        unit_of_work_factory: Callable[[], UnitOfWork],
        channel_resolver: KickChannelResolver,
    ) -> None:
        self._unit_of_work_factory = unit_of_work_factory
        self._channel_resolver = channel_resolver

    async def execute(self, slug: str) -> ChannelDTO:
        resolved_channel = await self._channel_resolver.resolve(slug)

        async with self._unit_of_work_factory() as unit_of_work:
            existing_channel = await unit_of_work.channels.get_by_slug(resolved_channel.slug)

            if existing_channel is None:
                channel = await unit_of_work.channels.add(
                    Channel(
                        kick_channel_id=resolved_channel.kick_channel_id,
                        kick_chatroom_id=resolved_channel.kick_chatroom_id,
                        slug=resolved_channel.slug,
                        display_name=resolved_channel.display_name,
                        profile_image_url=resolved_channel.profile_image_url,
                        banner_image_url=resolved_channel.banner_image_url,
                        raw_payload=resolved_channel.raw_payload,
                    )
                )
            else:
                existing_channel.kick_channel_id = resolved_channel.kick_channel_id
                existing_channel.kick_chatroom_id = resolved_channel.kick_chatroom_id
                existing_channel.display_name = resolved_channel.display_name
                existing_channel.profile_image_url = resolved_channel.profile_image_url
                existing_channel.banner_image_url = resolved_channel.banner_image_url
                existing_channel.raw_payload = resolved_channel.raw_payload
                existing_channel.enable()
                channel = await unit_of_work.channels.update(existing_channel)

            await unit_of_work.commit()

        return channel_to_dto(channel)
