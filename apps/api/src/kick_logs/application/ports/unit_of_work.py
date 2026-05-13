from typing import Protocol, Self

from kick_logs.application.ports.analytics_repository import AnalyticsRepository
from kick_logs.application.ports.channel_repository import ChannelRepository
from kick_logs.application.ports.message_repository import MessageRepository
from kick_logs.application.ports.operations_repository import OperationsRepository
from kick_logs.application.ports.raw_event_repository import RawEventRepository
from kick_logs.application.ports.sender_repository import SenderRepository
from kick_logs.application.ports.user_repository import UserRepository
from kick_logs.application.ports.worker_heartbeat_repository import WorkerHeartbeatRepository


class UnitOfWork(Protocol):
    users: UserRepository
    analytics: AnalyticsRepository
    channels: ChannelRepository
    senders: SenderRepository
    messages: MessageRepository
    raw_events: RawEventRepository
    worker_heartbeats: WorkerHeartbeatRepository
    operations: OperationsRepository

    async def __aenter__(self) -> Self: ...

    async def __aexit__(self, exc_type: object, exc: object, traceback: object) -> None: ...

    async def commit(self) -> None: ...

    async def rollback(self) -> None: ...
