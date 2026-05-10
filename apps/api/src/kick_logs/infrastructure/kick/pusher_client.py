import json
from collections.abc import AsyncIterator, Callable
from contextlib import AbstractAsyncContextManager
from typing import Protocol

from websockets.asyncio.client import connect as websockets_connect

from kick_logs.application.dto.listener import ListenerChannelDTO


class WebSocketConnection(Protocol):
    async def send(self, message: str) -> None: ...

    def __aiter__(self) -> AsyncIterator[str | bytes]: ...


ConnectWebSocket = Callable[[str], AbstractAsyncContextManager[WebSocketConnection]]


class KickPusherClient:
    def __init__(
        self,
        url: str,
        connect: ConnectWebSocket | None = None,
    ) -> None:
        self._url = url
        self._connect = connect or self._connect_with_defaults

    async def listen(self, channels: list[ListenerChannelDTO]) -> AsyncIterator[str]:
        async with self._connect(self._url) as websocket:
            await self._subscribe(websocket, channels)

            async for message in websocket:
                if isinstance(message, bytes):
                    yield message.decode("utf-8", errors="replace")
                else:
                    yield message

    async def _subscribe(
        self,
        websocket: WebSocketConnection,
        channels: list[ListenerChannelDTO],
    ) -> None:
        for channel_name in self._subscription_names(channels):
            await websocket.send(
                json.dumps(
                    {
                        "event": "pusher:subscribe",
                        "data": {"auth": "", "channel": channel_name},
                    }
                )
            )

    def _connect_with_defaults(self, url: str) -> AbstractAsyncContextManager[WebSocketConnection]:
        return websockets_connect(url, ping_interval=30, ping_timeout=10)

    def _subscription_names(self, channels: list[ListenerChannelDTO]) -> list[str]:
        names: list[str] = []
        seen: set[str] = set()

        for channel in channels:
            for name in (
                f"chatrooms.{channel.kick_chatroom_id}.v2",
                f"channel.{channel.kick_channel_id}",
            ):
                if name not in seen:
                    names.append(name)
                    seen.add(name)

        return names
