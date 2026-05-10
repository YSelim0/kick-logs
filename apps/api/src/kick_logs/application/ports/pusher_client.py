from collections.abc import AsyncIterator
from typing import Protocol

from kick_logs.application.dto.listener import ListenerChannelDTO


class PusherClient(Protocol):
    def listen(self, channels: list[ListenerChannelDTO]) -> AsyncIterator[str]: ...
