from typing import Protocol

from kick_logs.domain.entities.worker_heartbeat import WorkerHeartbeat


class WorkerHeartbeatRepository(Protocol):
    async def upsert(self, heartbeat: WorkerHeartbeat) -> WorkerHeartbeat: ...

    async def get_by_service_name(self, service_name: str) -> WorkerHeartbeat | None: ...
