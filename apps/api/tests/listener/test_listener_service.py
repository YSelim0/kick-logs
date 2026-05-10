import asyncio
import json

from kick_logs.application.dto.channels import ResolvedKickChannelDTO
from kick_logs.application.dto.senders import ResolvedSenderProfileDTO
from kick_logs.application.exceptions import SenderProfileResolutionError
from kick_logs.domain.entities import Channel, ChatMessage, Sender
from kick_logs.infrastructure.kick import KickEventParser, ReconnectPolicy
from kick_logs.presentation.worker.listener_service import ListenerService


def build_chat_event(message_id: str = "message-1") -> str:
    return json.dumps(
        {
            "event": r"App\Events\ChatMessageEvent",
            "channel": "chatrooms.123.v2",
            "data": json.dumps(
                {
                    "id": message_id,
                    "chatroom_id": 123,
                    "content": "hello [emote:37226:KEKW]",
                    "type": "message",
                    "created_at": "2026-05-10T01:02:03Z",
                    "sender": {
                        "id": 456,
                        "username": "Yavuz",
                        "slug": "yavuz",
                        "identity": {
                            "color": "#fff600",
                            "badges": [{"type": "moderator"}],
                        },
                    },
                    "metadata": {"message_ref": "ref-1"},
                }
            ),
        }
    )


class FakePusherClient:
    def __init__(self, events: list[str]) -> None:
        self.events = events
        self.listen_call_count = 0

    async def listen(self, _channels):
        self.listen_call_count += 1
        for event in self.events:
            yield event


class FailingPusherClient:
    def __init__(self) -> None:
        self.listen_call_count = 0

    async def listen(self, _channels):
        self.listen_call_count += 1
        if False:
            yield ""
        raise RuntimeError("websocket failure")


class FakeChannelResolver:
    async def resolve(self, slug: str) -> ResolvedKickChannelDTO:
        return ResolvedKickChannelDTO(
            kick_channel_id=100,
            kick_chatroom_id=123,
            slug=slug,
            display_name="Hype",
        )


class FakeSenderProfileResolver:
    async def resolve(self, slug: str) -> ResolvedSenderProfileDTO:
        return ResolvedSenderProfileDTO(
            slug=slug,
            username="Yavuz",
            profile_image_url="https://example.com/avatar.png",
            raw_payload={"user": {"profile_pic": "https://example.com/avatar.png"}},
        )


class FailingSenderProfileResolver:
    async def resolve(self, _slug: str) -> ResolvedSenderProfileDTO:
        raise SenderProfileResolutionError("failed")


class FakeChannelRepository:
    def __init__(self, channels: list[Channel]) -> None:
        self.channels = channels

    async def list_enabled(self) -> list[Channel]:
        return [channel for channel in self.channels if channel.is_enabled]

    async def update(self, channel: Channel) -> Channel:
        return channel

    async def get_by_chatroom_id(self, kick_chatroom_id: int) -> Channel | None:
        for channel in self.channels:
            if channel.kick_chatroom_id == kick_chatroom_id:
                return channel
        return None


class FakeSenderRepository:
    def __init__(self) -> None:
        self.senders: list[Sender] = []

    async def get_by_kick_user_id(self, kick_user_id: int) -> Sender | None:
        for sender in self.senders:
            if sender.kick_user_id == kick_user_id:
                return sender
        return None

    async def get_by_slug(self, slug: str) -> Sender | None:
        for sender in self.senders:
            if sender.slug == slug:
                return sender
        return None

    async def add(self, sender: Sender) -> Sender:
        sender.id = len(self.senders) + 1
        self.senders.append(sender)
        return sender

    async def update(self, sender: Sender) -> Sender:
        return sender


class FakeMessageRepository:
    def __init__(self) -> None:
        self.messages: list[ChatMessage] = []

    async def get_by_kick_message_id(self, kick_message_id: str) -> ChatMessage | None:
        for message in self.messages:
            if message.kick_message_id == kick_message_id:
                return message
        return None

    async def add(self, message: ChatMessage) -> ChatMessage:
        message.id = len(self.messages) + 1
        self.messages.append(message)
        return message


class FakeUnitOfWork:
    def __init__(self) -> None:
        self.channels = FakeChannelRepository(
            [
                Channel(
                    id=1,
                    slug="hype",
                    display_name="Hype",
                    kick_channel_id=100,
                    kick_chatroom_id=123,
                )
            ]
        )
        self.senders = FakeSenderRepository()
        self.messages = FakeMessageRepository()

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc, traceback) -> None:
        return None

    async def commit(self) -> None:
        return None

    async def rollback(self) -> None:
        return None


def build_service(
    unit_of_work: FakeUnitOfWork,
    pusher_client,
    sender_profile_resolver,
    sleep=None,
) -> ListenerService:
    return ListenerService(
        unit_of_work_factory=lambda: unit_of_work,
        channel_resolver=FakeChannelResolver(),
        pusher_client=pusher_client,
        event_parser=KickEventParser(),
        sender_profile_resolver=sender_profile_resolver,
        reconnect_policy=ReconnectPolicy(initial_delay_seconds=0, max_delay_seconds=0),
        sleep=sleep or asyncio.sleep,
    )


async def test_listener_service_ingests_fake_pusher_chat_event() -> None:
    unit_of_work = FakeUnitOfWork()
    pusher_client = FakePusherClient([build_chat_event()])
    service = build_service(unit_of_work, pusher_client, FakeSenderProfileResolver())

    ingested_count = await service.run_once()

    assert ingested_count == 1
    assert pusher_client.listen_call_count == 1
    assert unit_of_work.messages.messages[0].kick_message_id == "message-1"
    assert unit_of_work.messages.messages[0].content == "hello [emote:37226:KEKW]"
    assert unit_of_work.messages.messages[0].emotes[0].image_url == (
        "https://files.kick.com/emotes/37226/fullsize"
    )
    assert unit_of_work.senders.senders[0].profile_image_url == "https://example.com/avatar.png"


async def test_listener_service_ignores_malformed_events() -> None:
    unit_of_work = FakeUnitOfWork()
    pusher_client = FakePusherClient(["not-json", build_chat_event()])
    service = build_service(unit_of_work, pusher_client, FakeSenderProfileResolver())

    ingested_count = await service.run_once()

    assert ingested_count == 1
    assert len(unit_of_work.messages.messages) == 1


async def test_listener_service_continues_when_sender_enrichment_fails() -> None:
    unit_of_work = FakeUnitOfWork()
    pusher_client = FakePusherClient([build_chat_event()])
    service = build_service(unit_of_work, pusher_client, FailingSenderProfileResolver())

    ingested_count = await service.run_once()

    assert ingested_count == 1
    assert unit_of_work.senders.senders[0].profile_image_url is None


async def test_listener_service_schedules_reconnect_after_pusher_failure() -> None:
    unit_of_work = FakeUnitOfWork()
    pusher_client = FailingPusherClient()
    delays: list[float] = []

    async def stop_after_first_sleep(delay: float) -> None:
        delays.append(delay)
        raise asyncio.CancelledError

    service = build_service(
        unit_of_work,
        pusher_client,
        FakeSenderProfileResolver(),
        sleep=stop_after_first_sleep,
    )

    try:
        await service.run_forever()
    except asyncio.CancelledError:
        pass

    assert pusher_client.listen_call_count == 1
    assert delays == [0]
