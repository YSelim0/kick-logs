from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from kick_logs.domain.entities.user import User
from kick_logs.infrastructure.database.mappers import user_to_domain, user_to_model
from kick_logs.infrastructure.database.models import UserModel


class SqlAlchemyUserRepository:
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def add(self, user: User) -> User:
        model = user_to_model(user)
        self._session.add(model)
        await self._session.flush()
        await self._session.refresh(model)
        return user_to_domain(model)

    async def update(self, user: User) -> User:
        if user.id is None:
            raise ValueError("Cannot update a user without id.")

        model = await self._session.get(UserModel, user.id)
        if model is None:
            raise ValueError("User not found.")

        model.email = user.email
        model.password_hash = user.password_hash
        model.role = user.role.value
        model.is_active = user.is_active
        await self._session.flush()
        await self._session.refresh(model)
        return user_to_domain(model)

    async def get_by_id(self, user_id: int) -> User | None:
        model = await self._session.get(UserModel, user_id)
        return user_to_domain(model) if model else None

    async def get_by_email(self, email: str) -> User | None:
        result = await self._session.execute(
            select(UserModel).where(UserModel.email == email.strip().lower())
        )
        model = result.scalar_one_or_none()
        return user_to_domain(model) if model else None

    async def list_active(self) -> list[User]:
        result = await self._session.execute(
            select(UserModel).where(UserModel.is_active.is_(True)).order_by(UserModel.email.asc())
        )
        return [user_to_domain(model) for model in result.scalars().all()]
