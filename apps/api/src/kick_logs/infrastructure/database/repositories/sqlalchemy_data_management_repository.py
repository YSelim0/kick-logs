from sqlalchemy import func, select, text
from sqlalchemy.ext.asyncio import AsyncSession

from kick_logs.application.dto.data_management import (
    DataCleanupCountsDTO,
    DataCleanupCriteriaDTO,
    DataManagementCountsDTO,
    DataManagementTableDTO,
    RetentionSettingsDTO,
)
from kick_logs.infrastructure.database.models import (
    ChannelModel,
    ChatMessageModel,
    DataRetentionSettingsModel,
    RawKickEventModel,
    SenderModel,
)

RETENTION_SETTINGS_ID = 1
DATA_TABLES = ("channels", "senders", "chat_messages", "raw_kick_events")


class SqlAlchemyDataManagementRepository:
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def get_retention_settings(self) -> RetentionSettingsDTO:
        model = await self._ensure_settings_model()
        return self._settings_to_dto(model)

    async def update_retention_settings(
        self,
        *,
        message_retention_days: int | None,
        raw_event_retention_days: int | None,
    ) -> RetentionSettingsDTO:
        model = await self._ensure_settings_model()
        model.message_retention_days = message_retention_days
        model.raw_event_retention_days = raw_event_retention_days
        await self._session.flush()
        await self._session.refresh(model)
        return self._settings_to_dto(model)

    async def get_counts(self) -> DataManagementCountsDTO:
        return DataManagementCountsDTO(
            channels=await self._count(ChannelModel),
            senders=await self._count(SenderModel),
            messages=await self._count(ChatMessageModel),
            raw_events=await self._count(RawKickEventModel),
        )

    async def get_database_size(self) -> int:
        return await self._scalar_int(text("select pg_database_size(current_database())"))

    async def get_table_sizes(self) -> list[DataManagementTableDTO]:
        return [
            DataManagementTableDTO(
                table_name=table_name,
                total_bytes=await self._table_size(table_name),
                row_count=await self._table_row_count(table_name),
            )
            for table_name in DATA_TABLES
        ]

    async def count_cleanup(self, criteria: DataCleanupCriteriaDTO) -> DataCleanupCountsDTO:
        if criteria.target == "old_messages":
            if criteria.cutoff_at is None:
                return DataCleanupCountsDTO()
            return DataCleanupCountsDTO(
                messages=await self._scalar_int(
                    text(
                        "select count(*) from chat_messages where message_created_at < :cutoff_at"
                    ),
                    {"cutoff_at": criteria.cutoff_at},
                )
            )

        if criteria.target == "old_raw_events":
            if criteria.cutoff_at is None:
                return DataCleanupCountsDTO()
            return DataCleanupCountsDTO(
                raw_events=await self._scalar_int(
                    text("select count(*) from raw_kick_events where received_at < :cutoff_at"),
                    {"cutoff_at": criteria.cutoff_at},
                )
            )

        if criteria.target == "channel":
            params = {"channel_slug": criteria.channel_slug}
            return DataCleanupCountsDTO(
                messages=await self._scalar_int(self._count_channel_messages_sql(), params),
                raw_events=await self._scalar_int(self._count_channel_raw_events_sql(), params),
            )

        params = {"sender": criteria.sender}
        return DataCleanupCountsDTO(
            messages=await self._scalar_int(self._count_sender_messages_sql(), params),
            raw_events=await self._scalar_int(self._count_sender_raw_events_sql(), params),
        )

    async def execute_cleanup(self, criteria: DataCleanupCriteriaDTO) -> DataCleanupCountsDTO:
        affected = await self.count_cleanup(criteria)

        if criteria.target == "old_messages":
            if criteria.cutoff_at is not None:
                await self._session.execute(
                    text("delete from chat_messages where message_created_at < :cutoff_at"),
                    {"cutoff_at": criteria.cutoff_at},
                )
            return affected

        if criteria.target == "old_raw_events":
            if criteria.cutoff_at is not None:
                await self._session.execute(
                    text("delete from raw_kick_events where received_at < :cutoff_at"),
                    {"cutoff_at": criteria.cutoff_at},
                )
            return affected

        if criteria.target == "channel":
            params = {"channel_slug": criteria.channel_slug}
            await self._session.execute(self._delete_channel_raw_events_sql(), params)
            await self._session.execute(self._delete_channel_messages_sql(), params)
            return affected

        params = {"sender": criteria.sender}
        await self._session.execute(self._delete_sender_raw_events_sql(), params)
        await self._session.execute(self._delete_sender_messages_sql(), params)
        return affected

    async def _ensure_settings_model(self) -> DataRetentionSettingsModel:
        model = await self._session.get(DataRetentionSettingsModel, RETENTION_SETTINGS_ID)
        if model is None:
            model = DataRetentionSettingsModel(
                id=RETENTION_SETTINGS_ID,
                message_retention_days=None,
                raw_event_retention_days=None,
            )
            self._session.add(model)
            await self._session.flush()
            await self._session.refresh(model)
        return model

    @staticmethod
    def _settings_to_dto(model: DataRetentionSettingsModel) -> RetentionSettingsDTO:
        return RetentionSettingsDTO(
            message_retention_days=model.message_retention_days,
            raw_event_retention_days=model.raw_event_retention_days,
            updated_at=model.updated_at,
        )

    async def _count(self, model: type) -> int:
        return await self._scalar_int(select(func.count()).select_from(model))

    async def _table_size(self, table_name: str) -> int:
        return await self._scalar_int(
            text("select coalesce(pg_total_relation_size(to_regclass(:table_name)), 0)"),
            {"table_name": table_name},
        )

    async def _table_row_count(self, table_name: str) -> int:
        return await self._scalar_int(text(f"select count(*) from {table_name}"))

    async def _scalar_int(self, statement, parameters: dict[str, object] | None = None) -> int:
        result = await self._session.execute(statement, parameters or {})
        return int(result.scalar_one() or 0)

    @staticmethod
    def _count_channel_messages_sql():
        return text(
            "select count(*) "
            "from chat_messages m "
            "join channels c on c.id = m.channel_id "
            "where lower(c.slug) = lower(:channel_slug)"
        )

    @staticmethod
    def _delete_channel_messages_sql():
        return text(
            "delete from chat_messages as m "
            "using channels as c "
            "where m.channel_id = c.id and lower(c.slug) = lower(:channel_slug)"
        )

    @staticmethod
    def _count_channel_raw_events_sql():
        return text(
            "select count(*) "
            "from raw_kick_events r "
            "where r.channel_id in ("
            "  select id from channels where lower(slug) = lower(:channel_slug)"
            ") "
            "or r.chatroom_id in ("
            "  select kick_chatroom_id from channels "
            "  where lower(slug) = lower(:channel_slug) and kick_chatroom_id is not null"
            ") "
            "or r.kick_channel_id in ("
            "  select kick_channel_id from channels "
            "  where lower(slug) = lower(:channel_slug) and kick_channel_id is not null"
            ")"
        )

    @staticmethod
    def _delete_channel_raw_events_sql():
        return text(
            "delete from raw_kick_events as r "
            "where r.channel_id in ("
            "  select id from channels where lower(slug) = lower(:channel_slug)"
            ") "
            "or r.chatroom_id in ("
            "  select kick_chatroom_id from channels "
            "  where lower(slug) = lower(:channel_slug) and kick_chatroom_id is not null"
            ") "
            "or r.kick_channel_id in ("
            "  select kick_channel_id from channels "
            "  where lower(slug) = lower(:channel_slug) and kick_channel_id is not null"
            ")"
        )

    @staticmethod
    def _count_sender_messages_sql():
        return text(
            "select count(*) "
            "from chat_messages m "
            "join senders s on s.id = m.sender_id "
            "where lower(s.slug) = lower(:sender) "
            "or lower(s.username) = lower(:sender) "
            "or lower(m.sender_slug_snapshot) = lower(:sender) "
            "or lower(m.sender_username_snapshot) = lower(:sender)"
        )

    @staticmethod
    def _delete_sender_messages_sql():
        return text(
            "delete from chat_messages as m "
            "using senders as s "
            "where m.sender_id = s.id "
            "and (lower(s.slug) = lower(:sender) "
            "or lower(s.username) = lower(:sender) "
            "or lower(m.sender_slug_snapshot) = lower(:sender) "
            "or lower(m.sender_username_snapshot) = lower(:sender))"
        )

    @staticmethod
    def _count_sender_raw_events_sql():
        return text(
            "select count(*) "
            "from raw_kick_events "
            "where lower(payload #>> '{sender,slug}') = lower(:sender) "
            "or lower(payload #>> '{sender,username}') = lower(:sender)"
        )

    @staticmethod
    def _delete_sender_raw_events_sql():
        return text(
            "delete from raw_kick_events "
            "where lower(payload #>> '{sender,slug}') = lower(:sender) "
            "or lower(payload #>> '{sender,username}') = lower(:sender)"
        )
