from collections.abc import Callable

from kick_logs.application.dto.channel_profiles import (
    ChannelProfileChannelDTO,
    ChannelProfileDTO,
)
from kick_logs.application.dto.messages import (
    MessageSearchItemDTO,
    chat_message_to_dto,
    message_channel_to_dto,
    message_sender_to_dto,
)
from kick_logs.application.exceptions import ChannelNotFoundError
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.application.use_cases.messages.search_messages import MessageSearchDataError
from kick_logs.domain.value_objects.analytics_filters import AnalyticsFilters
from kick_logs.domain.value_objects.pagination import CursorPagination


class GetChannelProfileUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(self, slug: str) -> ChannelProfileDTO:
        async with self._unit_of_work_factory() as unit_of_work:
            channel = await unit_of_work.channels.get_by_slug(slug)
            if channel is None or channel.id is None:
                raise ChannelNotFoundError("Channel profile not found.")

            filters = AnalyticsFilters(channel=channel.slug)
            latest_messages = await unit_of_work.messages.list_latest_by_channel_id(
                channel.id,
                CursorPagination(limit=10),
            )
            senders = await unit_of_work.senders.list_by_ids(
                {message.sender_id for message in latest_messages}
            )

            return ChannelProfileDTO(
                channel=ChannelProfileChannelDTO(
                    id=channel.id,
                    kick_channel_id=channel.kick_channel_id,
                    kick_chatroom_id=channel.kick_chatroom_id,
                    slug=channel.slug,
                    display_name=channel.display_name,
                    profile_image_url=channel.profile_image_url,
                    banner_image_url=channel.banner_image_url,
                    is_enabled=channel.is_enabled,
                ),
                overview=await unit_of_work.analytics.get_overview(filters),
                message_volume=await unit_of_work.analytics.get_message_volume(filters, "day"),
                top_senders=await unit_of_work.analytics.get_top_senders(filters, 5),
                top_emotes=await unit_of_work.analytics.get_top_emotes(filters, 5),
                latest_messages=self._build_latest_items(
                    latest_messages=latest_messages,
                    channel=channel,
                    senders=senders,
                ),
            )

    def _build_latest_items(
        self,
        *,
        latest_messages,
        channel,
        senders,
    ) -> list[MessageSearchItemDTO]:
        senders_by_id = {sender.id: sender for sender in senders}
        items: list[MessageSearchItemDTO] = []

        for message in latest_messages:
            sender = senders_by_id.get(message.sender_id)
            if sender is None:
                raise MessageSearchDataError("Stored message references missing data.")

            items.append(
                MessageSearchItemDTO(
                    message=chat_message_to_dto(message),
                    sender=message_sender_to_dto(sender),
                    channel=message_channel_to_dto(channel),
                )
            )

        return items
