from collections.abc import Callable
from datetime import UTC, datetime
from typing import Any

from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.domain.entities.worker_heartbeat import WorkerHeartbeat

UnitOfWorkFactory = Callable[[], UnitOfWork]


class RecordWorkerHeartbeatUseCase:
    def __init__(self, unit_of_work_factory: UnitOfWorkFactory) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(
        self,
        *,
        service_name: str,
        metadata: dict[str, Any] | None = None,
    ) -> WorkerHeartbeat:
        heartbeat = WorkerHeartbeat(
            service_name=service_name,
            last_seen_at=datetime.now(UTC),
            metadata=metadata or {},
        )
        async with self._unit_of_work_factory() as unit_of_work:
            saved = await unit_of_work.worker_heartbeats.upsert(heartbeat)
            await unit_of_work.commit()
        return saved
