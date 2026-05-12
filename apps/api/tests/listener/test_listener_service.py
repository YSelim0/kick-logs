import asyncio
import json
from datetime import UTC, datetime, timedelta

from kick_logs.application.dto.channels import ResolvedKickChannelDTO
from kick_logs.application.dto.senders import ResolvedSenderProfileDTO
from kick_logs.application.use_cases.listener import ProcessRawKickEventsUseCase
from kick_logs.domain.entities import Channel, ChatMessage, RawKickEvent, Sender
from kick_logs.domain.value_objects.raw_event_status import RawEventStatus
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


class HangingPusherClient:
    def __init__(self) -> None:
        self.listen_call_count = 0

    async def listen(self, _channels):
        self.listen_call_count += 1
        while True:
            await asyncio.sleep(1)
            if False:
                yield ""


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


class FakeRawEventRepository:
    def __init__(self) -> None:
        self.events: list[RawKickEvent] = []

    async def add(self, event: RawKickEvent) -> RawKickEvent:
        event.id = len(self.events) + 1
        event.received_at = event.received_at or datetime.now(UTC)
        self.events.append(event)
        return event

    async def get_by_id(self, event_id: int) -> RawKickEvent | None:
        for event in self.events:
            if event.id == event_id:
                return event
        return None

    async def get_by_kick_message_id(self, kick_message_id: str) -> RawKickEvent | None:
        for event in self.events:
            if event.kick_message_id == kick_message_id:
                return event
        return None

    async def claim_pending(
        self,
        *,
        limit: int,
        processing_timeout_seconds: int,
    ) -> list[RawKickEvent]:
        stale_before = datetime.now(UTC) - timedelta(seconds=processing_timeout_seconds)
        claimed: list[RawKickEvent] = []
        for event in sorted(self.events, key=lambda item: item.received_at or datetime.now(UTC)):
            if len(claimed) >= limit:
                break
            is_pending = event.status == RawEventStatus.PENDING
            is_stale = (
                event.status == RawEventStatus.PROCESSING
                and event.processing_started_at is not None
                and event.processing_started_at < stale_before
            )
            if not is_pending and not is_stale:
                continue

            event.status = RawEventStatus.PROCESSING
            event.processing_started_at = datetime.now(UTC)
            claimed.append(event)
        return claimed

    async def mark_processed(self, event_id: int) -> RawKickEvent:
        event = await self.get_by_id(event_id)
        if event is None:
            raise ValueError("Raw Kick event not found.")
        event.status = RawEventStatus.PROCESSED
        event.processed_at = datetime.now(UTC)
        event.processing_started_at = None
        event.last_error = None
        return event

    async def mark_failed(
        self,
        *,
        event_id: int,
        error: str,
        max_attempts: int,
    ) -> RawKickEvent:
        event = await self.get_by_id(event_id)
        if event is None:
            raise ValueError("Raw Kick event not found.")
        event.attempts += 1
        event.status = (
            RawEventStatus.FAILED if event.attempts >= max_attempts else RawEventStatus.PENDING
        )
        event.processing_started_at = None
        event.last_error = error
        return event

    async def pending_count(self) -> int:
        return len([event for event in self.events if event.status == RawEventStatus.PENDING])


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
        self.raw_events = FakeRawEventRepository()

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
    channel_resync_interval_seconds: float = 60.0,
    raw_event_worker_count: int = 0,
) -> ListenerService:
    return ListenerService(
        unit_of_work_factory=lambda: unit_of_work,
        channel_resolver=FakeChannelResolver(),
        pusher_client=pusher_client,
        event_parser=KickEventParser(),
        sender_profile_resolver=sender_profile_resolver,
        reconnect_policy=ReconnectPolicy(initial_delay_seconds=0, max_delay_seconds=0),
        raw_event_worker_count=raw_event_worker_count,
        channel_resync_interval_seconds=channel_resync_interval_seconds,
        sleep=sleep or asyncio.sleep,
    )


async def test_listener_service_stores_fake_pusher_chat_event() -> None:
    unit_of_work = FakeUnitOfWork()
    pusher_client = FakePusherClient([build_chat_event()])
    service = build_service(unit_of_work, pusher_client, FakeSenderProfileResolver())

    stored_count = await service.run_once()

    assert stored_count == 1
    assert pusher_client.listen_call_count == 1
    assert unit_of_work.raw_events.events[0].kick_message_id == "message-1"
    assert unit_of_work.raw_events.events[0].chatroom_id == 123
    assert unit_of_work.raw_events.events[0].channel_id == 1
    assert unit_of_work.messages.messages == []


