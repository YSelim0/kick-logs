from sqlalchemy import and_, func, or_, select
from sqlalchemy.ext.asyncio import AsyncSession

from kick_logs.domain.entities.chat_message import ChatMessage
from kick_logs.domain.value_objects.pagination import CursorPagination
from kick_logs.domain.value_objects.search_filters import MessageSearchFilters
from kick_logs.domain.value_objects.sender_slug import build_sender_lookup_terms
from kick_logs.infrastructure.database.mappers import chat_message_to_domain, chat_message_to_model
from kick_logs.infrastructure.database.models import ChannelModel, ChatMessageModel, SenderModel


class SqlAlchemyMessageRepository:
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def add(self, message: ChatMessage) -> ChatMessage:
        model = chat_message_to_model(message)
        self._session.add(model)
        await self._session.flush()
        await self._session.refresh(model)
        return chat_message_to_domain(model)

    async def get_by_kick_message_id(self, kick_message_id: str) -> ChatMessage | None:
        result = await self._session.execute(
            select(ChatMessageModel).where(ChatMessageModel.kick_message_id == kick_message_id)
        )
        model = result.scalar_one_or_none()
        return chat_message_to_domain(model) if model else None

    async def search(
        self,
        filters: MessageSearchFilters,
        pagination: CursorPagination,
    ) -> list[ChatMessage]:
        statement = (
            select(ChatMessageModel)
            .join(ChannelModel, ChannelModel.id == ChatMessageModel.channel_id)
            .join(SenderModel, SenderModel.id == ChatMessageModel.sender_id)
        )

        if filters.sender:
            sender_terms = build_sender_lookup_terms(filters.sender)
            statement = statement.where(
                or_(
                    func.lower(SenderModel.username).in_(sender_terms),
                    func.lower(SenderModel.slug).in_(sender_terms),
                    func.lower(ChatMessageModel.sender_username_snapshot).in_(sender_terms),
                    func.lower(ChatMessageModel.sender_slug_snapshot).in_(sender_terms),
                )
            )

        if filters.channel:
            channel_query = f"%{filters.channel.lower()}%"
            statement = statement.where(
                or_(
                    func.lower(ChannelModel.slug).like(channel_query),
                    func.lower(ChannelModel.display_name).like(channel_query),
                )
            )

        if filters.q:
            statement = statement.where(
                func.lower(ChatMessageModel.content).like(f"%{filters.q.lower()}%")
            )

        if filters.start:
            statement = statement.where(ChatMessageModel.message_created_at >= filters.start)

        if filters.end:
            statement = statement.where(ChatMessageModel.message_created_at <= filters.end)

        if filters.reply_only:
            statement = statement.where(ChatMessageModel.message_type == "reply")

        if filters.emote_only:
            statement = statement.where(func.jsonb_array_length(ChatMessageModel.emotes) > 0)

        if pagination.cursor:
            statement = statement.where(
                or_(
                    ChatMessageModel.message_created_at < pagination.cursor.message_created_at,
                    and_(
                        ChatMessageModel.message_created_at == pagination.cursor.message_created_at,
                        ChatMessageModel.id < pagination.cursor.message_id,
                    ),
                )
            )

        statement = statement.order_by(
            ChatMessageModel.message_created_at.desc(),
            ChatMessageModel.id.desc(),
        ).limit(pagination.limit)

        result = await self._session.execute(statement)
        return [chat_message_to_domain(model) for model in result.scalars().all()]

    async def list_latest_by_channel_id(
        self,
        channel_id: int,
        pagination: CursorPagination,
    ) -> list[ChatMessage]:
        statement = select(ChatMessageModel).where(ChatMessageModel.channel_id == channel_id)

        if pagination.cursor:
            statement = statement.where(
                or_(
                    ChatMessageModel.message_created_at < pagination.cursor.message_created_at,
                    and_(
                        ChatMessageModel.message_created_at == pagination.cursor.message_created_at,
                        ChatMessageModel.id < pagination.cursor.message_id,
                    ),
                )
            )

        statement = statement.order_by(
            ChatMessageModel.message_created_at.desc(),
            ChatMessageModel.id.desc(),
        ).limit(pagination.limit)

        result = await self._session.execute(statement)
        return [chat_message_to_domain(model) for model in result.scalars().all()]
