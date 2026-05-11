from collections.abc import Callable
from types import TracebackType

from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker

from kick_logs.infrastructure.database.repositories import (
    SqlAlchemyChannelRepository,
    SqlAlchemyMessageRepository,
    SqlAlchemyRawEventRepository,
    SqlAlchemySenderRepository,
    SqlAlchemyUserRepository,
)
from kick_logs.infrastructure.database.session import create_session_factory

SessionFactory = Callable[[], AsyncSession] | async_sessionmaker[AsyncSession]


class SqlAlchemyUnitOfWork:
    def __init__(
        self,
        session_factory: SessionFactory | None = None,
    ) -> None:
        self._session_factory = session_factory or create_session_factory()
        self.session: AsyncSession | None = None
        self.users: SqlAlchemyUserRepository
        self.channels: SqlAlchemyChannelRepository
        self.senders: SqlAlchemySenderRepository
        self.messages: SqlAlchemyMessageRepository
        self.raw_events: SqlAlchemyRawEventRepository

    async def __aenter__(self) -> "SqlAlchemyUnitOfWork":
        self.session = self._session_factory()
        self.users = SqlAlchemyUserRepository(self.session)
        self.channels = SqlAlchemyChannelRepository(self.session)
        self.senders = SqlAlchemySenderRepository(self.session)
        self.messages = SqlAlchemyMessageRepository(self.session)
        self.raw_events = SqlAlchemyRawEventRepository(self.session)
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        traceback: TracebackType | None,
    ) -> None:
        if self.session is None:
            return

        if exc_type is not None:
            await self.rollback()

        await self.session.close()

    async def commit(self) -> None:
        if self.session is None:
            raise RuntimeError("Unit of work has not been entered.")
        await self.session.commit()

    async def rollback(self) -> None:
        if self.session is None:
            raise RuntimeError("Unit of work has not been entered.")
        await self.session.rollback()
