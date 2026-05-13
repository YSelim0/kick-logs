from collections.abc import AsyncIterator
from datetime import UTC, datetime, timedelta
from uuid import uuid4

import pytest
from httpx import ASGITransport, AsyncClient
from sqlalchemy import text
from sqlalchemy.ext.asyncio import async_sessionmaker, create_async_engine

from kick_logs.core.config import Settings, get_settings
from kick_logs.domain.entities import (
    Channel,
    ChatMessage,
    RawKickEvent,
    Sender,
    User,
    WorkerHeartbeat,
)
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
                text("select to_regclass('public.worker_heartbeats')")
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


async def seed_test_admin(session_factory, email: str, password: str = "admin123") -> None:
    hasher = PasslibPasswordHasher()
    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        await unit_of_work.users.add(
            User.create_admin(email=email, password_hash=hasher.hash(password))
        )
        await unit_of_work.commit()


async def login(client: AsyncClient, email: str, password: str = "admin123") -> None:
    response = await client.post(
        "/auth/login",
        json={"email": email, "password": password},
    )
    assert response.status_code == 200


async def seed_operations_rows(
    session_factory,
    *,
    heartbeat_seen_at: datetime | None = None,
) -> None:
    now = datetime.now(UTC)
    async with SqlAlchemyUnitOfWork(session_factory) as unit_of_work:
        enabled_channel = await unit_of_work.channels.add(
            Channel(
                slug=unique_value("hype"),
                display_name="Hype",
                kick_channel_id=100000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=200000 + int(uuid4().hex[:6], 16),
            )
        )
        await unit_of_work.channels.add(
            Channel(
                slug=unique_value("disabled"),
                display_name="Disabled",
                kick_channel_id=300000 + int(uuid4().hex[:6], 16),
                kick_chatroom_id=400000 + int(uuid4().hex[:6], 16),
                is_enabled=False,
            )
        )
        sender = await unit_of_work.senders.add(
            Sender(
                kick_user_id=500000 + int(uuid4().hex[:6], 16),
                username="Yavuz",
                slug=unique_value("yavuz"),
            )
        )
        await unit_of_work.messages.add(
            ChatMessage(
                kick_message_id=unique_value("message"),
                channel_id=enabled_channel.id or 0,
                sender_id=sender.id or 0,
                chatroom_id=enabled_channel.kick_chatroom_id or 1,
                content="hello ops",
                message_type="message",
                sender_username_snapshot=sender.username,
                sender_slug_snapshot=sender.slug,
                message_created_at=now,
            )
        )
        await unit_of_work.raw_events.add(
            RawKickEvent.pending(
                event_name=r"App\Events\ChatMessageEvent",
                kick_message_id=unique_value("pending-raw"),
                chatroom_id=enabled_channel.kick_chatroom_id,
                channel_id=enabled_channel.id,
                payload={"id": unique_value("pending-payload")},
            )
        )
        await unit_of_work.raw_events.add(
            RawKickEvent(
                event_name=r"App\Events\ChatMessageEvent",
                kick_message_id=unique_value("processed-raw"),
                chatroom_id=enabled_channel.kick_chatroom_id,
                channel_id=enabled_channel.id,
                payload={"id": unique_value("processed-payload")},
                status=RawEventStatus.PROCESSED,
                received_at=now - timedelta(minutes=5),
                processed_at=now - timedelta(minutes=1),
            )
        )
        await unit_of_work.raw_events.add(
            RawKickEvent(
                event_name=r"App\Events\ChatMessageEvent",
                kick_message_id=unique_value("failed-raw"),
                chatroom_id=enabled_channel.kick_chatroom_id,
                channel_id=enabled_channel.id,
                payload={"id": unique_value("failed-payload")},
                status=RawEventStatus.FAILED,
                received_at=now - timedelta(minutes=4),
                last_error="boom",
            )
        )
        if heartbeat_seen_at is not None:
            await unit_of_work.worker_heartbeats.upsert(
                WorkerHeartbeat(
                    service_name="listener",
                    last_seen_at=heartbeat_seen_at,
                    metadata={"test": True},
                )
            )
        await unit_of_work.commit()


async def test_admin_operations_summary_rejects_unauthenticated_requests(client) -> None:
    response = await client.get("/admin/operations/summary")

    assert response.status_code == 401


async def test_admin_operations_summary_returns_counts_storage_and_fresh_listener(
    client,
    session_factory,
) -> None:
    email = f"{unique_value('root')}@example.com"
    await seed_test_super_admin(session_factory, email)
    await login(client, email)

    baseline = (await client.get("/admin/operations/summary")).json()
    await seed_operations_rows(session_factory, heartbeat_seen_at=datetime.now(UTC))

    response = await client.get("/admin/operations/summary")
    payload = response.json()

    assert response.status_code == 200
    assert payload["counts"]["channels"] == baseline["counts"]["channels"] + 2
    assert payload["counts"]["enabled_channels"] == baseline["counts"]["enabled_channels"] + 1
    assert payload["counts"]["senders"] == baseline["counts"]["senders"] + 1
    assert payload["counts"]["messages"] == baseline["counts"]["messages"] + 1
    assert payload["counts"]["raw_events"] == baseline["counts"]["raw_events"] + 3
    assert (
        payload["raw_event_status_counts"]["pending"]
        == baseline["raw_event_status_counts"]["pending"] + 1
    )
    assert (
        payload["raw_event_status_counts"]["processed"]
        == baseline["raw_event_status_counts"]["processed"] + 1
    )
    assert (
        payload["raw_event_status_counts"]["failed"]
        == baseline["raw_event_status_counts"]["failed"] + 1
    )
    assert payload["storage"]["database_bytes"] > 0
    assert {table["table_name"] for table in payload["storage"]["tables"]} == {
        "chat_messages",
        "raw_kick_events",
    }
    assert all(table["total_bytes"] >= 0 for table in payload["storage"]["tables"])
    assert payload["timestamps"]["latest_message_at"] is not None
    assert payload["timestamps"]["latest_raw_event_received_at"] is not None
    assert payload["timestamps"]["latest_raw_event_processed_at"] is not None
    assert payload["timestamps"]["oldest_pending_raw_event_received_at"] is not None
    assert payload["listener"]["service_name"] == "listener"
    assert payload["listener"]["is_fresh"] is True
    assert payload["listener"]["seconds_since_last_seen"] is not None


async def test_admin_operations_summary_marks_listener_stale(
    client,
    session_factory,
) -> None:
    email = f"{unique_value('admin')}@example.com"
    await seed_test_admin(session_factory, email)
    await seed_operations_rows(
        session_factory,
        heartbeat_seen_at=datetime.now(UTC) - timedelta(minutes=10),
    )
    await login(client, email)

    response = await client.get("/admin/operations/summary")

    assert response.status_code == 200
    assert response.json()["listener"]["is_fresh"] is False
