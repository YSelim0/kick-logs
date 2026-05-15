from typing import Protocol

from kick_logs.application.dto.operations import (
    OperationsCountsDTO,
    OperationsStorageDTO,
    OperationsTimestampsDTO,
)


class OperationsRepository(Protocol):
    async def get_counts(self) -> OperationsCountsDTO: ...

    async def get_raw_event_status_counts(self) -> dict[str, int]: ...

    async def get_storage(self) -> OperationsStorageDTO: ...

    async def get_timestamps(self) -> OperationsTimestampsDTO: ...
