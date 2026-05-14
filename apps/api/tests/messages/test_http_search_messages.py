import csv
from collections.abc import AsyncIterator
from datetime import UTC, datetime, timedelta
from io import StringIO
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


async def seed_search_dataset(session_factory) -> dict:
    suffix = uuid4().hex[:8]
    base_time = datetime(2035, 1, 1, 12, 0, tzinfo=UTC)

    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        channel_one = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=100000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=200000 + int(uuid4().hex[:6], 16),
                slug=f"example-{suffix}",
                display_name=f"Example {suffix}",
                profile_image_url="https://example.com/channel-one.png",
                banner_image_url="https://example.com/banner-one.png",
            )
        )
        channel_two = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=300000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=400000 + int(uuid4().hex[:6], 16),
                slug=f"other-{suffix}",
                display_name=f"Other {suffix}",
            )
        )
        sender_yavuz = await unit_of_work.senders.add(
            Sender(
                kick_user_id=500000 + int(uuid4().hex[:6], 16),
                username=f"Yavuz{suffix}",
                slug=f"yavuz-{suffix}",
                profile_image_url="https://example.com/yavuz.png",
                last_seen_color="#fff600",
            )
        )
        sender_other = await unit_of_work.senders.add(
            Sender(
                kick_user_id=600000 + int(uuid4().hex[:6], 16),
                username=f"Other{suffix}",
                slug=f"other-user-{suffix}",
            )
        )

        messages = [
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel_one.id or 0,
                    sender_id=sender_yavuz.id or 0,
                    chatroom_id=channel_one.kick_chatroom_id or 0,
                    content=f"selam combo-{suffix} [emote:37226:KEKW]",
                    message_type="message",
                    sender_username_snapshot=sender_yavuz.username,
                    sender_slug_snapshot=sender_yavuz.slug,
                    sender_color_snapshot="#fff600",
                    sender_badges=[{"type": "founder"}],
                    emotes=[
                        Emote(
                            kick_emote_id="37226",
                            name="KEKW",
                            token="[emote:37226:KEKW]",
                        )
                    ],
                    message_created_at=base_time + timedelta(minutes=3),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel_two.id or 0,
                    sender_id=sender_yavuz.id or 0,
                    chatroom_id=channel_two.kick_chatroom_id or 0,
                    content=f"other text combo-{suffix}",
                    message_type="message",
                    sender_username_snapshot=sender_yavuz.username,
                    sender_slug_snapshot=sender_yavuz.slug,
                    message_created_at=base_time + timedelta(minutes=2),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel_one.id or 0,
                    sender_id=sender_other.id or 0,
                    chatroom_id=channel_one.kick_chatroom_id or 0,
                    content=f"hello combo-{suffix}",
                    message_type="message",
                    sender_username_snapshot=sender_other.username,
                    sender_slug_snapshot=sender_other.slug,
                    message_created_at=base_time + timedelta(minutes=1),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel_two.id or 0,
                    sender_id=sender_other.id or 0,
                    chatroom_id=channel_two.kick_chatroom_id or 0,
                    content=f"hello combo-{suffix} older",
                    message_type="message",
                    sender_username_snapshot=sender_other.username,
                    sender_slug_snapshot=sender_other.slug,
                    message_created_at=base_time,
                )
            ),
        ]
        await unit_of_work.commit()

    return {
        "suffix": suffix,
        "base_time": base_time,
        "channel_one": channel_one,
        "channel_two": channel_two,
        "sender_yavuz": sender_yavuz,
        "sender_other": sender_other,
        "messages": messages,
    }


