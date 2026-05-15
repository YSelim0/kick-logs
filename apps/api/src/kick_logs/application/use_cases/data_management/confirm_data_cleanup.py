from collections.abc import Callable

from kick_logs.application.dto.data_management import (
    DataCleanupRequestDTO,
    DataCleanupResultDTO,
)
from kick_logs.application.exceptions import CleanupConfirmationError
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.application.use_cases.data_management.cleanup_support import (
    build_cleanup_criteria,
    build_confirmation_text,
    ensure_preview_can_execute,
)
from kick_logs.application.use_cases.data_management.preview_data_cleanup import (
    PreviewDataCleanupUseCase,
)


class ConfirmDataCleanupUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(
        self,
        request: DataCleanupRequestDTO,
        *,
        confirmation_text: str,
    ) -> DataCleanupResultDTO:
        preview = await PreviewDataCleanupUseCase(self._unit_of_work_factory).execute(request)
        ensure_preview_can_execute(preview)

        if confirmation_text != preview.confirmation_text:
            raise CleanupConfirmationError("Cleanup confirmation text does not match.")

        async with self._unit_of_work_factory() as unit_of_work:
            settings = await unit_of_work.data_management.get_retention_settings()
            criteria = build_cleanup_criteria(request, settings)
            if confirmation_text != build_confirmation_text(criteria):
                raise CleanupConfirmationError("Cleanup confirmation text does not match.")
            deleted = await unit_of_work.data_management.execute_cleanup(criteria)
            await unit_of_work.commit()
            return DataCleanupResultDTO(
                target=criteria.target,
                deleted=deleted,
                confirmation_text=confirmation_text,
                cutoff_at=criteria.cutoff_at,
                channel_slug=criteria.channel_slug,
                sender=criteria.sender,
                retention_days=criteria.retention_days,
            )
