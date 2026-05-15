from collections.abc import Callable

from kick_logs.application.dto.data_management import DataManagementSummaryDTO
from kick_logs.application.ports.unit_of_work import UnitOfWork


class GetDataManagementSummaryUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(self) -> DataManagementSummaryDTO:
        async with self._unit_of_work_factory() as unit_of_work:
            return DataManagementSummaryDTO(
                counts=await unit_of_work.data_management.get_counts(),
                database_bytes=await unit_of_work.data_management.get_database_size(),
                tables=await unit_of_work.data_management.get_table_sizes(),
                retention_settings=await unit_of_work.data_management.get_retention_settings(),
            )
