from collections.abc import Callable
from datetime import UTC, datetime, timedelta

from kick_logs.application.dto.operations import ListenerHeartbeatDTO, OperationsSummaryDTO
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.domain.value_objects.raw_event_status import RawEventStatus

UnitOfWorkFactory = Callable[[], UnitOfWork]


class GetOperationsSummaryUseCase:
    def __init__(
        self,
        unit_of_work_factory: UnitOfWorkFactory,
        *,
        listener_service_name: str = "listener",
        heartbeat_stale_after_seconds: int = 45,
    ) -> None:
        self._unit_of_work_factory = unit_of_work_factory
        self._listener_service_name = listener_service_name
        self._heartbeat_stale_after_seconds = max(1, heartbeat_stale_after_seconds)

    async def execute(self) -> OperationsSummaryDTO:
        async with self._unit_of_work_factory() as unit_of_work:
            counts = await unit_of_work.operations.get_counts()
            raw_status_counts = await unit_of_work.operations.get_raw_event_status_counts()
            storage = await unit_of_work.operations.get_storage()
            timestamps = await unit_of_work.operations.get_timestamps()
            heartbeat = await unit_of_work.worker_heartbeats.get_by_service_name(
                self._listener_service_name
            )

        normalized_status_counts = {
            status.value: raw_status_counts.get(status.value, 0) for status in RawEventStatus
        }

        return OperationsSummaryDTO(
            counts=counts,
            raw_event_status_counts=normalized_status_counts,
            storage=storage,
            timestamps=timestamps,
            listener=self._build_listener_dto(
                last_seen_at=heartbeat.last_seen_at if heartbeat else None
            ),
        )

    def _build_listener_dto(self, last_seen_at: datetime | None) -> ListenerHeartbeatDTO:
        seconds_since_last_seen: int | None = None
        is_fresh = False
        if last_seen_at is not None:
            now = datetime.now(UTC)
            normalized_last_seen = (
                last_seen_at.replace(tzinfo=UTC)
                if last_seen_at.tzinfo is None
                else last_seen_at.astimezone(UTC)
            )
            seconds_since_last_seen = max(0, int((now - normalized_last_seen).total_seconds()))
            is_fresh = now - normalized_last_seen <= timedelta(
                seconds=self._heartbeat_stale_after_seconds
            )

        return ListenerHeartbeatDTO(
            service_name=self._listener_service_name,
            last_seen_at=last_seen_at,
            is_fresh=is_fresh,
            stale_after_seconds=self._heartbeat_stale_after_seconds,
            seconds_since_last_seen=seconds_since_last_seen,
        )
