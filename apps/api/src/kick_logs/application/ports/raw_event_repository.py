from typing import Protocol

from kick_logs.domain.entities.raw_kick_event import RawKickEvent


class RawEventRepository(Protocol):
    async def add(self, event: RawKickEvent) -> RawKickEvent: ...

    async def get_by_id(self, event_id: int) -> RawKickEvent | None: ...

    async def get_by_kick_message_id(self, kick_message_id: str) -> RawKickEvent | None: ...

    async def claim_pending(
        self,
        *,
        limit: int,
        processing_timeout_seconds: int,
    ) -> list[RawKickEvent]: ...

    async def mark_processed(self, event_id: int) -> RawKickEvent: ...

    async def mark_failed(
        self,
        *,
        event_id: int,
        error: str,
        max_attempts: int,
    ) -> RawKickEvent: ...

    async def pending_count(self) -> int: ...
