from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from kick_logs.domain.entities.channel import Channel
from kick_logs.infrastructure.database.mappers import channel_to_domain, channel_to_model
from kick_logs.infrastructure.database.models import ChannelModel


class SqlAlchemyChannelRepository:
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def add(self, channel: Channel) -> Channel:
        model = channel_to_model(channel)
        self._session.add(model)
        await self._session.flush()
        await self._session.refresh(model)
        return channel_to_domain(model)

    async def update(self, channel: Channel) -> Channel:
        if channel.id is None:
            raise ValueError("Cannot update a channel without id.")

        model = await self._session.get(ChannelModel, channel.id)
        if model is None:
            raise ValueError("Channel not found.")

        model.kick_channel_id = channel.kick_channel_id
        model.kick_chatroom_id = channel.kick_chatroom_id
        model.slug = channel.slug
        model.display_name = channel.display_name
        model.profile_image_url = channel.profile_image_url
        model.banner_image_url = channel.banner_image_url
        model.is_enabled = channel.is_enabled
        model.raw_payload = channel.raw_payload
        await self._session.flush()
        await self._session.refresh(model)
        return channel_to_domain(model)

    async def get_by_id(self, channel_id: int) -> Channel | None:
        model = await self._session.get(ChannelModel, channel_id)
        return channel_to_domain(model) if model else None

    async def get_by_slug(self, slug: str) -> Channel | None:
        result = await self._session.execute(
            select(ChannelModel).where(ChannelModel.slug == slug.strip().lower())
        )
        model = result.scalar_one_or_none()
        return channel_to_domain(model) if model else None

    async def list_enabled(self) -> list[Channel]:
        result = await self._session.execute(
            select(ChannelModel)
            .where(ChannelModel.is_enabled.is_(True))
            .order_by(ChannelModel.slug.asc())
        )
        return [channel_to_domain(model) for model in result.scalars().all()]

    async def list_all(self) -> list[Channel]:
        result = await self._session.execute(select(ChannelModel).order_by(ChannelModel.slug.asc()))
        return [channel_to_domain(model) for model in result.scalars().all()]
