from collections.abc import AsyncIterator
from uuid import uuid4

import pytest
from httpx import ASGITransport, AsyncClient
from sqlalchemy import text
from sqlalchemy.ext.asyncio import async_sessionmaker, create_async_engine

from kick_logs.core.config import Settings, get_settings
from kick_logs.domain.entities.user import User
from kick_logs.domain.value_objects.roles import UserRole
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
            table_exists = await healthcheck.scalar(text("select to_regclass('public.users')"))
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


async def test_login_succeeds_for_seeded_super_admin(client, session_factory) -> None:
    email = unique_email("root")
    await seed_test_super_admin(session_factory, email)

    response = await client.post(
        "/auth/login",
        json={"email": email, "password": "admin123"},
    )

    assert response.status_code == 200
    assert response.json()["user"]["email"] == email
    assert get_settings().jwt_cookie_name in response.cookies


async def test_invalid_login_fails_safely(client, session_factory) -> None:
    email = unique_email("root")
    await seed_test_super_admin(session_factory, email)

    response = await client.post(
        "/auth/login",
        json={"email": email, "password": "wrong"},
    )

    assert response.status_code == 401
    assert "password" not in response.text.lower()


async def test_auth_me_works_with_cookie(client, session_factory) -> None:
    email = unique_email("root")
    await seed_test_super_admin(session_factory, email)
    login_response = await client.post(
        "/auth/login",
        json={"email": email, "password": "admin123"},
    )

    assert login_response.status_code == 200
    response = await client.get("/auth/me")

    assert response.status_code == 200
    assert response.json()["email"] == email


async def test_admin_routes_reject_unauthenticated_requests(client) -> None:
    response = await client.get("/admin/users")

    assert response.status_code == 401


async def test_admin_user_creation_requires_super_admin(client, session_factory) -> None:
    email = unique_email("admin")
    await seed_test_admin(session_factory, email)
    response = await client.post(
        "/auth/login",
        json={"email": email, "password": "admin123"},
    )
    assert response.status_code == 200

    response = await client.post(
        "/admin/users",
        json={"email": unique_email("new"), "password": "password123"},
    )

    assert response.status_code == 403


async def test_super_admin_can_create_and_list_admin_users(client, session_factory) -> None:
    email = unique_email("root")
    await seed_test_super_admin(session_factory, email)
    login_response = await client.post(
        "/auth/login",
        json={"email": email, "password": "admin123"},
    )
    assert login_response.status_code == 200

    created_response = await client.post(
        "/admin/users",
        json={"email": unique_email("new"), "password": "password123"},
    )
    list_response = await client.get("/admin/users")

    assert created_response.status_code == 201
    assert created_response.json()["role"] == UserRole.ADMIN
    assert list_response.status_code == 200
    assert len(list_response.json()) >= 2


async def test_health_route_remains_public(client) -> None:
    response = await client.get("/health")

    assert response.status_code == 200
    assert response.json() == {"status": "ok"}
