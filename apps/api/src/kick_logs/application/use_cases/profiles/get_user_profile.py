from collections.abc import Callable

from kick_logs.application.dto.messages import (
    MessageSearchItemDTO,
    chat_message_to_dto,
    message_channel_to_dto,
    message_sender_to_dto,
)
from kick_logs.application.dto.user_profiles import UserProfileDTO, UserProfileSenderDTO
from kick_logs.application.exceptions import SenderNotFoundError
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.application.use_cases.messages.search_messages import MessageSearchDataError
from kick_logs.domain.value_objects.analytics_filters import AnalyticsFilters
from kick_logs.domain.value_objects.pagination import CursorPagination
from kick_logs.domain.value_objects.search_filters import MessageSearchFilters


class GetUserProfileUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(self, slug: str) -> UserProfileDTO:
        async with self._unit_of_work_factory() as unit_of_work:
            sender = await unit_of_work.senders.get_by_slug(slug)
            if sender is None or sender.id is None:
                raise SenderNotFoundError("Sender profile not found.")

            filters = AnalyticsFilters(sender=sender.slug)
            latest_messages = await unit_of_work.messages.search(
                MessageSearchFilters(sender=sender.slug),
                CursorPagination(limit=10),
            )
            channels = await unit_of_work.channels.list_by_ids(
                {message.channel_id for message in latest_messages}
            )
            senders = await unit_of_work.senders.list_by_ids(
                {message.sender_id for message in latest_messages}
            )

            return UserProfileDTO(
                sender=UserProfileSenderDTO(
                    id=sender.id,
                    kick_user_id=sender.kick_user_id,
                    username=sender.username,
                    slug=sender.slug,
                    profile_image_url=sender.profile_image_url,
                ),
                overview=await unit_of_work.analytics.get_overview(filters),
                message_volume=await unit_of_work.analytics.get_message_volume(filters, "day"),
                top_channels=await unit_of_work.analytics.get_top_channels(filters, 5),
                top_emotes=await unit_of_work.analytics.get_top_emotes(filters, 5),
                latest_messages=self._build_latest_items(
                    latest_messages=latest_messages,
                    channels=channels,
                    senders=senders,
                ),
            )

    def _build_latest_items(
        self,
        *,
        latest_messages,
        channels,
        senders,
    ) -> list[MessageSearchItemDTO]:
        channels_by_id = {channel.id: channel for channel in channels}
        senders_by_id = {sender.id: sender for sender in senders}
        items: list[MessageSearchItemDTO] = []

        for message in latest_messages:
            channel = channels_by_id.get(message.channel_id)
            sender = senders_by_id.get(message.sender_id)
            if channel is None or sender is None:
                raise MessageSearchDataError("Stored message references missing data.")

            items.append(
                MessageSearchItemDTO(
                    message=chat_message_to_dto(message),
                    sender=message_sender_to_dto(sender),
                    channel=message_channel_to_dto(channel),
                )
            )

        return items
