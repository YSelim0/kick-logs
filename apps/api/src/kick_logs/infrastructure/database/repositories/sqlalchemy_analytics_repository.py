from sqlalchemy import func, or_, select, true
from sqlalchemy.ext.asyncio import AsyncSession

from kick_logs.application.dto.analytics import (
    AnalyticsOverviewDTO,
    MessageVolumePointDTO,
    TopChannelDTO,
    TopEmoteDTO,
    TopSenderDTO,
)
from kick_logs.domain.value_objects.analytics_filters import AnalyticsBucket, AnalyticsFilters
from kick_logs.infrastructure.database.models import ChannelModel, ChatMessageModel, SenderModel


class SqlAlchemyAnalyticsRepository:
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def get_overview(self, filters: AnalyticsFilters) -> AnalyticsOverviewDTO:
        statement = self._apply_filters(
            select(
                func.count(ChatMessageModel.id).label("total_messages"),
                func.count(func.distinct(ChatMessageModel.sender_id)).label("total_senders"),
                func.count(func.distinct(ChatMessageModel.channel_id)).label("total_channels"),
                func.coalesce(func.sum(func.jsonb_array_length(ChatMessageModel.emotes)), 0).label(
                    "total_emote_usages"
                ),
                func.min(ChatMessageModel.message_created_at).label("first_message_at"),
                func.max(ChatMessageModel.message_created_at).label("latest_message_at"),
            )
            .select_from(ChatMessageModel)
            .join(ChannelModel, ChannelModel.id == ChatMessageModel.channel_id)
            .join(SenderModel, SenderModel.id == ChatMessageModel.sender_id),
            filters,
        )
        result = await self._session.execute(statement)
        row = result.one()
        return AnalyticsOverviewDTO(
            total_messages=int(row.total_messages or 0),
            total_senders=int(row.total_senders or 0),
            total_channels=int(row.total_channels or 0),
            total_emote_usages=int(row.total_emote_usages or 0),
            first_message_at=row.first_message_at,
            latest_message_at=row.latest_message_at,
        )

    async def get_message_volume(
        self,
        filters: AnalyticsFilters,
        bucket: AnalyticsBucket,
    ) -> list[MessageVolumePointDTO]:
        bucket_start = func.date_trunc(bucket, ChatMessageModel.message_created_at).label(
            "bucket_start"
        )
        statement = self._apply_filters(
            select(
                bucket_start,
                func.count(ChatMessageModel.id).label("message_count"),
            )
            .select_from(ChatMessageModel)
            .join(ChannelModel, ChannelModel.id == ChatMessageModel.channel_id)
            .join(SenderModel, SenderModel.id == ChatMessageModel.sender_id),
            filters,
        )
        statement = statement.group_by(bucket_start).order_by(bucket_start.asc())
        result = await self._session.execute(statement)
        return [
            MessageVolumePointDTO(
                bucket_start=row.bucket_start,
                message_count=int(row.message_count or 0),
            )
            for row in result.all()
        ]

    async def get_top_senders(
        self,
        filters: AnalyticsFilters,
        limit: int,
    ) -> list[TopSenderDTO]:
        message_count = func.count(ChatMessageModel.id).label("message_count")
        latest_message_at = func.max(ChatMessageModel.message_created_at).label("latest_message_at")
        first_message_at = func.min(ChatMessageModel.message_created_at).label("first_message_at")
        statement = self._apply_filters(
            select(
                SenderModel.id.label("sender_id"),
                SenderModel.kick_user_id,
                SenderModel.username,
                SenderModel.slug,
                SenderModel.profile_image_url,
                message_count,
                first_message_at,
                latest_message_at,
            )
            .select_from(ChatMessageModel)
            .join(ChannelModel, ChannelModel.id == ChatMessageModel.channel_id)
            .join(SenderModel, SenderModel.id == ChatMessageModel.sender_id),
            filters,
        )
        statement = (
            statement.group_by(
                SenderModel.id,
                SenderModel.kick_user_id,
                SenderModel.username,
                SenderModel.slug,
                SenderModel.profile_image_url,
            )
            .order_by(message_count.desc(), latest_message_at.desc(), func.lower(SenderModel.slug))
            .limit(self._normalize_limit(limit))
        )
        result = await self._session.execute(statement)
        return [
            TopSenderDTO(
                sender_id=row.sender_id,
                kick_user_id=row.kick_user_id,
                username=row.username,
                slug=row.slug,
                profile_image_url=row.profile_image_url,
                message_count=int(row.message_count),
                first_message_at=row.first_message_at,
                latest_message_at=row.latest_message_at,
            )
            for row in result.all()
        ]

    async def get_top_channels(
        self,
        filters: AnalyticsFilters,
        limit: int,
    ) -> list[TopChannelDTO]:
        message_count = func.count(ChatMessageModel.id).label("message_count")
        latest_message_at = func.max(ChatMessageModel.message_created_at).label("latest_message_at")
        first_message_at = func.min(ChatMessageModel.message_created_at).label("first_message_at")
        statement = self._apply_filters(
            select(
                ChannelModel.id.label("channel_id"),
                ChannelModel.slug,
                ChannelModel.display_name,
                ChannelModel.profile_image_url,
                ChannelModel.banner_image_url,
                message_count,
                first_message_at,
                latest_message_at,
            )
            .select_from(ChatMessageModel)
            .join(ChannelModel, ChannelModel.id == ChatMessageModel.channel_id)
            .join(SenderModel, SenderModel.id == ChatMessageModel.sender_id),
            filters,
        )
        statement = (
            statement.group_by(
                ChannelModel.id,
                ChannelModel.slug,
                ChannelModel.display_name,
                ChannelModel.profile_image_url,
                ChannelModel.banner_image_url,
            )
            .order_by(message_count.desc(), latest_message_at.desc(), func.lower(ChannelModel.slug))
            .limit(self._normalize_limit(limit))
        )
        result = await self._session.execute(statement)
        return [
            TopChannelDTO(
                channel_id=row.channel_id,
                slug=row.slug,
                display_name=row.display_name,
                profile_image_url=row.profile_image_url,
                banner_image_url=row.banner_image_url,
                message_count=int(row.message_count),
                first_message_at=row.first_message_at,
                latest_message_at=row.latest_message_at,
            )
            for row in result.all()
        ]

    async def get_top_emotes(
        self,
        filters: AnalyticsFilters,
        limit: int,
    ) -> list[TopEmoteDTO]:
        emote = (
            func.jsonb_array_elements(ChatMessageModel.emotes).table_valued("value").alias("emote")
        )
        emote_id = emote.c.value.op("->>")("id")
        emote_name = emote.c.value.op("->>")("name")
        emote_token = emote.c.value.op("->>")("token")
        emote_image_url = emote.c.value.op("->>")("image_url")
        usage_count = func.count().label("usage_count")
        message_count = func.count(func.distinct(ChatMessageModel.id)).label("message_count")
        statement = self._apply_filters(
            select(
                emote_id.label("id"),
                emote_name.label("name"),
                emote_token.label("token"),
                emote_image_url.label("image_url"),
                usage_count,
                message_count,
            )
            .select_from(ChatMessageModel)
            .join(ChannelModel, ChannelModel.id == ChatMessageModel.channel_id)
            .join(SenderModel, SenderModel.id == ChatMessageModel.sender_id)
            .join(emote, true()),
            filters,
        )
        statement = (
            statement.group_by(emote_id, emote_name, emote_token, emote_image_url)
            .order_by(usage_count.desc(), message_count.desc(), emote_name.asc())
            .limit(self._normalize_limit(limit))
        )
        result = await self._session.execute(statement)
        return [
            TopEmoteDTO(
                id=row.id,
                name=row.name,
                token=row.token,
                image_url=row.image_url,
                usage_count=int(row.usage_count),
                message_count=int(row.message_count),
            )
            for row in result.all()
        ]

    def _apply_filters(self, statement, filters: AnalyticsFilters):
        if filters.start:
            statement = statement.where(ChatMessageModel.message_created_at >= filters.start)

        if filters.end:
            statement = statement.where(ChatMessageModel.message_created_at <= filters.end)

        if filters.channel:
            channel_query = filters.channel.lower()
            statement = statement.where(
                or_(
                    func.lower(ChannelModel.slug) == channel_query,
                    func.lower(ChannelModel.display_name) == channel_query,
                )
            )

        if filters.sender:
            sender_query = filters.sender.lower()
            statement = statement.where(
                or_(
                    func.lower(SenderModel.username) == sender_query,
                    func.lower(SenderModel.slug) == sender_query,
                    func.lower(ChatMessageModel.sender_username_snapshot) == sender_query,
                    func.lower(ChatMessageModel.sender_slug_snapshot) == sender_query,
                )
            )

        return statement

    def _normalize_limit(self, limit: int) -> int:
        return min(max(limit, 1), 100)
