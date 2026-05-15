from typing import Protocol

from kick_logs.application.dto.data_management import (
    DataCleanupCountsDTO,
    DataCleanupCriteriaDTO,
    DataManagementCountsDTO,
    DataManagementTableDTO,
    RetentionSettingsDTO,
)


class DataManagementRepository(Protocol):
    async def get_retention_settings(self) -> RetentionSettingsDTO: ...

    async def update_retention_settings(
        self,
        *,
        message_retention_days: int | None,
        raw_event_retention_days: int | None,
    ) -> RetentionSettingsDTO: ...

    async def get_counts(self) -> DataManagementCountsDTO: ...

    async def get_database_size(self) -> int: ...

    async def get_table_sizes(self) -> list[DataManagementTableDTO]: ...

    async def count_cleanup(self, criteria: DataCleanupCriteriaDTO) -> DataCleanupCountsDTO: ...

    async def execute_cleanup(self, criteria: DataCleanupCriteriaDTO) -> DataCleanupCountsDTO: ...
