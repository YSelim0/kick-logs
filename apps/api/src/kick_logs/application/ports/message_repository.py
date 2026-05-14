from typing import Protocol

from kick_logs.domain.entities.chat_message import ChatMessage
from kick_logs.domain.value_objects.pagination import CursorPagination
from kick_logs.domain.value_objects.search_filters import MessageSearchFilters


class MessageRepository(Protocol):
    async def add(self, message: ChatMessage) -> ChatMessage: ...

    async def get_by_kick_message_id(self, kick_message_id: str) -> ChatMessage | None: ...

    async def search(
        self,
        filters: MessageSearchFilters,
        pagination: CursorPagination,
    ) -> list[ChatMessage]: ...

    async def list_latest_by_channel_id(
        self,
        channel_id: int,
        pagination: CursorPagination,
    ) -> list[ChatMessage]: ...
