from collections.abc import Callable

from kick_logs.application.dto.messages import (
    MessageSearchItemDTO,
    MessageSearchPageDTO,
    chat_message_to_dto,
    message_channel_to_dto,
    message_sender_to_dto,
)
from kick_logs.application.exceptions import ApplicationError
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.domain.value_objects.pagination import CursorPagination, MessageCursor
from kick_logs.domain.value_objects.search_filters import MessageSearchFilters


class MessageSearchDataError(ApplicationError):
    """Raised when stored search result relationships are incomplete."""


class SearchMessagesUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(
        self,
        filters: MessageSearchFilters,
        pagination: CursorPagination,
    ) -> MessageSearchPageDTO:
        async with self._unit_of_work_factory() as unit_of_work:
            messages = await unit_of_work.messages.search(filters, pagination)
            channels = await unit_of_work.channels.list_by_ids(
                {message.channel_id for message in messages}
            )
            senders = await unit_of_work.senders.list_by_ids(
                {message.sender_id for message in messages}
            )

        channels_by_id = {channel.id: channel for channel in channels}
        senders_by_id = {sender.id: sender for sender in senders}
        items: list[MessageSearchItemDTO] = []

        for message in messages:
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

        return MessageSearchPageDTO(
            items=items,
            next_cursor=self._build_next_cursor(messages, pagination),
        )

    def _build_next_cursor(
        self,
        messages,
        pagination: CursorPagination,
    ) -> MessageCursor | None:
        if len(messages) < pagination.limit:
            return None

        last_message = messages[-1]
        if last_message.id is None:
            return None

        return MessageCursor(
            message_created_at=last_message.message_created_at,
            message_id=last_message.id,
        )
