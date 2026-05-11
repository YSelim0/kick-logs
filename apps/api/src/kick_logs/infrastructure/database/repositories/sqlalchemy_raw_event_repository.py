from datetime import UTC, datetime, timedelta

from sqlalchemy import func, or_, select
from sqlalchemy.ext.asyncio import AsyncSession

from kick_logs.domain.entities.raw_kick_event import RawKickEvent
from kick_logs.domain.value_objects.raw_event_status import RawEventStatus
from kick_logs.infrastructure.database.mappers import (
    raw_kick_event_to_domain,
    raw_kick_event_to_model,
)
from kick_logs.infrastructure.database.models import RawKickEventModel


class SqlAlchemyRawEventRepository:
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def add(self, event: RawKickEvent) -> RawKickEvent:
        model = raw_kick_event_to_model(event)
        self._session.add(model)
        await self._session.flush()
        await self._session.refresh(model)
        return raw_kick_event_to_domain(model)

    async def get_by_id(self, event_id: int) -> RawKickEvent | None:
        model = await self._session.get(RawKickEventModel, event_id)
        return raw_kick_event_to_domain(model) if model else None

    async def get_by_kick_message_id(self, kick_message_id: str) -> RawKickEvent | None:
        result = await self._session.execute(
            select(RawKickEventModel).where(RawKickEventModel.kick_message_id == kick_message_id)
        )
        model = result.scalar_one_or_none()
        return raw_kick_event_to_domain(model) if model else None

    async def claim_pending(
        self,
        *,
        limit: int,
        processing_timeout_seconds: int,
    ) -> list[RawKickEvent]:
        now = datetime.now(UTC)
        stale_before = now - timedelta(seconds=processing_timeout_seconds)
        statement = (
            select(RawKickEventModel)
            .where(
                or_(
                    RawKickEventModel.status == RawEventStatus.PENDING.value,
                    (
                        (RawKickEventModel.status == RawEventStatus.PROCESSING.value)
                        & (RawKickEventModel.processing_started_at < stale_before)
                    ),
                )
            )
            .order_by(RawKickEventModel.received_at.asc(), RawKickEventModel.id.asc())
            .limit(limit)
            .with_for_update(skip_locked=True)
        )
        result = await self._session.execute(statement)
        models = list(result.scalars().all())
        for model in models:
            model.status = RawEventStatus.PROCESSING.value
            model.processing_started_at = now
        await self._session.flush()
        for model in models:
            await self._session.refresh(model)
        return [raw_kick_event_to_domain(model) for model in models]

    async def mark_processed(self, event_id: int) -> RawKickEvent:
        model = await self._get_required(event_id)
        model.status = RawEventStatus.PROCESSED.value
        model.processed_at = datetime.now(UTC)
        model.processing_started_at = None
        model.last_error = None
        await self._session.flush()
        await self._session.refresh(model)
        return raw_kick_event_to_domain(model)

    async def mark_failed(
        self,
        *,
        event_id: int,
        error: str,
        max_attempts: int,
    ) -> RawKickEvent:
        model = await self._get_required(event_id)
        attempts = model.attempts + 1
        model.attempts = attempts
        model.status = (
            RawEventStatus.FAILED.value
            if attempts >= max_attempts
            else RawEventStatus.PENDING.value
        )
        model.processing_started_at = None
        model.last_error = error[:4000]
        await self._session.flush()
        await self._session.refresh(model)
        return raw_kick_event_to_domain(model)

    async def pending_count(self) -> int:
        result = await self._session.execute(
            select(func.count())
            .select_from(RawKickEventModel)
            .where(RawKickEventModel.status == RawEventStatus.PENDING.value)
        )
        return int(result.scalar_one())

    async def _get_required(self, event_id: int) -> RawKickEventModel:
        model = await self._session.get(RawKickEventModel, event_id)
        if model is None:
            raise ValueError("Raw Kick event not found.")
        return model
