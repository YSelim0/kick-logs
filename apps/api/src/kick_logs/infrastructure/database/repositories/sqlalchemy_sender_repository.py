from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from kick_logs.domain.entities.sender import Sender
from kick_logs.infrastructure.database.mappers import sender_to_domain, sender_to_model
from kick_logs.infrastructure.database.models import SenderModel


class SqlAlchemySenderRepository:
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def add(self, sender: Sender) -> Sender:
        model = sender_to_model(sender)
        self._session.add(model)
        await self._session.flush()
        await self._session.refresh(model)
        return sender_to_domain(model)

    async def update(self, sender: Sender) -> Sender:
        if sender.id is None:
            raise ValueError("Cannot update a sender without id.")

        model = await self._session.get(SenderModel, sender.id)
        if model is None:
            raise ValueError("Sender not found.")

        model.kick_user_id = sender.kick_user_id
        model.username = sender.username
        model.slug = sender.slug
        model.profile_image_url = sender.profile_image_url
        model.last_seen_color = sender.last_seen_color
        model.raw_profile_payload = sender.raw_profile_payload
        await self._session.flush()
        await self._session.refresh(model)
        return sender_to_domain(model)

    async def get_by_id(self, sender_id: int) -> Sender | None:
        model = await self._session.get(SenderModel, sender_id)
        return sender_to_domain(model) if model else None

    async def get_by_kick_user_id(self, kick_user_id: int) -> Sender | None:
        result = await self._session.execute(
            select(SenderModel).where(SenderModel.kick_user_id == kick_user_id)
        )
        model = result.scalar_one_or_none()
        return sender_to_domain(model) if model else None

    async def get_by_slug(self, slug: str) -> Sender | None:
        result = await self._session.execute(
            select(SenderModel).where(SenderModel.slug == slug.strip().lower())
        )
        model = result.scalar_one_or_none()
        return sender_to_domain(model) if model else None

    async def list_by_ids(self, sender_ids: set[int]) -> list[Sender]:
        if not sender_ids:
            return []

        result = await self._session.execute(
            select(SenderModel).where(SenderModel.id.in_(sender_ids))
        )
        return [sender_to_domain(model) for model in result.scalars().all()]
