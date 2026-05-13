from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from kick_logs.domain.entities.worker_heartbeat import WorkerHeartbeat
from kick_logs.infrastructure.database.mappers import worker_heartbeat_to_domain
from kick_logs.infrastructure.database.models import WorkerHeartbeatModel


class SqlAlchemyWorkerHeartbeatRepository:
    def __init__(self, session: AsyncSession) -> None:
        self._session = session

    async def upsert(self, heartbeat: WorkerHeartbeat) -> WorkerHeartbeat:
        model = await self._session.get(WorkerHeartbeatModel, heartbeat.service_name)
        if model is None:
            model = WorkerHeartbeatModel(
                service_name=heartbeat.service_name,
                last_seen_at=heartbeat.last_seen_at,
                heartbeat_metadata=heartbeat.metadata,
            )
            self._session.add(model)
        else:
            model.last_seen_at = heartbeat.last_seen_at
            model.heartbeat_metadata = heartbeat.metadata

        await self._session.flush()
        await self._session.refresh(model)
        return worker_heartbeat_to_domain(model)

    async def get_by_service_name(self, service_name: str) -> WorkerHeartbeat | None:
        result = await self._session.execute(
            select(WorkerHeartbeatModel).where(WorkerHeartbeatModel.service_name == service_name)
        )
        model = result.scalar_one_or_none()
        return worker_heartbeat_to_domain(model) if model else None
