from collections.abc import AsyncIterator
from uuid import uuid4

import pytest
from httpx import ASGITransport, AsyncClient
from sqlalchemy import text
from sqlalchemy.ext.asyncio import async_sessionmaker, create_async_engine

from kick_logs.application.dto.channels import ResolvedKickChannelDTO
from kick_logs.application.exceptions import ChannelResolutionError
from kick_logs.core.config import Settings, get_settings
from kick_logs.domain.entities.user import User
from kick_logs.infrastructure.auth import PasslibPasswordHasher
from kick_logs.infrastructure.database import SqlAlchemyUnitOfWork
from kick_logs.infrastructure.seed import seed_super_admin
from kick_logs.presentation.http.app import create_app
from kick_logs.presentation.http.dependencies import get_channel_resolver, get_unit_of_work_factory


class FakeChannelResolver:
    async def resolve(self, slug: str) -> ResolvedKickChannelDTO:
        normalized_slug = slug.strip().lower()
        return ResolvedKickChannelDTO(
            kick_channel_id=990001,
            kick_chatroom_id=880001,
            slug=normalized_slug,
            display_name="Hype",
            profile_image_url="https://example.com/profile.png",
            banner_image_url="https://example.com/banner.png",
            raw_payload={
                "id": 990001,
                "slug": normalized_slug,
                "chatroom": {"id": 880001},
            },
        )


class FailingChannelResolver:
    async def resolve(self, _slug: str) -> ResolvedKickChannelDTO:
        raise ChannelResolutionError("Could not resolve channel.")


@pytest.fixture
async def session_factory() -> AsyncIterator[async_sessionmaker]:
    engine = create_async_engine(get_settings().database_url, pool_pre_ping=True)

    try:
        async with engine.connect() as healthcheck:
            table_exists = await healthcheck.scalar(
                text("select to_regclass('public.channels')")
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
async def app(session_factory):
    fastapi_app = create_app(seed_super_admin_on_startup=False)
    fastapi_app.dependency_overrides[get_unit_of_work_factory] = lambda: (
        lambda: SqlAlchemyUnitOfWork(session_factory)
    )
    fastapi_app.dependency_overrides[get_channel_resolver] = lambda: FakeChannelResolver()

    try:
        yield fastapi_app
    finally:
        fastapi_app.dependency_overrides.clear()


@pytest.fixture
async def client(app) -> AsyncIterator[AsyncClient]:
    async with AsyncClient(
        transport=ASGITransport(app=app),
        base_url="http://testserver",
    ) as async_client:
        yield async_client


def unique_email(prefix: str) -> str:
    return f"{prefix}-{uuid4().hex[:10]}@example.com"


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


async def test_admin_channel_routes_reject_unauthenticated_requests(client) -> None:
    get_response = await client.get("/admin/channels")
    post_response = await client.post("/admin/channels", json={"slug": "hype"})
    delete_response = await client.delete("/admin/channels/1")

    assert get_response.status_code == 401
    assert post_response.status_code == 401
    assert delete_response.status_code == 401


async def test_regular_admin_can_add_list_disable_and_re_enable_channel(
    client,
    session_factory,
) -> None:
    email = unique_email("admin")
    await seed_test_admin(session_factory, email)
    await login(client, email)

    created_response = await client.post("/admin/channels", json={"slug": " HYPE "})
    created = created_response.json()

    assert created_response.status_code == 201
    assert created["slug"] == "hype"
    assert created["display_name"] == "Hype"
    assert created["profile_image_url"] == "https://example.com/profile.png"
    assert created["is_enabled"] is True

    list_response = await client.get("/admin/channels")
    channels_by_slug = {channel["slug"]: channel for channel in list_response.json()}
    assert list_response.status_code == 200
    assert channels_by_slug["hype"]["id"] == created["id"]

    delete_response = await client.delete(f"/admin/channels/{created['id']}")
    assert delete_response.status_code == 200
    assert delete_response.json()["is_enabled"] is False

    reenabled_response = await client.post("/admin/channels", json={"slug": "hype"})
    assert reenabled_response.status_code == 201
    assert reenabled_response.json()["id"] == created["id"]
    assert reenabled_response.json()["is_enabled"] is True


async def test_super_admin_receives_422_when_channel_resolution_fails(
    app,
    client,
    session_factory,
) -> None:
    app.dependency_overrides[get_channel_resolver] = lambda: FailingChannelResolver()
    email = unique_email("root")
    await seed_test_super_admin(session_factory, email)
    await login(client, email)

    response = await client.post("/admin/channels", json={"slug": "missing"})

    assert response.status_code == 422
    assert response.json()["detail"] == "Kick channel could not be resolved."
