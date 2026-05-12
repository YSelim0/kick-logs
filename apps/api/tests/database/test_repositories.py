from collections.abc import AsyncIterator
from datetime import UTC, datetime, timedelta
from uuid import uuid4

import pytest
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine

from kick_logs.core.config import get_settings
from kick_logs.domain.entities import Channel, ChatMessage, RawKickEvent, Sender, User
from kick_logs.domain.value_objects.pagination import CursorPagination, MessageCursor
from kick_logs.domain.value_objects.raw_event_status import RawEventStatus
from kick_logs.domain.value_objects.roles import UserRole
from kick_logs.domain.value_objects.search_filters import MessageSearchFilters
from kick_logs.infrastructure.database.repositories import (
    SqlAlchemyChannelRepository,
    SqlAlchemyMessageRepository,
    SqlAlchemyRawEventRepository,
    SqlAlchemySenderRepository,
    SqlAlchemyUserRepository,
)
from kick_logs.infrastructure.database.session import create_session_factory
from kick_logs.infrastructure.database.unit_of_work import SqlAlchemyUnitOfWork


@pytest.fixture
async def db_session() -> AsyncIterator[AsyncSession]:
    engine = create_async_engine(get_settings().database_url, pool_pre_ping=True)

    try:
        async with engine.connect() as healthcheck:
            table_exists = await healthcheck.scalar(
                text("select to_regclass('public.chat_messages')")
            )
            if table_exists is None:
                pytest.skip("Database schema is not migrated.")
    except OSError:
        pytest.skip("PostgreSQL is not available.")

    async with engine.connect() as connection:
        transaction = await connection.begin()
        session = AsyncSession(bind=connection, expire_on_commit=False)
        try:
            yield session
        finally:
            await session.close()
            await transaction.rollback()
            await engine.dispose()


def unique_value(prefix: str) -> str:
    return f"{prefix}-{uuid4().hex[:10]}"


async def create_channel_and_sender(
    db_session: AsyncSession,
) -> tuple[Channel, Sender]:
    channel = await SqlAlchemyChannelRepository(db_session).add(
        Channel(
            slug=unique_value("hype"),
            display_name="Hype",
            kick_channel_id=100000 + int(uuid4().hex[:6], 16),
            kick_chatroom_id=200000 + int(uuid4().hex[:6], 16),
        )
    )
    sender = await SqlAlchemySenderRepository(db_session).add(
        Sender(
            kick_user_id=300000 + int(uuid4().hex[:6], 16),
            username="Yavuz",
            slug=unique_value("yavuz"),
            profile_image_url="https://example.com/avatar.png",
        )
    )
    return channel, sender


@pytest.mark.asyncio
async def test_user_repository_create_read_update(db_session: AsyncSession) -> None:
    repository = SqlAlchemyUserRepository(db_session)
    user = await repository.add(
        User(
            email=f"{unique_value('admin')}@example.com",
            password_hash="hash",
            role=UserRole.ADMIN,
        )
    )

    loaded = await repository.get_by_email(user.email)
    assert loaded is not None
    assert loaded.id == user.id

    loaded.deactivate()
    updated = await repository.update(loaded)

    active_user_ids = [active_user.id for active_user in await repository.list_active()]
    assert updated.is_active is False
    assert updated.id not in active_user_ids


@pytest.mark.asyncio
async def test_channel_repository_create_read_update(db_session: AsyncSession) -> None:
    repository = SqlAlchemyChannelRepository(db_session)
    channel = await repository.add(Channel(slug=unique_value("channel"), display_name="Channel"))

    loaded = await repository.get_by_slug(channel.slug)
    assert loaded is not None
    assert loaded.id == channel.id

    loaded.disable()
    updated = await repository.update(loaded)

    enabled_channel_ids = [
        enabled_channel.id for enabled_channel in await repository.list_enabled()
    ]
    assert updated.is_enabled is False
    assert updated.id not in enabled_channel_ids


@pytest.mark.asyncio
async def test_sender_repository_create_read_update(db_session: AsyncSession) -> None:
    repository = SqlAlchemySenderRepository(db_session)
    sender = await repository.add(
        Sender(
            kick_user_id=300000 + int(uuid4().hex[:6], 16),
            username="Yavuz",
            slug=unique_value("yavuz"),
        )
    )

    loaded = await repository.get_by_kick_user_id(sender.kick_user_id)
    assert loaded is not None
    assert loaded.id == sender.id

    loaded.profile_image_url = "https://example.com/new.png"
    updated = await repository.update(loaded)

    assert updated.profile_image_url == "https://example.com/new.png"


