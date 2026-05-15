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


def unique_value(prefix: str) -> str:
    return f"{prefix}-{uuid4().hex[:10]}"


async def seed_analytics_dataset(session_factory) -> dict:
    suffix = uuid4().hex[:8]
    base_time = datetime(2040, 5, 1, 10, 0, tzinfo=UTC)

    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        channel_one = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=100000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=200000 + int(uuid4().hex[:6], 16),
                slug=f"analytics-a-{suffix}",
                display_name=f"Analytics A {suffix}",
                profile_image_url="https://example.com/channel-a.png",
                banner_image_url="https://example.com/banner-a.png",
            )
        )
        channel_two = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=300000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=400000 + int(uuid4().hex[:6], 16),
                slug=f"analytics-b-{suffix}",
                display_name=f"Analytics B {suffix}",
            )
        )
        sender_alpha = await unit_of_work.senders.add(
            Sender(
                kick_user_id=500000 + int(uuid4().hex[:6], 16),
                username=f"Alpha{suffix}",
                slug=f"alpha-{suffix}",
                profile_image_url="https://example.com/alpha.png",
            )
        )
        sender_beta = await unit_of_work.senders.add(
            Sender(
                kick_user_id=600000 + int(uuid4().hex[:6], 16),
                username=f"Beta{suffix}",
                slug=f"beta-{suffix}",
            )
        )
        sender_gamma = await unit_of_work.senders.add(
            Sender(
                kick_user_id=700000 + int(uuid4().hex[:6], 16),
                username=f"Gamma{suffix}",
                slug=f"gamma-{suffix}",
            )
        )

        messages = [
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel_one.id or 0,
                    sender_id=sender_alpha.id or 0,
                    chatroom_id=channel_one.kick_chatroom_id or 0,
                    content=f"analytics combo {suffix} alpha one",
                    message_type="message",
                    sender_username_snapshot=sender_alpha.username,
                    sender_slug_snapshot=sender_alpha.slug,
                    emotes=[_emote("111", "Kappa")],
                    message_created_at=base_time,
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel_one.id or 0,
                    sender_id=sender_alpha.id or 0,
                    chatroom_id=channel_one.kick_chatroom_id or 0,
                    content=f"analytics combo {suffix} alpha two",
                    message_type="message",
                    sender_username_snapshot=sender_alpha.username,
                    sender_slug_snapshot=sender_alpha.slug,
                    emotes=[_emote("111", "Kappa"), _emote("222", "Pog")],
                    message_created_at=base_time + timedelta(hours=1),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel_one.id or 0,
                    sender_id=sender_beta.id or 0,
                    chatroom_id=channel_one.kick_chatroom_id or 0,
                    content=f"analytics combo {suffix} beta one",
                    message_type="message",
                    sender_username_snapshot=sender_beta.username,
                    sender_slug_snapshot=sender_beta.slug,
                    message_created_at=base_time + timedelta(days=1),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel_two.id or 0,
                    sender_id=sender_beta.id or 0,
                    chatroom_id=channel_two.kick_chatroom_id or 0,
                    content=f"analytics combo {suffix} beta two",
                    message_type="message",
                    sender_username_snapshot=sender_beta.username,
                    sender_slug_snapshot=sender_beta.slug,
                    emotes=[_emote("222", "Pog")],
                    message_created_at=base_time + timedelta(days=1, hours=1),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel_two.id or 0,
                    sender_id=sender_gamma.id or 0,
                    chatroom_id=channel_two.kick_chatroom_id or 0,
                    content=f"analytics combo {suffix} gamma",
                    message_type="message",
                    sender_username_snapshot=sender_gamma.username,
                    sender_slug_snapshot=sender_gamma.slug,
                    emotes=[_emote("111", "Kappa")],
                    message_created_at=base_time + timedelta(days=2),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel_one.id or 0,
                    sender_id=sender_alpha.id or 0,
                    chatroom_id=channel_one.kick_chatroom_id or 0,
                    content=f"analytics combo {suffix} alpha three",
                    message_type="message",
                    sender_username_snapshot=sender_alpha.username,
                    sender_slug_snapshot=sender_alpha.slug,
                    message_created_at=base_time + timedelta(days=2, hours=1),
                )
            ),
        ]
        await unit_of_work.commit()

    return {
        "suffix": suffix,
        "base_time": base_time,
        "channel_one": channel_one,
        "channel_two": channel_two,
        "sender_alpha": sender_alpha,
        "sender_beta": sender_beta,
        "sender_gamma": sender_gamma,
        "messages": messages,
    }


def _emote(emote_id: str, name: str) -> Emote:
    return Emote(kick_emote_id=emote_id, name=name, token=f"[emote:{emote_id}:{name}]")


def _date(value: str) -> str:
    return value[:10]


