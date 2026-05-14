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


async def seed_channel_profile_dataset(session_factory) -> dict:
    suffix = uuid4().hex[:8]
    base_time = datetime(2042, 7, 1, 10, 0, tzinfo=UTC)

    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        channel = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=100000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=200000 + int(uuid4().hex[:6], 16),
                slug=f"channel-profile-{suffix}",
                display_name=f"Channel Profile {suffix}",
                profile_image_url="https://example.com/channel-profile.png",
                banner_image_url="https://example.com/channel-banner.png",
            )
        )
        other_channel = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=300000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=400000 + int(uuid4().hex[:6], 16),
                slug=f"channel-profile-other-{suffix}",
                display_name=f"Other Channel Profile {suffix}",
            )
        )
        sender_alpha = await unit_of_work.senders.add(
            Sender(
                kick_user_id=500000 + int(uuid4().hex[:6], 16),
                username=f"ChannelAlpha{suffix}",
                slug=f"channel-alpha-{suffix}",
                profile_image_url="https://example.com/channel-alpha.png",
            )
        )
        sender_beta = await unit_of_work.senders.add(
            Sender(
                kick_user_id=600000 + int(uuid4().hex[:6], 16),
                username=f"ChannelBeta{suffix}",
                slug=f"channel-beta-{suffix}",
            )
        )

        messages = [
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("channel-profile-message"),
                    channel_id=channel.id or 0,
                    sender_id=sender_alpha.id or 0,
                    chatroom_id=channel.kick_chatroom_id or 0,
                    content=f"channel profile combo {suffix} first [emote:111:Kappa]",
                    message_type="message",
                    sender_username_snapshot=sender_alpha.username,
                    sender_slug_snapshot=sender_alpha.slug,
                    emotes=[_emote("111", "Kappa")],
                    message_created_at=base_time,
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("channel-profile-message"),
                    channel_id=channel.id or 0,
                    sender_id=sender_beta.id or 0,
                    chatroom_id=channel.kick_chatroom_id or 0,
                    content=f"channel profile combo {suffix} second [emote:222:Pog]",
                    message_type="message",
                    sender_username_snapshot=sender_beta.username,
                    sender_slug_snapshot=sender_beta.slug,
                    emotes=[_emote("222", "Pog")],
                    message_created_at=base_time + timedelta(hours=2),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("channel-profile-message"),
                    channel_id=channel.id or 0,
                    sender_id=sender_alpha.id or 0,
                    chatroom_id=channel.kick_chatroom_id or 0,
                    content=f"channel profile combo {suffix} newest [emote:111:Kappa]",
                    message_type="message",
                    sender_username_snapshot=sender_alpha.username,
                    sender_slug_snapshot=sender_alpha.slug,
                    emotes=[_emote("111", "Kappa")],
                    message_created_at=base_time + timedelta(days=1),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("channel-profile-message"),
                    channel_id=other_channel.id or 0,
                    sender_id=sender_beta.id or 0,
                    chatroom_id=other_channel.kick_chatroom_id or 0,
                    content=f"channel profile combo {suffix} unrelated",
                    message_type="message",
                    sender_username_snapshot=sender_beta.username,
                    sender_slug_snapshot=sender_beta.slug,
                    message_created_at=base_time + timedelta(days=2),
                )
            ),
        ]
        await unit_of_work.commit()

    return {
        "base_time": base_time,
        "channel": channel,
        "other_channel": other_channel,
        "sender_alpha": sender_alpha,
        "sender_beta": sender_beta,
        "messages": messages,
    }


async def test_public_channel_profile_returns_metadata_analytics_and_latest_messages(
    client,
    session_factory,
) -> None:
    dataset = await seed_channel_profile_dataset(session_factory)

    response = await client.get(f"/channels/{dataset['channel'].slug}/analytics")

    assert response.status_code == 200
    payload = response.json()
    assert payload["channel"] == {
        "id": dataset["channel"].id,
        "kick_channel_id": dataset["channel"].kick_channel_id,
        "kick_chatroom_id": dataset["channel"].kick_chatroom_id,
        "slug": dataset["channel"].slug,
        "display_name": dataset["channel"].display_name,
        "profile_image_url": "https://example.com/channel-profile.png",
        "banner_image_url": "https://example.com/channel-banner.png",
        "is_enabled": True,
    }
    assert payload["overview"]["total_messages"] == 3
    assert payload["overview"]["total_senders"] == 2
    assert payload["overview"]["total_channels"] == 1
    assert payload["overview"]["total_emote_usages"] == 3
    assert [
        (item["bucket_start"][:10], item["message_count"]) for item in payload["message_volume"]
    ] == [
        ("2042-07-01", 2),
        ("2042-07-02", 1),
    ]
    assert [(item["slug"], item["message_count"]) for item in payload["top_senders"]] == [
        (dataset["sender_alpha"].slug, 2),
        (dataset["sender_beta"].slug, 1),
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


async def test_public_channel_profile_returns_404_for_unknown_channel(client) -> None:
    response = await client.get("/channels/unknown-channel-profile/analytics")

    assert response.status_code == 404
    assert response.json()["detail"] == "Channel profile not found."


def unique_value(prefix: str) -> str:
    return f"{prefix}-{uuid4().hex[:10]}"


def _emote(emote_id: str, name: str) -> Emote:
    return Emote(kick_emote_id=emote_id, name=name, token=f"[emote:{emote_id}:{name}]")