async def seed_search_improvement_dataset(session_factory) -> dict:
    suffix = uuid4().hex[:8]
    base_time = datetime(2036, 1, 1, 12, 0, tzinfo=UTC)
    reply_metadata = {
        "original_sender": {"id": 101, "username": f"Original{suffix}"},
        "original_message": {
            "id": unique_value("reply-source"),
            "content": f"original reply text combo-{suffix}",
        },
    }

    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        channel = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=700000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=800000 + int(uuid4().hex[:6], 16),
                slug=f"improve-{suffix}",
                display_name=f"Improve {suffix}",
            )
        )
        sender = await unit_of_work.senders.add(
            Sender(
                kick_user_id=900000 + int(uuid4().hex[:6], 16),
                username=f"ImproveSender{suffix}",
                slug=f"improve-sender-{suffix}",
            )
        )

        messages = [
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel.id or 0,
                    sender_id=sender.id or 0,
                    chatroom_id=channel.kick_chatroom_id or 0,
                    content=f"reply emote combo-{suffix} https://example.com/reply",
                    message_type="reply",
                    sender_username_snapshot=sender.username,
                    sender_slug_snapshot=sender.slug,
                    emotes=[
                        Emote(
                            kick_emote_id="111",
                            name="ReplyEmote",
                            token="[emote:111:ReplyEmote]",
                        )
                    ],
                    reply_metadata=reply_metadata,
                    thread_parent_id="parent-message-id",
                    message_created_at=base_time + timedelta(minutes=2),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel.id or 0,
                    sender_id=sender.id or 0,
                    chatroom_id=channel.kick_chatroom_id or 0,
                    content=f"plain emote combo-{suffix} [emote:222:PlainEmote]",
                    message_type="message",
                    sender_username_snapshot=sender.username,
                    sender_slug_snapshot=sender.slug,
                    emotes=[
                        Emote(
                            kick_emote_id="222",
                            name="PlainEmote",
                            token="[emote:222:PlainEmote]",
                        )
                    ],
                    message_created_at=base_time + timedelta(minutes=1),
                )
            ),
            await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=unique_value("message"),
                    channel_id=channel.id or 0,
                    sender_id=sender.id or 0,
                    chatroom_id=channel.kick_chatroom_id or 0,
                    content=f"plain text combo-{suffix}",
                    message_type="message",
                    sender_username_snapshot=sender.username,
                    sender_slug_snapshot=sender.slug,
                    message_created_at=base_time,
                )
            ),
        ]
        await unit_of_work.commit()

    return {
        "suffix": suffix,
        "base_time": base_time,
        "channel": channel,
        "sender": sender,
        "messages": messages,
        "reply_metadata": reply_metadata,
    }


def message_ids(response_json: dict) -> list[str]:
    return [item["kick_message_id"] for item in response_json["items"]]


async def test_public_search_returns_latest_messages_without_auth(client, session_factory) -> None:
    dataset = await seed_search_dataset(session_factory)

    response = await client.get("/messages", params={"limit": 4})

    assert response.status_code == 200
    assert message_ids(response.json())[:4] == [
        message.kick_message_id for message in dataset["messages"]
    ]
    first_item = response.json()["items"][0]
    assert first_item["sender"]["profile_image_url"] == "https://example.com/yavuz.png"
    assert first_item["channel"]["profile_image_url"] == "https://example.com/channel-one.png"
    assert first_item["emotes"][0]["image_url"] == "https://files.kick.com/emotes/37226/fullsize"


async def test_public_search_returns_reply_context_fields(client, session_factory) -> None:
    suffix = uuid4().hex[:8]
    reply_metadata = {
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

    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        channel = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=100000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=200000 + int(uuid4().hex[:6], 16),
                slug=f"reply-channel-{suffix}",
                display_name=f"Reply Channel {suffix}",
            )
        )
        sender = await unit_of_work.senders.add(
            Sender(
                kick_user_id=500000 + int(uuid4().hex[:6], 16),
                username=f"ReplySender{suffix}",
                slug=f"reply-sender-{suffix}",
            )
        )
        message = await unit_of_work.messages.add(
            ChatMessage(
                kick_message_id=unique_value("message"),
                channel_id=channel.id or 0,
                sender_id=sender.id or 0,
                chatroom_id=channel.kick_chatroom_id or 0,
                content=f"current reply content combo-{suffix}",
                message_type="reply",
                sender_username_snapshot=sender.username,
                sender_slug_snapshot=sender.slug,
                reply_metadata=reply_metadata,
                thread_parent_id="cad8a796-d688-4de1-9e13-2e0a4d0b5f1f",
                message_created_at=datetime(2035, 1, 1, 13, 0, tzinfo=UTC),
            )
        )
        await unit_of_work.commit()

    response = await client.get("/messages", params={"q": f"combo-{suffix}"})

    assert response.status_code == 200
    item = response.json()["items"][0]
    assert item["kick_message_id"] == message.kick_message_id
    assert item["message_type"] == "reply"
    assert item["reply_metadata"] == reply_metadata
    assert item["thread_parent_id"] == "cad8a796-d688-4de1-9e13-2e0a4d0b5f1f"


