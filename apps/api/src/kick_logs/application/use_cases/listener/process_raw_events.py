from collections.abc import Callable
from dataclasses import dataclass

from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.application.use_cases.messages import IngestMessageUseCase
from kick_logs.domain.entities.raw_kick_event import RawKickEvent

UnitOfWorkFactory = Callable[[], UnitOfWork]


@dataclass(frozen=True, slots=True)
class RawEventProcessingResult:
    claimed: int
    processed: int
    failed: int
    pending_count: int


class ProcessRawKickEventsUseCase:
    def __init__(self, unit_of_work_factory: UnitOfWorkFactory) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute_once(
        self,
        *,
        limit: int,
        processing_timeout_seconds: int,
        max_attempts: int,
    ) -> RawEventProcessingResult:
        raw_events = await self._claim_events(
            limit=limit,
            processing_timeout_seconds=processing_timeout_seconds,
        )

        processed = 0
        failed = 0
        for raw_event in raw_events:
            try:
                await IngestMessageUseCase(self._unit_of_work_factory).execute(raw_event.payload)
                await self._mark_processed(raw_event)
            except Exception as exc:
                await self._mark_failed(raw_event, exc, max_attempts=max_attempts)
                failed += 1
            else:
                processed += 1

        pending_count = await self._pending_count()
        return RawEventProcessingResult(
            claimed=len(raw_events),
            processed=processed,
            failed=failed,
            pending_count=pending_count,
        )

    async def _claim_events(
        self,
        *,
        limit: int,
        processing_timeout_seconds: int,
    ) -> list[RawKickEvent]:
        async with self._unit_of_work_factory() as unit_of_work:
            raw_events = await unit_of_work.raw_events.claim_pending(
                limit=limit,
                processing_timeout_seconds=processing_timeout_seconds,
            )
            await unit_of_work.commit()
            return raw_events

    async def _mark_processed(self, raw_event: RawKickEvent) -> None:
        if raw_event.id is None:
            return

        async with self._unit_of_work_factory() as unit_of_work:
            await unit_of_work.raw_events.mark_processed(raw_event.id)
            await unit_of_work.commit()

    async def _mark_failed(
        self,
        raw_event: RawKickEvent,
        exc: Exception,
        *,
        max_attempts: int,
    ) -> None:
        if raw_event.id is None:
            return

        async with self._unit_of_work_factory() as unit_of_work:
            await unit_of_work.raw_events.mark_failed(
                event_id=raw_event.id,
                error=f"{type(exc).__name__}: {exc}",
                max_attempts=max_attempts,
            )
            await unit_of_work.commit()

    async def _pending_count(self) -> int:
        async with self._unit_of_work_factory() as unit_of_work:
            count = await unit_of_work.raw_events.pending_count()
            await unit_of_work.commit()
            return count
