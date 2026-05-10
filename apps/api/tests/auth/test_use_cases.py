from collections.abc import Iterator

import pytest

from kick_logs.application.exceptions import (
    AuthenticationFailedError,
    DuplicateUserEmailError,
    PermissionDeniedError,
    UserNotFoundError,
)
from kick_logs.application.use_cases.auth import GetCurrentUserUseCase, LoginUseCase
from kick_logs.application.use_cases.users import CreateAdminUserUseCase, ListAdminUsersUseCase
from kick_logs.core.config import Settings
from kick_logs.domain.entities.user import User
from kick_logs.domain.value_objects.roles import UserRole
from kick_logs.infrastructure.auth import JwtTokenService, PasslibPasswordHasher
from kick_logs.infrastructure.seed import seed_super_admin


class FakeUserRepository:
    def __init__(self) -> None:
        self.users: dict[int, User] = {}
        self.next_id = 1

    async def add(self, user: User) -> User:
        user.id = self.next_id
        self.next_id += 1
        self.users[user.id] = user
        return user

    async def update(self, user: User) -> User:
        if user.id is None:
            raise ValueError("User id is required.")
        self.users[user.id] = user
        return user

    async def get_by_id(self, user_id: int) -> User | None:
        return self.users.get(user_id)

    async def get_by_email(self, email: str) -> User | None:
        normalized_email = email.strip().lower()
        for user in self.users.values():
            if user.email == normalized_email:
                return user
        return None

    async def list_active(self) -> list[User]:
        return [user for user in self.users.values() if user.is_active]


class FakeUnitOfWork:
    def __init__(self, users: FakeUserRepository) -> None:
        self.users = users
        self.committed = False

    async def __aenter__(self) -> "FakeUnitOfWork":
        return self

    async def __aexit__(self, exc_type: object, exc: object, traceback: object) -> None:
        return None

    async def commit(self) -> None:
        self.committed = True

    async def rollback(self) -> None:
        return None


@pytest.fixture
def fake_users() -> FakeUserRepository:
    return FakeUserRepository()


@pytest.fixture
def uow_factory(fake_users: FakeUserRepository) -> Iterator:
    def factory() -> FakeUnitOfWork:
        return FakeUnitOfWork(fake_users)

    yield factory


@pytest.fixture
def hasher() -> PasslibPasswordHasher:
    return PasslibPasswordHasher()


async def test_seed_super_admin_creates_default_user(
    uow_factory,
    hasher: PasslibPasswordHasher,
) -> None:
    user = await seed_super_admin(
        uow_factory,
        hasher,
        Settings(
            default_super_admin_email="admin@kicklogs.local",
            default_super_admin_password="admin123",
        ),
    )

    assert user.email == "admin@kicklogs.local"
    assert user.role == UserRole.SUPER_ADMIN
    assert hasher.verify("admin123", user.password_hash)


async def test_seed_super_admin_is_idempotent(uow_factory, hasher: PasslibPasswordHasher) -> None:
    first_user = await seed_super_admin(uow_factory, hasher)
    second_user = await seed_super_admin(uow_factory, hasher)

    assert first_user.id == second_user.id


async def test_login_succeeds_for_active_user(uow_factory, fake_users, hasher) -> None:
    user = await fake_users.add(
        User.create_super_admin("admin@example.com", hasher.hash("admin123"))
    )
    use_case = LoginUseCase(
        uow_factory,
        hasher,
        JwtTokenService(Settings(jwt_secret_key="test-secret-with-at-least-32-bytes")),
    )

    session = await use_case.execute("ADMIN@example.com", "admin123")

    assert session.user.id == user.id
    assert session.access_token


async def test_login_fails_safely_for_invalid_password(uow_factory, fake_users, hasher) -> None:
    await fake_users.add(User.create_super_admin("admin@example.com", hasher.hash("admin123")))
    use_case = LoginUseCase(
        uow_factory,
        hasher,
        JwtTokenService(Settings(jwt_secret_key="test-secret-with-at-least-32-bytes")),
    )

    with pytest.raises(AuthenticationFailedError):
        await use_case.execute("admin@example.com", "wrong")


async def test_get_current_user_rejects_missing_user(uow_factory) -> None:
    use_case = GetCurrentUserUseCase(uow_factory)

    with pytest.raises(UserNotFoundError):
        await use_case.execute(999)


async def test_create_admin_user_requires_super_admin(uow_factory, hasher) -> None:
    current_user = User.create_admin("admin@example.com", "hash")
    use_case = CreateAdminUserUseCase(uow_factory, hasher)

    with pytest.raises(PermissionDeniedError):
        await use_case.execute(current_user, "new@example.com", "password")


async def test_create_admin_user_rejects_duplicate_email(uow_factory, fake_users, hasher) -> None:
    current_user = await fake_users.add(User.create_super_admin("root@example.com", "hash"))
    await fake_users.add(User.create_admin("new@example.com", "hash"))
    use_case = CreateAdminUserUseCase(uow_factory, hasher)

    with pytest.raises(DuplicateUserEmailError):
        await use_case.execute(current_user, "new@example.com", "password")


async def test_create_and_list_admin_users(uow_factory, fake_users, hasher) -> None:
    current_user = await fake_users.add(User.create_super_admin("root@example.com", "hash"))
    create_use_case = CreateAdminUserUseCase(uow_factory, hasher)
    list_use_case = ListAdminUsersUseCase(uow_factory)

    created_user = await create_use_case.execute(current_user, "new@example.com", "password")
    users = await list_use_case.execute()

    assert created_user.email == "new@example.com"
    assert created_user.role == UserRole.ADMIN
    assert {user.email for user in users} == {"root@example.com", "new@example.com"}