async def test_public_search_combines_optional_filters(client, session_factory) -> None:
    dataset = await seed_search_dataset(session_factory)
    suffix = dataset["suffix"]
    messages = dataset["messages"]

    sender_response = await client.get(
        "/messages",
        params={"sender": f"yavuz-{suffix}", "start": dataset["base_time"].isoformat()},
    )
    sender_and_content_response = await client.get(
        "/messages",
        params={"sender": f"yavuz-{suffix}", "q": f"selam combo-{suffix}"},
    )
    channel_and_content_response = await client.get(
        "/messages",
        params={"channel": f"example-{suffix}", "q": f"hello combo-{suffix}"},
    )
    content_response = await client.get(
        "/messages",
        params={"q": f"hello combo-{suffix}"},
    )
    partial_sender_response = await client.get(
        "/messages",
        params={"sender": "yavuz", "q": f"combo-{suffix}"},
    )
    all_filters_response = await client.get(
        "/messages",
        params={
            "sender": f"other-user-{suffix}",
            "channel": f"other-{suffix}",
            "q": f"hello combo-{suffix} older",
        },
    )

    assert message_ids(sender_response.json()) == [
        messages[0].kick_message_id,
        messages[1].kick_message_id,
    ]
    assert message_ids(sender_and_content_response.json()) == [messages[0].kick_message_id]
    assert message_ids(channel_and_content_response.json()) == [messages[2].kick_message_id]
    assert message_ids(content_response.json()) == [
        messages[2].kick_message_id,
        messages[3].kick_message_id,
    ]
    assert message_ids(partial_sender_response.json()) == []
    assert message_ids(all_filters_response.json()) == [messages[3].kick_message_id]


async def test_public_search_sender_filter_accepts_kick_profile_slug_for_underscores(
    client,
    session_factory,
) -> None:
    suffix = uuid4().hex[:8]
    legacy_slug = f"search_user_{suffix}"
    profile_slug = legacy_slug.replace("_", "-")

    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        channel = await unit_of_work.channels.add(
            Channel(
                kick_channel_id=700000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=800000 + int(uuid4().hex[:6], 16),
                slug=f"search-underscore-{suffix}",
                display_name=f"Search Underscore {suffix}",
            )
        )
        sender = await unit_of_work.senders.add(
            Sender(
                kick_user_id=900000 + int(uuid4().hex[:6], 16),
                username=legacy_slug,
                slug=legacy_slug,
            )
        )
        message = await unit_of_work.messages.add(
            ChatMessage(
                kick_message_id=unique_value("search-underscore-message"),
                channel_id=channel.id or 0,
                sender_id=sender.id or 0,
                chatroom_id=channel.kick_chatroom_id or 0,
                content=f"search underscore combo-{suffix}",
                message_type="message",
                sender_username_snapshot=legacy_slug,
                sender_slug_snapshot=legacy_slug,
                message_created_at=datetime(2035, 1, 1, 12, 0, tzinfo=UTC),
            )
        )
        await unit_of_work.commit()

    response = await client.get(
        "/messages",
        params={"sender": profile_slug, "q": f"combo-{suffix}"},
    )

    assert response.status_code == 200
    assert message_ids(response.json()) == [message.kick_message_id]


async def test_public_search_filters_by_date_range(client, session_factory) -> None:
    dataset = await seed_search_dataset(session_factory)
    base_time = dataset["base_time"]

    response = await client.get(
        "/messages",
        params={
            "q": f"combo-{dataset['suffix']}",
            "start": (base_time + timedelta(minutes=1)).isoformat(),
            "end": (base_time + timedelta(minutes=2)).isoformat(),
        },
    )

    assert message_ids(response.json()) == [
        dataset["messages"][1].kick_message_id,
        dataset["messages"][2].kick_message_id,
    ]


