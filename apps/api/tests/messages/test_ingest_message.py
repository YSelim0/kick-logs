from collections.abc import AsyncIterator
from uuid import uuid4

import pytest
from sqlalchemy import text
from sqlalchemy.ext.asyncio import async_sessionmaker, create_async_engine

from kick_logs.application.exceptions import ChannelNotFoundError, MessageIngestionError
from kick_logs.application.use_cases.messages import IngestMessageUseCase
from kick_logs.core.config import get_settings
from kick_logs.domain.entities import Channel
from kick_logs.domain.value_objects.pagination import CursorPagination
from kick_logs.domain.value_objects.search_filters import MessageSearchFilters
from kick_logs.infrastructure.database import SqlAlchemyUnitOfWork


@pytest.fixture
async def session_factory() -> AsyncIterator[async_sessionmaker]:
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
        factory = async_sessionmaker(bind=connection, expire_on_commit=False)
        try:
            yield factory
        finally:
            if transaction.is_active:
                await transaction.rollback()
            await engine.dispose()


def unique_value(prefix: str) -> str:
    return f"{prefix}-{uuid4().hex[:10]}"


async def seed_channel(session_factory, chatroom_id: int) -> Channel:
    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        channel = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=100000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=chatroom_id,
                slug=unique_value("hype"),
                display_name="Hype",
            )
        )
        await unit_of_work.commit()
        return channel


def build_payload(chatroom_id: int, content: str | None = None) -> dict:
    marker = unique_value("marker")
    return {
        "id": unique_value("message"),
        "chatroom_id": chatroom_id,
        "content": content or f"hello {marker} [emote:37226:KEKW]",
        "type": "message",
        "created_at": "2026-05-10T01:02:03Z",
        "thread_parent_id": "thread-1",
        "sender": {
            "id": 990002,
            "username": "Yavuz",
            "slug": "Yavuz",
            "profile_pic": "https://example.com/avatar.png",
            "identity": {
                "color": "#fff600",
                "badges": [{"type": "moderator", "text": "Mod"}],
            },
        },
        "metadata": {
            "message_ref": "ref-1",
            "original_sender": {"username": "Other"},
            "original_message": {"content": "previous"},
        },
    }


async def test_ingest_message_persists_normalized_message(session_factory) -> None:
    chatroom_id = 700000 + int(uuid4().hex[:6], 16)
    channel = await seed_channel(session_factory, chatroom_id)
    payload = build_payload(chatroom_id)

    message = await IngestMessageUseCase(
        lambda: SqlAlchemyUnitOfWork(session_factory)
    ).execute(payload)

    assert message.channel_id == channel.id
    assert message.chatroom_id == chatroom_id
    assert message.content == payload["content"]
    assert message.sender_username_snapshot == "Yavuz"
    assert message.sender_slug_snapshot == "yavuz"
    assert message.sender_color_snapshot == "#fff600"
    assert message.sender_badges == [{"type": "moderator", "text": "Mod"}]
    assert message.reply_metadata["message_ref"] == "ref-1"
    assert message.thread_parent_id == "thread-1"
    assert message.emotes == [
        {
            "id": "37226",
            "name": "KEKW",
            "token": "[emote:37226:KEKW]",
            "image_url": "https://files.kick.com/emotes/37226/fullsize",
        }
    ]

    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        sender = await unit_of_work.senders.get_by_kick_user_id(990002)

    assert sender is not None
    assert sender.profile_image_url == "https://example.com/avatar.png"
    assert sender.last_seen_color == "#fff600"


async def test_ingest_message_persists_reply_payload_shape(session_factory) -> None:
    chatroom_id = 700000 + int(uuid4().hex[:6], 16)
    await seed_channel(session_factory, chatroom_id)
    payload = build_payload(chatroom_id, content="current reply content")
    payload["type"] = "reply"
    payload["thread_parent_id"] = "cad8a796-d688-4de1-9e13-2e0a4d0b5f1f"
    payload["metadata"] = {
        "original_sender": {
            "id": 97891494,
            "username": "Cansu98xx",
        },
        "original_message": {
            "id": "1be196b8-55c7-4980-8022-a1112723acea",
            "content": "senin saat ne saati 5dk 1 saatmiş",
        },
        "message_ref": "1778535344619",
    }

    message = await IngestMessageUseCase(
        lambda: SqlAlchemyUnitOfWork(session_factory)
    ).execute(payload)

    assert message.message_type == "reply"
    assert message.thread_parent_id == "cad8a796-d688-4de1-9e13-2e0a4d0b5f1f"
    assert message.reply_metadata["original_sender"]["username"] == "Cansu98xx"
    assert (
        message.reply_metadata["original_message"]["content"]
        == "senin saat ne saati 5dk 1 saatmiş"
    )


async def test_ingest_message_is_idempotent_by_kick_message_id(session_factory) -> None:
    chatroom_id = 700000 + int(uuid4().hex[:6], 16)
    await seed_channel(session_factory, chatroom_id)
    content = f"idempotent {unique_value('query')} [emote:1:A]"
    payload = build_payload(chatroom_id, content=content)
    use_case = IngestMessageUseCase(lambda: SqlAlchemyUnitOfWork(session_factory))

    first = await use_case.execute(payload)
    duplicate_payload = {**payload, "content": "changed content should not persist"}
    second = await use_case.execute(duplicate_payload)

    assert second.id == first.id
    assert second.content == content

    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        results = await unit_of_work.messages.search(
            MessageSearchFilters(q=content),
            CursorPagination(limit=10),
        )

    assert [message.id for message in results] == [first.id]


async def test_ingest_message_requires_followed_channel(session_factory) -> None:
    payload = build_payload(999999)

    with pytest.raises(ChannelNotFoundError):
        await IngestMessageUseCase(lambda: SqlAlchemyUnitOfWork(session_factory)).execute(payload)


async def test_ingest_message_rejects_invalid_payload(session_factory) -> None:
    with pytest.raises(MessageIngestionError):
        await IngestMessageUseCase(lambda: SqlAlchemyUnitOfWork(session_factory)).execute(
            {"id": "message-1"}
        )