async def test_listener_service_ignores_malformed_events() -> None:
    unit_of_work = FakeUnitOfWork()
    pusher_client = FakePusherClient(["not-json", build_chat_event()])
    service = build_service(unit_of_work, pusher_client, FakeSenderProfileResolver())

    stored_count = await service.run_once()

    assert stored_count == 1
    assert len(unit_of_work.raw_events.events) == 1
    assert unit_of_work.messages.messages == []


async def test_raw_event_processor_ingests_stored_events() -> None:
    unit_of_work = FakeUnitOfWork()
    pusher_client = FakePusherClient([build_chat_event()])
    service = build_service(unit_of_work, pusher_client, FakeSenderProfileResolver())

    await service.run_once()
    result = await ProcessRawKickEventsUseCase(lambda: unit_of_work).execute_once(
        limit=10,
        processing_timeout_seconds=300,
        max_attempts=2,
    )

    assert result.claimed == 1
    assert result.processed == 1
    assert result.failed == 0
    assert unit_of_work.raw_events.events[0].status == RawEventStatus.PROCESSED
    assert unit_of_work.messages.messages[0].kick_message_id == "message-1"
    assert unit_of_work.messages.messages[0].content == "hello [emote:37226:KEKW]"
    assert unit_of_work.messages.messages[0].emotes[0].image_url == (
        "https://files.kick.com/emotes/37226/fullsize"
    )


async def test_raw_event_processor_keeps_message_writes_idempotent() -> None:
    unit_of_work = FakeUnitOfWork()
    event = KickEventParser().parse(build_chat_event("message-duplicate"))
    assert event is not None
    await unit_of_work.raw_events.add(
        RawKickEvent.pending(
            event_name=event.event,
            payload=event.payload,
            kick_message_id="message-duplicate",
            chatroom_id=123,
            channel_id=1,
        )
    )
    await unit_of_work.raw_events.add(
        RawKickEvent.pending(
            event_name=event.event,
            payload=event.payload,
            kick_message_id="message-duplicate",
            chatroom_id=123,
            channel_id=1,
        )
    )

    result = await ProcessRawKickEventsUseCase(lambda: unit_of_work).execute_once(
        limit=10,
        processing_timeout_seconds=300,
        max_attempts=2,
    )

    assert result.processed == 2
    assert len(unit_of_work.messages.messages) == 1
    assert all(
        raw_event.status == RawEventStatus.PROCESSED for raw_event in unit_of_work.raw_events.events
    )


async def test_raw_event_processor_retries_failed_events_with_payload_and_error() -> None:
    unit_of_work = FakeUnitOfWork()
    event = KickEventParser().parse(build_chat_event())
    assert event is not None
    event.payload["chatroom_id"] = 999
    await unit_of_work.raw_events.add(
        RawKickEvent.pending(
            event_name=event.event,
            payload=event.payload,
            kick_message_id="message-failure",
            chatroom_id=999,
        )
    )

    result = await ProcessRawKickEventsUseCase(lambda: unit_of_work).execute_once(
        limit=10,
        processing_timeout_seconds=300,
        max_attempts=1,
    )

    assert result.claimed == 1
    assert result.processed == 0
    assert result.failed == 1
    assert unit_of_work.raw_events.events[0].status == RawEventStatus.FAILED
    assert unit_of_work.raw_events.events[0].attempts == 1
    assert unit_of_work.raw_events.events[0].payload["chatroom_id"] == 999
    assert "ChannelNotFoundError" in (unit_of_work.raw_events.events[0].last_error or "")


async def test_listener_service_resyncs_channel_subscriptions() -> None:
    unit_of_work = FakeUnitOfWork()
    pusher_client = HangingPusherClient()
    service = build_service(
        unit_of_work,
        pusher_client,
        FakeSenderProfileResolver(),
        channel_resync_interval_seconds=0.01,
    )

    stored_count = await asyncio.wait_for(service.run_once(), timeout=1)

    assert stored_count == 0
    assert pusher_client.listen_call_count == 1


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
