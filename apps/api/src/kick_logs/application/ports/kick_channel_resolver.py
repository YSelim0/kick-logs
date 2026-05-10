from typing import Protocol

from kick_logs.application.dto.channels import ResolvedKickChannelDTO


class KickChannelResolver(Protocol):
    async def resolve(self, slug: str) -> ResolvedKickChannelDTO: ...