@pytest.mark.asyncio
async def test_message_repository_create_read_and_search(db_session: AsyncSession) -> None:
    channel, sender = await create_channel_and_sender(db_session)
    repository = SqlAlchemyMessageRepository(db_session)
    now = datetime.now(UTC)
    search_term = unique_value("hello")

    old_message = await repository.add(
        ChatMessage(
            kick_message_id=unique_value("message"),
            channel_id=channel.id or 0,
            sender_id=sender.id or 0,
            chatroom_id=channel.kick_chatroom_id or 1,
            content=f"older {search_term} message",
            message_type="message",
            sender_username_snapshot=sender.username,
            sender_slug_snapshot=sender.slug,
            message_created_at=now - timedelta(minutes=10),
        )
    )
    newest_message = await repository.add(
        ChatMessage(
            kick_message_id=unique_value("message"),
            channel_id=channel.id or 0,
            sender_id=sender.id or 0,
            chatroom_id=channel.kick_chatroom_id or 1,
            content=f"newest {search_term} message",
            message_type="message",
            sender_username_snapshot=sender.username,
            sender_slug_snapshot=sender.slug,
            message_created_at=now,
        )
    )

    loaded = await repository.get_by_kick_message_id(newest_message.kick_message_id)
    assert loaded is not None
    assert loaded.id == newest_message.id

    results = await repository.search(
        MessageSearchFilters(sender="yav", channel=channel.slug, q=search_term),
        CursorPagination(limit=10),
    )
    assert [message.id for message in results] == [newest_message.id, old_message.id]

    paged = await repository.search(
        MessageSearchFilters(q=search_term),
        CursorPagination(
            limit=10,
            cursor=MessageCursor(
                message_created_at=newest_message.message_created_at,
                message_id=newest_message.id or 0,
            ),
        ),
    )
    assert [message.id for message in paged] == [old_message.id]


@pytest.mark.asyncio
async def test_raw_event_repository_claim_and_mark_processed(db_session: AsyncSession) -> None:
    channel, _sender = await create_channel_and_sender(db_session)
    repository = SqlAlchemyRawEventRepository(db_session)

    event = await repository.add(
        RawKickEvent.pending(
            event_name=r"App\Events\ChatMessageEvent",
            kick_message_id=unique_value("raw-message"),
            chatroom_id=channel.kick_chatroom_id,
            kick_channel_id=channel.kick_channel_id,
            channel_id=channel.id,
            payload={
                "id": unique_value("payload-message"),
                "chatroom_id": channel.kick_chatroom_id,
            },
        )
    )

    loaded = await repository.get_by_kick_message_id(event.kick_message_id or "")
    assert loaded is not None
    assert loaded.id == event.id

    claimed = await repository.claim_pending(limit=10, processing_timeout_seconds=300)
    assert [claimed_event.id for claimed_event in claimed] == [event.id]
    assert claimed[0].status == RawEventStatus.PROCESSING
    assert claimed[0].processing_started_at is not None

    processed = await repository.mark_processed(event.id or 0)
    assert processed.status == RawEventStatus.PROCESSED
    assert processed.processed_at is not None


@pytest.mark.asyncio
async def test_raw_event_repository_requeues_stale_processing_rows(
    db_session: AsyncSession,
) -> None:
    repository = SqlAlchemyRawEventRepository(db_session)
    stale_event = await repository.add(
        RawKickEvent(
            event_name=r"App\Events\ChatMessageEvent",
            kick_message_id=unique_value("stale-message"),
            chatroom_id=123,
            payload={"id": unique_value("payload-message"), "chatroom_id": 123},
            status=RawEventStatus.PROCESSING,
            received_at=datetime.now(UTC) - timedelta(minutes=10),
            processing_started_at=datetime.now(UTC) - timedelta(minutes=10),
        )
    )

    claimed = await repository.claim_pending(limit=10, processing_timeout_seconds=60)

    assert [claimed_event.id for claimed_event in claimed] == [stale_event.id]
    assert claimed[0].status == RawEventStatus.PROCESSING
    assert claimed[0].processing_started_at is not None


@pytest.mark.asyncio
async def test_raw_event_repository_marks_failed_with_retry_state(
    db_session: AsyncSession,
) -> None:
    repository = SqlAlchemyRawEventRepository(db_session)
    event = await repository.add(
        RawKickEvent.pending(
            event_name=r"App\Events\ChatMessageEvent",
            kick_message_id=unique_value("failed-message"),
            chatroom_id=123,
            payload={"id": unique_value("payload-message"), "chatroom_id": 123},
        )
    )

    retryable = await repository.mark_failed(
        event_id=event.id or 0,
        error="temporary failure",
        max_attempts=2,
    )
    assert retryable.status == RawEventStatus.PENDING
    assert retryable.attempts == 1
    assert retryable.last_error == "temporary failure"

    final = await repository.mark_failed(
        event_id=event.id or 0,
        error="permanent failure",
        max_attempts=2,
    )
    assert final.status == RawEventStatus.FAILED
    assert final.attempts == 2
    assert final.last_error == "permanent failure"


@pytest.mark.asyncio
async def test_unit_of_work_rollback(db_session: AsyncSession) -> None:
    email = f"{unique_value('rollback')}@example.com"

    async with SqlAlchemyUnitOfWork(create_session_factory()) as uow:
        await uow.users.add(User(email=email, password_hash="hash", role=UserRole.ADMIN))
        await uow.rollback()

    assert await SqlAlchemyUserRepository(db_session).get_by_email(email) is None
