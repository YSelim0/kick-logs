from collections.abc import Callable

from kick_logs.application.dto.data_management import (
    RetentionSettingsDTO,
    UpdateRetentionSettingsDTO,
)
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.application.use_cases.data_management.cleanup_support import validate_retention_days


class UpdateRetentionSettingsUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(self, settings: UpdateRetentionSettingsDTO) -> RetentionSettingsDTO:
        validate_retention_days(settings.message_retention_days)
        validate_retention_days(settings.raw_event_retention_days)

        async with self._unit_of_work_factory() as unit_of_work:
            updated = await unit_of_work.data_management.update_retention_settings(
                message_retention_days=settings.message_retention_days,
                raw_event_retention_days=settings.raw_event_retention_days,
            )
            await unit_of_work.commit()
            return updated
