from collections.abc import AsyncIterator
from datetime import UTC, datetime, timedelta
from uuid import uuid4

import pytest
from httpx import ASGITransport, AsyncClient
from sqlalchemy import text
from sqlalchemy.ext.asyncio import async_sessionmaker, create_async_engine

from kick_logs.core.config import get_settings
from kick_logs.domain.entities import Channel, ChatMessage, Emote, Sender
from kick_logs.infrastructure.database import SqlAlchemyUnitOfWork
from kick_logs.presentation.http.app import create_app
from kick_logs.presentation.http.dependencies import get_unit_of_work_factory


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


@pytest.fixture
async def client(session_factory) -> AsyncIterator[AsyncClient]:
    app = create_app(seed_super_admin_on_startup=False)
    app.dependency_overrides[get_unit_of_work_factory] = lambda: (
        lambda: SqlAlchemyUnitOfWork(session_factory)
    )

    async with AsyncClient(
        transport=ASGITransport(app=app),
        base_url="http://testserver",
    ) as async_client:
        yield async_client

    app.dependency_overrides.clear()


async def seed_user_profile_dataset(session_factory) -> dict:
    suffix = uuid4().hex[:8]
    base_time = datetime(2041, 6, 1, 10, 0, tzinfo=UTC)

    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        channel_one = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=100000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=200000 + int(uuid4().hex[:6], 16),
                slug=f"profile-a-{suffix}",
                display_name=f"Profile A {suffix}",
                profile_image_url="https://example.com/profile-a.png",
                banner_image_url="https://example.com/profile-a-banner.png",
            )
        )
        channel_two = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=300000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=400000 + int(uuid4().hex[:6], 16),
                slug=f"profile-b-{suffix}",
                display_name=f"Profile B {suffix}",
            )
        )
        sender = await unit_of_work.senders.add(
            Sender(
                kick_user_id=500000 + int(uuid4().hex[:6], 16),
                username=f"ProfileUser{suffix}",
                slug=f"profile-user-{suffix}",
                profile_image_url="https://example.com/profile-user.png",
            )
        )
        other_sender = await unit_of_work.senders.add(
            Sender(
                kick_user_id=600000 + int(uuid4().hex[:6], 16),
                username=f"OtherProfileUser{suffix}",
                slug=f"other-profile-user-{suffix}",
            )
        )

        messages = [
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("profile-message"),
                    channel_id=channel_one.id or 0,
                    sender_id=sender.id or 0,
                    chatroom_id=channel_one.kick_chatroom_id or 0,
                    content=f"profile combo {suffix} first [emote:111:Kappa]",
                    message_type="message",
                    sender_username_snapshot=sender.username,
                    sender_slug_snapshot=sender.slug,
                    emotes=[_emote("111", "Kappa")],
                    message_created_at=base_time,
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("profile-message"),
                    channel_id=channel_one.id or 0,
                    sender_id=sender.id or 0,
                    chatroom_id=channel_one.kick_chatroom_id or 0,
                    content=f"profile combo {suffix} second",
                    message_type="message",
                    sender_username_snapshot=sender.username,
                    sender_slug_snapshot=sender.slug,
                    emotes=[_emote("111", "Kappa"), _emote("222", "Pog")],
                    message_created_at=base_time + timedelta(hours=2),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("profile-message"),
                    channel_id=channel_two.id or 0,
                    sender_id=sender.id or 0,
                    chatroom_id=channel_two.kick_chatroom_id or 0,
                    content=f"profile combo {suffix} newest",
                    message_type="message",
                    sender_username_snapshot=sender.username,
                    sender_slug_snapshot=sender.slug,
                    message_created_at=base_time + timedelta(days=1),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("profile-message"),
                    channel_id=channel_two.id or 0,
                    sender_id=other_sender.id or 0,
                    chatroom_id=channel_two.kick_chatroom_id or 0,
                    content=f"profile combo {suffix} unrelated",
                    message_type="message",
                    sender_username_snapshot=other_sender.username,
                    sender_slug_snapshot=other_sender.slug,
                    emotes=[_emote("333", "Other")],
                    message_created_at=base_time + timedelta(days=2),
                )
            ),
        ]
        await unit_of_work.commit()

    return {
        "base_time": base_time,
        "channel_one": channel_one,
        "channel_two": channel_two,
        "sender": sender,
        "messages": messages,
    }


async def test_public_user_profile_returns_identity_analytics_and_latest_messages(
    client,
    session_factory,
) -> None:
    dataset = await seed_user_profile_dataset(session_factory)

    response = await client.get(f"/users/{dataset['sender'].slug}/analytics")

    assert response.status_code == 200
    payload = response.json()
    assert payload["sender"] == {
        "id": dataset["sender"].id,
        "kick_user_id": dataset["sender"].kick_user_id,
        "username": dataset["sender"].username,
        "slug": dataset["sender"].slug,
        "profile_image_url": "https://example.com/profile-user.png",
    }
    assert payload["overview"]["total_messages"] == 3
    assert payload["overview"]["total_channels"] == 2
    assert payload["overview"]["total_senders"] == 1
    assert payload["overview"]["total_emote_usages"] == 3
    assert [
        (item["bucket_start"][:10], item["message_count"]) for item in payload["message_volume"]
    ] == [
        ("2041-06-01", 2),
        ("2041-06-02", 1),
    ]
    assert [(item["slug"], item["message_count"]) for item in payload["top_channels"]] == [
        (dataset["channel_one"].slug, 2),
        (dataset["channel_two"].slug, 1),
    ]
    assert [
        (item["id"], item["usage_count"], item["message_count"]) for item in payload["top_emotes"]
    ] == [
        ("111", 2, 2),
        ("222", 1, 1),
    ]
    assert [item["kick_message_id"] for item in payload["latest_messages"]] == [
        dataset["messages"][2].kick_message_id,
        dataset["messages"][1].kick_message_id,
        dataset["messages"][0].kick_message_id,
    ]


async def test_public_user_profile_returns_404_for_unknown_sender(client) -> None:
    response = await client.get("/users/unknown-profile-user/analytics")

    assert response.status_code == 404
    assert response.json()["detail"] == "Sender profile not found."


def unique_value(prefix: str) -> str:
    return f"{prefix}-{uuid4().hex[:10]}"


def _emote(emote_id: str, name: str) -> Emote:
    return Emote(kick_emote_id=emote_id, name=name, token=f"[emote:{emote_id}:{name}]")
