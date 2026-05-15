from collections.abc import Callable

from kick_logs.application.dto.data_management import (
    DataCleanupPreviewDTO,
    DataCleanupRequestDTO,
)
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.application.use_cases.data_management.cleanup_support import (
    build_cleanup_criteria,
    build_preview,
)


class PreviewDataCleanupUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(self, request: DataCleanupRequestDTO) -> DataCleanupPreviewDTO:
        async with self._unit_of_work_factory() as unit_of_work:
            settings = await unit_of_work.data_management.get_retention_settings()
            criteria = build_cleanup_criteria(request, settings)
            affected = await unit_of_work.data_management.count_cleanup(criteria)
            return build_preview(
                criteria=criteria,
                affected_messages=affected.messages,
                affected_raw_events=affected.raw_events,
            )
