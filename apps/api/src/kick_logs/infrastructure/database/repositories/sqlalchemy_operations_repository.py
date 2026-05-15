from sqlalchemy import func, select, text
from sqlalchemy.ext.asyncio import AsyncSession

from kick_logs.application.dto.operations import (
    OperationsCountsDTO,
    OperationsStorageDTO,
    OperationsStorageTableDTO,
    OperationsTimestampsDTO,
)
from kick_logs.domain.value_objects.raw_event_status import RawEventStatus
from kick_logs.infrastructure.database.models import (
    ChannelModel,
    ChatMessageModel,
    RawKickEventModel,
    SenderModel,
)


class SqlAlchemyOperationsRepository:
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def get_counts(self) -> OperationsCountsDTO:
        channels = await self._count(ChannelModel)
        enabled_channels = await self._scalar_int(
            select(func.count()).select_from(ChannelModel).where(ChannelModel.is_enabled.is_(True))
        )
        senders = await self._count(SenderModel)
        messages = await self._count(ChatMessageModel)
        raw_events = await self._count(RawKickEventModel)
        return OperationsCountsDTO(
            channels=channels,
            enabled_channels=enabled_channels,
            senders=senders,
            messages=messages,
            raw_events=raw_events,
        )

    async def get_raw_event_status_counts(self) -> dict[str, int]:
        result = await self._session.execute(
            select(RawKickEventModel.status, func.count()).group_by(RawKickEventModel.status)
        )
        counts = {status.value: 0 for status in RawEventStatus}
        for status_value, count in result.all():
            counts[str(status_value)] = int(count)
        return counts

    async def get_storage(self) -> OperationsStorageDTO:
        database_bytes = await self._scalar_int(text("select pg_database_size(current_database())"))
        tables = [
            OperationsStorageTableDTO(
                table_name=table_name,
                total_bytes=await self._table_size(table_name),
            )
            for table_name in ("chat_messages", "raw_kick_events")
        ]
        return OperationsStorageDTO(database_bytes=database_bytes, tables=tables)

    async def get_timestamps(self) -> OperationsTimestampsDTO:
        latest_message_at = await self._scalar_optional(
            select(func.max(ChatMessageModel.message_created_at))
        )
        latest_raw_event_received_at = await self._scalar_optional(
            select(func.max(RawKickEventModel.received_at))
        )
        latest_raw_event_processed_at = await self._scalar_optional(
            select(func.max(RawKickEventModel.processed_at))
        )
        oldest_pending_raw_event_received_at = await self._scalar_optional(
            select(func.min(RawKickEventModel.received_at)).where(
                RawKickEventModel.status == RawEventStatus.PENDING.value
            )
        )
        return OperationsTimestampsDTO(
            latest_message_at=latest_message_at,
            latest_raw_event_received_at=latest_raw_event_received_at,
            latest_raw_event_processed_at=latest_raw_event_processed_at,
            oldest_pending_raw_event_received_at=oldest_pending_raw_event_received_at,
        )

    async def _count(self, model: type) -> int:
        return await self._scalar_int(select(func.count()).select_from(model))

    async def _table_size(self, table_name: str) -> int:
        return await self._scalar_int(
            text("select coalesce(pg_total_relation_size(to_regclass(:table_name)), 0)"),
            {"table_name": table_name},
        )

    async def _scalar_int(self, statement, parameters: dict[str, object] | None = None) -> int:
        result = await self._session.execute(statement, parameters or {})
        return int(result.scalar_one() or 0)

    async def _scalar_optional(self, statement):
        result = await self._session.execute(statement)
        return result.scalar_one_or_none()
