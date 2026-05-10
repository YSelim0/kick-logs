from typing import Protocol

from kick_logs.application.dto.senders import ResolvedSenderProfileDTO


class SenderProfileResolver(Protocol):
    async def resolve(self, slug: str) -> ResolvedSenderProfileDTO: ...