async def test_public_search_filters_reply_only_emote_only_and_combined(
    client,
    session_factory,
) -> None:
    dataset = await seed_search_improvement_dataset(session_factory)
    suffix = dataset["suffix"]
    messages = dataset["messages"]

    reply_response = await client.get(
        "/messages",
        params={"q": f"combo-{suffix}", "reply_only": "true"},
    )
    emote_response = await client.get(
        "/messages",
        params={"q": f"combo-{suffix}", "emote_only": "true"},
    )
    combined_response = await client.get(
        "/messages",
        params={"q": f"combo-{suffix}", "reply_only": "true", "emote_only": "true"},
    )

    assert message_ids(reply_response.json()) == [messages[0].kick_message_id]
    assert message_ids(emote_response.json()) == [
        messages[0].kick_message_id,
        messages[1].kick_message_id,
    ]
    assert message_ids(combined_response.json()) == [messages[0].kick_message_id]


async def test_public_export_returns_filtered_json_without_auth(client, session_factory) -> None:
    dataset = await seed_search_improvement_dataset(session_factory)
    suffix = dataset["suffix"]

    response = await client.get(
        "/messages/export",
        params={
            "format": "json",
            "q": f"combo-{suffix}",
            "reply_only": "true",
            "emote_only": "true",
        },
    )

    assert response.status_code == 200
    body = response.json()
    assert body["count"] == 1
    assert body["max_rows"] == 1000
    assert body["truncated"] is False
    assert message_ids(body) == [dataset["messages"][0].kick_message_id]
    assert body["items"][0]["reply_metadata"] == dataset["reply_metadata"]


async def test_public_export_applies_row_limit(client, session_factory) -> None:
    dataset = await seed_search_improvement_dataset(session_factory)

    response = await client.get(
        "/messages/export",
        params={"format": "json", "q": f"combo-{dataset['suffix']}", "limit": 2},
    )

    assert response.status_code == 200
    body = response.json()
    assert body["count"] == 2
    assert body["max_rows"] == 2
    assert body["truncated"] is True
    assert message_ids(body) == [
        dataset["messages"][0].kick_message_id,
        dataset["messages"][1].kick_message_id,
    ]


async def test_public_export_returns_csv_shape(client, session_factory) -> None:
    dataset = await seed_search_improvement_dataset(session_factory)

    response = await client.get(
        "/messages/export",
        params={
            "format": "csv",
            "q": f"combo-{dataset['suffix']}",
            "reply_only": "true",
            "limit": 1,
        },
    )

    assert response.status_code == 200
    assert response.headers["content-type"].startswith("text/csv")
    rows = list(csv.DictReader(StringIO(response.text)))
    assert list(rows[0].keys()) == [
        "message_created_at",
        "kick_message_id",
        "channel_slug",
        "channel_display_name",
        "sender_username",
        "sender_slug",
        "message_type",
        "content",
        "emotes",
        "reply_to_sender",
        "reply_to_content",
        "thread_parent_id",
    ]
    assert rows[0]["kick_message_id"] == dataset["messages"][0].kick_message_id
    assert rows[0]["message_type"] == "reply"
    assert rows[0]["reply_to_sender"] == dataset["reply_metadata"]["original_sender"]["username"]
    assert rows[0]["reply_to_content"] == dataset["reply_metadata"]["original_message"]["content"]


async def test_public_search_uses_cursor_pagination(client, session_factory) -> None:
    dataset = await seed_search_dataset(session_factory)

    first_page = await client.get(
        "/messages",
        params={"q": f"combo-{dataset['suffix']}", "limit": 2},
    )
    first_page_json = first_page.json()
    second_page = await client.get(
        "/messages",
        params={
            "q": f"combo-{dataset['suffix']}",
            "limit": 2,
            "cursor": first_page_json["next_cursor"],
        },
    )

    assert message_ids(first_page_json) == [
        dataset["messages"][0].kick_message_id,
        dataset["messages"][1].kick_message_id,
    ]
    assert first_page_json["next_cursor"] is not None
    assert message_ids(second_page.json()) == [
        dataset["messages"][2].kick_message_id,
        dataset["messages"][3].kick_message_id,
    ]


async def test_public_search_rejects_invalid_cursor(client) -> None:
    response = await client.get("/messages", params={"cursor": "not-a-cursor"})

    assert response.status_code == 422
