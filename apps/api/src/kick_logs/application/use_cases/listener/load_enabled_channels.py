from collections.abc import Callable

from kick_logs.application.dto.listener import (
    ListenerChannelDTO,
    LoadEnabledChannelsResultDTO,
    SkippedListenerChannelDTO,
    channel_to_listener_dto,
)
from kick_logs.application.exceptions import ChannelResolutionError
from kick_logs.application.ports.kick_channel_resolver import KickChannelResolver
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.domain.entities import Channel


class LoadEnabledChannelsUseCase:
    def __init__(
        self,
        unit_of_work_factory: Callable[[], UnitOfWork],
        channel_resolver: KickChannelResolver,
    ) -> None:
        self._unit_of_work_factory = unit_of_work_factory
        self._channel_resolver = channel_resolver

    async def execute(self) -> LoadEnabledChannelsResultDTO:
        loaded: list[ListenerChannelDTO] = []
        skipped: list[SkippedListenerChannelDTO] = []

        async with self._unit_of_work_factory() as unit_of_work:
            channels = await unit_of_work.channels.list_enabled()

            for channel in channels:
                channel = await self._ensure_resolved(unit_of_work, channel)
                if channel is None:
                    skipped.append(
                        SkippedListenerChannelDTO(
                            id=None,
                            slug="unknown",
                            reason="Channel could not be resolved.",
                        )
                    )
                    continue

                if channel.id is None:
                    skipped.append(
                        SkippedListenerChannelDTO(
                            id=None,
                            slug=channel.slug,
                            reason="Channel has no database id.",
                        )
                    )
                    continue

                if channel.kick_channel_id is None or channel.kick_chatroom_id is None:
                    skipped.append(
                        SkippedListenerChannelDTO(
                            id=channel.id,
                            slug=channel.slug,
                            reason="Channel has missing Kick metadata.",
                        )
                    )
                    continue

                loaded.append(channel_to_listener_dto(channel))

            await unit_of_work.commit()

        return LoadEnabledChannelsResultDTO(channels=loaded, skipped=skipped)

    async def _ensure_resolved(
        self,
        unit_of_work: UnitOfWork,
        channel: Channel,
    ) -> Channel | None:
        if channel.kick_channel_id is not None and channel.kick_chatroom_id is not None:
            return channel

        try:
            resolved = await self._channel_resolver.resolve(channel.slug)
        except ChannelResolutionError:
            return channel

        channel.kick_channel_id = resolved.kick_channel_id
        channel.kick_chatroom_id = resolved.kick_chatroom_id
        channel.display_name = resolved.display_name
        channel.profile_image_url = resolved.profile_image_url
        channel.banner_image_url = resolved.banner_image_url
        channel.raw_payload = resolved.raw_payload
        return await unit_of_work.channels.update(channel)
