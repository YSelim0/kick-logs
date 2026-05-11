from typing import Protocol, Self

from kick_logs.application.ports.channel_repository import ChannelRepository
from kick_logs.application.ports.message_repository import MessageRepository
from kick_logs.application.ports.raw_event_repository import RawEventRepository
from kick_logs.application.ports.sender_repository import SenderRepository
from kick_logs.application.ports.user_repository import UserRepository


class UnitOfWork(Protocol):
    users: UserRepository
    channels: ChannelRepository
    senders: SenderRepository
    messages: MessageRepository
    raw_events: RawEventRepository

    async def __aenter__(self) -> Self: ...

    async def __aexit__(self, exc_type: object, exc: object, traceback: object) -> None: ...

    async def commit(self) -> None: ...

    async def rollback(self) -> None: ...
