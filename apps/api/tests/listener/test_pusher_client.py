import json
from collections.abc import AsyncIterator

from kick_logs.application.dto.listener import ListenerChannelDTO
from kick_logs.infrastructure.kick import KickPusherClient


class FakeWebSocket:
    def __init__(self, messages: list[str | bytes]) -> None:
        self.messages = messages
        self.sent_messages: list[str] = []

    async def send(self, message: str) -> None:
        self.sent_messages.append(message)

    def __aiter__(self) -> AsyncIterator[str | bytes]:
        return self._iterate()

    async def _iterate(self) -> AsyncIterator[str | bytes]:
        for message in self.messages:
            yield message


class FakeConnector:
    def __init__(self, websocket: FakeWebSocket) -> None:
        self.websocket = websocket
        self.url: str | None = None

    def __call__(self, url: str):
        self.url = url
        return self

    async def __aenter__(self) -> FakeWebSocket:
        return self.websocket

    async def __aexit__(self, exc_type, exc, traceback) -> None:
        return None


async def test_pusher_client_subscribes_to_chatroom_and_channel_events() -> None:
    websocket = FakeWebSocket(messages=[])
    connector = FakeConnector(websocket)
    client = KickPusherClient("wss://example.test", connect=connector)
    channels = [
        ListenerChannelDTO(
            id=1,
            kick_channel_id=100,
            kick_chatroom_id=200,
            slug="hype",
            display_name="Hype",
        )
    ]

    received = [message async for message in client.listen(channels)]

    assert received == []
    assert connector.url == "wss://example.test"
    assert [json.loads(message) for message in websocket.sent_messages] == [
        {
            "event": "pusher:subscribe",
            "data": {"channel": "chatrooms.200.v2"},
        },
        {
            "event": "pusher:subscribe",
            "data": {"channel": "channel.100"},
        },
    ]


async def test_pusher_client_yields_text_messages_and_decodes_bytes() -> None:
    websocket = FakeWebSocket(messages=["text-event", b"bytes-event"])
    client = KickPusherClient("wss://example.test", connect=FakeConnector(websocket))
    channels = [
        ListenerChannelDTO(
            id=1,
            kick_channel_id=100,
            kick_chatroom_id=200,
            slug="hype",
            display_name="Hype",
        )
    ]

    received = [message async for message in client.listen(channels)]

    assert received == ["text-event", "bytes-event"]