async def test_public_analytics_overview_returns_filtered_counts(
    client,
    session_factory,
) -> None:
    dataset = await seed_analytics_dataset(session_factory)
    base_time = dataset["base_time"]

    response = await client.get(
        "/analytics/overview",
        params={
            "start": base_time.isoformat(),
            "end": (base_time + timedelta(days=3)).isoformat(),
        },
    )

    assert response.status_code == 200
    payload = response.json()
    assert payload["total_messages"] == 6
    assert payload["total_senders"] == 3
    assert payload["total_channels"] == 2
    assert payload["total_emote_usages"] == 5
    assert payload["first_message_at"].startswith("2040-05-01T10:00:00")
    assert payload["latest_message_at"].startswith("2040-05-03T11:00:00")


async def test_public_analytics_returns_empty_dataset_shapes(client) -> None:
    response = await client.get(
        "/analytics/overview",
        params={
            "start": datetime(2099, 1, 1, tzinfo=UTC).isoformat(),
            "end": datetime(2099, 1, 2, tzinfo=UTC).isoformat(),
        },
    )
    senders_response = await client.get(
        "/analytics/top-senders",
        params={
            "start": datetime(2099, 1, 1, tzinfo=UTC).isoformat(),
            "end": datetime(2099, 1, 2, tzinfo=UTC).isoformat(),
        },
    )

    assert response.status_code == 200
    assert response.json() == {
        "total_messages": 0,
        "total_senders": 0,
        "total_channels": 0,
        "total_emote_usages": 0,
        "first_message_at": None,
        "latest_message_at": None,
    }
    assert senders_response.status_code == 200
    assert senders_response.json() == {"items": []}


async def test_public_message_volume_supports_day_bucket_and_channel_scope(
    client,
    session_factory,
) -> None:
    dataset = await seed_analytics_dataset(session_factory)
    base_time = dataset["base_time"]

    response = await client.get(
        "/analytics/message-volume",
        params={
            "start": base_time.isoformat(),
            "end": (base_time + timedelta(days=3)).isoformat(),
            "channel": dataset["channel_one"].slug,
            "bucket": "day",
        },
    )

    assert response.status_code == 200
    assert [
        (_date(item["bucket_start"]), item["message_count"]) for item in response.json()["items"]
    ] == [
        ("2040-05-01", 2),
        ("2040-05-02", 1),
        ("2040-05-03", 1),
    ]


async def test_public_top_analytics_support_scopes_and_limits(
    client,
    session_factory,
) -> None:
    dataset = await seed_analytics_dataset(session_factory)
    base_time = dataset["base_time"]

    top_senders = await client.get(
        "/analytics/top-senders",
        params={
            "start": base_time.isoformat(),
            "end": (base_time + timedelta(days=3)).isoformat(),
            "limit": 2,
        },
    )
    top_channels_for_sender = await client.get(
        "/analytics/top-channels",
        params={
            "start": base_time.isoformat(),
            "end": (base_time + timedelta(days=3)).isoformat(),
            "sender": dataset["sender_alpha"].slug,
            "limit": 1,
        },
    )
    top_emotes_for_channel = await client.get(
        "/analytics/top-emotes",
        params={
            "start": base_time.isoformat(),
            "end": (base_time + timedelta(days=3)).isoformat(),
            "channel": dataset["channel_one"].slug,
        },
    )
    top_emotes_for_sender_channel = await client.get(
        "/analytics/top-emotes",
        params={
            "start": base_time.isoformat(),
            "end": (base_time + timedelta(days=3)).isoformat(),
            "sender": dataset["sender_beta"].slug,
            "channel": dataset["channel_two"].slug,
        },
    )

    assert top_senders.status_code == 200
    assert [(item["slug"], item["message_count"]) for item in top_senders.json()["items"]] == [
        (dataset["sender_alpha"].slug, 3),
        (dataset["sender_beta"].slug, 2),
    ]
    assert top_channels_for_sender.status_code == 200
    assert top_channels_for_sender.json()["items"] == [
        {
            "channel_id": dataset["channel_one"].id,
            "slug": dataset["channel_one"].slug,
            "display_name": dataset["channel_one"].display_name,
            "profile_image_url": "https://example.com/channel-a.png",
            "banner_image_url": "https://example.com/banner-a.png",
            "message_count": 3,
            "first_message_at": top_channels_for_sender.json()["items"][0]["first_message_at"],
            "latest_message_at": top_channels_for_sender.json()["items"][0]["latest_message_at"],
        }
    ]
    assert [
        (item["id"], item["name"], item["usage_count"], item["message_count"])
        for item in top_emotes_for_channel.json()["items"]
    ] == [
        ("111", "Kappa", 2, 2),
        ("222", "Pog", 1, 1),
    ]
    assert top_emotes_for_sender_channel.json()["items"][0]["id"] == "222"
    assert top_emotes_for_sender_channel.json()["items"][0]["usage_count"] == 1


async def test_public_analytics_rejects_invalid_date_range(client) -> None:
    response = await client.get(
        "/analytics/overview",
        params={
            "start": datetime(2040, 1, 2, tzinfo=UTC).isoformat(),
            "end": datetime(2040, 1, 1, tzinfo=UTC).isoformat(),
        },
    )

    assert response.status_code == 422
