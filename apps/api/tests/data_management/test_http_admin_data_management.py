from collections.abc import AsyncIterator
from datetime import UTC, datetime, timedelta
from uuid import uuid4

import pytest
from httpx import ASGITransport, AsyncClient
from sqlalchemy import text
from sqlalchemy.ext.asyncio import async_sessionmaker, create_async_engine

from kick_logs.core.config import Settings, get_settings
from kick_logs.domain.entities import Channel, ChatMessage, RawKickEvent, Sender, User
from kick_logs.domain.value_objects.raw_event_status import RawEventStatus
from kick_logs.infrastructure.auth import PasslibPasswordHasher
from kick_logs.infrastructure.database import SqlAlchemyUnitOfWork
from kick_logs.infrastructure.seed import seed_super_admin
from kick_logs.presentation.http.app import create_app
from kick_logs.presentation.http.dependencies import get_unit_of_work_factory


@pytest.fixture
async def session_factory() -> AsyncIterator[async_sessionmaker]:
    engine = create_async_engine(get_settings().database_url, pool_pre_ping=True)

    try:
        async with engine.connect() as healthcheck:
            table_exists = await healthcheck.scalar(
                text("select to_regclass('public.data_retention_settings')")
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


async def seed_test_admin(session_factory, email: str, password: str = "admin123") -> None:
    hasher = PasslibPasswordHasher()
    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        await unit_of_work.users.add(
            User.create_admin(email=email, password_hash=hasher.hash(password))
        )
        await unit_of_work.commit()


async def seed_test_super_admin(
    session_factory,
    email: str,
    password: str = "admin123",
) -> None:
    await seed_super_admin(
        lambda: SqlAlchemyUnitOfWork(session_factory),
        PasslibPasswordHasher(),
        Settings(default_super_admin_email=email, default_super_admin_password=password),
    )


async def login(client: AsyncClient, email: str, password: str = "admin123") -> None:
    response = await client.post(
        "/auth/login",
        json={"email": email, "password": password},
    )
    assert response.status_code == 200


async def seed_cleanup_rows(
    session_factory,
    *,
    channel_slug: str | None = None,
    sender_slug: str | None = None,
) -> tuple[str, str, str, str]:
    now = datetime.now(UTC)
    channel_slug = channel_slug or unique_value("cleanup-channel")
    sender_slug = sender_slug or unique_value("cleanup-sender")
    old_marker = unique_value("old-cleanup-message")
    new_marker = unique_value("new-cleanup-message")

    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        channel = await unit_of_work.channels.add(
            Channel(
                slug=channel_slug,
                display_name=channel_slug,
                kick_channel_id=100000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=200000 + int(uuid4().hex[:6], 16),
            )
        )
        sender = await unit_of_work.senders.add(
            Sender(
                kick_user_id=300000 + int(uuid4().hex[:6], 16),
                username=sender_slug,
                slug=sender_slug,
            )
        )
        await unit_of_work.messages.add(
            ChatMessage(
                kick_message_id=unique_value("old-message"),
                channel_id=channel.id or 0,
                sender_id=sender.id or 0,
                chatroom_id=channel.kick_chatroom_id or 1,
                content=old_marker,
                message_type="message",
                sender_username_snapshot=sender.username,
                sender_slug_snapshot=sender.slug,
                message_created_at=now - timedelta(days=40),
            )
        )
        await unit_of_work.messages.add(
            ChatMessage(
                kick_message_id=unique_value("new-message"),
                channel_id=channel.id or 0,
                sender_id=sender.id or 0,
                chatroom_id=channel.kick_chatroom_id or 1,
                content=new_marker,
                message_type="message",
                sender_username_snapshot=sender.username,
                sender_slug_snapshot=sender.slug,
                message_created_at=now - timedelta(days=5),
            )
        )
        await unit_of_work.raw_events.add(
            RawKickEvent(
                event_name=r"App\Events\ChatMessageEvent",
                kick_message_id=unique_value("old-raw"),
                chatroom_id=channel.kick_chatroom_id,
                kick_channel_id=channel.kick_channel_id,
                channel_id=channel.id,
                payload={
                    "id": unique_value("old-raw-payload"),
                    "sender": {"username": sender.username, "slug": sender.slug},
                },
                status=RawEventStatus.PROCESSED,
                received_at=now - timedelta(days=40),
                processed_at=now - timedelta(days=39),
            )
        )
        await unit_of_work.raw_events.add(
            RawKickEvent(
                event_name=r"App\Events\ChatMessageEvent",
                kick_message_id=unique_value("new-raw"),
                chatroom_id=channel.kick_chatroom_id,
                kick_channel_id=channel.kick_channel_id,
                channel_id=channel.id,
                payload={
                    "id": unique_value("new-raw-payload"),
                    "sender": {"username": sender.username, "slug": sender.slug},
                },
                status=RawEventStatus.PENDING,
                received_at=now - timedelta(days=5),
            )
        )
        await unit_of_work.commit()

    return channel_slug, sender_slug, old_marker, new_marker


async def test_data_management_summary_rejects_unauthenticated_requests(client) -> None:
    response = await client.get("/admin/data-management/summary")

    assert response.status_code == 401


async def test_data_management_summary_defaults_to_keep_forever_and_updates_settings(
    client,
    session_factory,
) -> None:
    email = f"{unique_value('admin')}@example.com"
    await seed_test_admin(session_factory, email)
    await login(client, email)

    summary_response = await client.get("/admin/data-management/summary")
    summary = summary_response.json()

    assert summary_response.status_code == 200
    assert summary["retention_settings"]["message_retention_days"] is None
    assert summary["retention_settings"]["raw_event_retention_days"] is None
    assert summary["database_bytes"] > 0
    assert {table["table_name"] for table in summary["tables"]} == {
        "channels",
        "senders",
        "chat_messages",
        "raw_kick_events",
    }

    update_response = await client.put(
        "/admin/data-management/retention-settings",
        json={"message_retention_days": 30, "raw_event_retention_days": 90},
    )

    assert update_response.status_code == 200
    assert update_response.json()["message_retention_days"] == 30
    assert update_response.json()["raw_event_retention_days"] == 90

    invalid_response = await client.put(
        "/admin/data-management/retention-settings",
        json={"message_retention_days": 7, "raw_event_retention_days": None},
    )
    assert invalid_response.status_code == 422


async def test_cleanup_preview_counts_old_messages_and_requires_exact_confirmation(
    client,
    session_factory,
) -> None:
    email = f"{unique_value('root')}@example.com"
    await seed_test_super_admin(session_factory, email)
    await login(client, email)
    _channel_slug, _sender_slug, old_marker, new_marker = await seed_cleanup_rows(session_factory)
    await client.put(
        "/admin/data-management/retention-settings",
        json={"message_retention_days": 30, "raw_event_retention_days": None},
    )

    preview_response = await client.post(
        "/admin/data-management/cleanup/preview",
        json={"target": "old_messages"},
    )
    preview = preview_response.json()

    assert preview_response.status_code == 200
    assert preview["affected"]["messages"] >= 1
    assert preview["affected"]["raw_events"] == 0
    assert preview["confirmation_text"] == "DELETE OLD MESSAGES"
    assert preview["can_execute"] is True

    rejected_response = await client.post(
        "/admin/data-management/cleanup/confirm",
        json={"target": "old_messages", "confirmation_text": "DELETE"},
    )
    assert rejected_response.status_code == 400

    confirmed_response = await client.post(
        "/admin/data-management/cleanup/confirm",
        json={"target": "old_messages", "confirmation_text": "DELETE OLD MESSAGES"},
    )

    assert confirmed_response.status_code == 200
    assert confirmed_response.json()["deleted"]["messages"] >= 1
    assert (await client.get(f"/messages?q={old_marker}&limit=10")).json()["items"] == []
    assert len((await client.get(f"/messages?q={new_marker}&limit=10")).json()["items"]) == 1


async def test_cleanup_handles_old_raw_events_channel_and_sender_targets(
    client,
    session_factory,
) -> None:
    email = f"{unique_value('admin')}@example.com"
    await seed_test_admin(session_factory, email)
    await login(client, email)
    channel_slug, sender_slug, _old_marker, _new_marker = await seed_cleanup_rows(session_factory)
    await client.put(
        "/admin/data-management/retention-settings",
        json={"message_retention_days": None, "raw_event_retention_days": 30},
    )

    raw_preview = (
        await client.post(
            "/admin/data-management/cleanup/preview",
            json={"target": "old_raw_events"},
        )
    ).json()
    assert raw_preview["affected"]["raw_events"] >= 1
    assert raw_preview["confirmation_text"] == "DELETE OLD RAW EVENTS"

    channel_preview = (
        await client.post(
            "/admin/data-management/cleanup/preview",
            json={"target": "channel", "channel_slug": channel_slug},
        )
    ).json()
    assert channel_preview["affected"]["messages"] == 2
    assert channel_preview["affected"]["raw_events"] == 2
    assert channel_preview["confirmation_text"] == f"DELETE CHANNEL {channel_slug}"

    sender_preview = (
        await client.post(
            "/admin/data-management/cleanup/preview",
            json={"target": "sender", "sender": sender_slug},
        )
    ).json()
    assert sender_preview["affected"]["messages"] == 2
    assert sender_preview["affected"]["raw_events"] == 2
    assert sender_preview["confirmation_text"] == f"DELETE SENDER {sender_slug}"

    confirmed_response = await client.post(
        "/admin/data-management/cleanup/confirm",
        json={
            "target": "channel",
            "channel_slug": channel_slug,
            "confirmation_text": f"DELETE CHANNEL {channel_slug}",
        },
    )
    assert confirmed_response.status_code == 200
    assert confirmed_response.json()["deleted"] == {"messages": 2, "raw_events": 2, "total": 4}

    after_cleanup = (
        await client.post(
            "/admin/data-management/cleanup/preview",
            json={"target": "channel", "channel_slug": channel_slug},
        )
    ).json()
    assert after_cleanup["affected"] == {"messages": 0, "raw_events": 0, "total": 0}
