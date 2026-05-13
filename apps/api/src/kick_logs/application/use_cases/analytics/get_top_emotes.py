from collections.abc import Callable

from kick_logs.application.dto.analytics import TopEmoteDTO
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.domain.value_objects.analytics_filters import AnalyticsFilters


class GetTopEmotesUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._unit_of_work_factory = unit_of_work_factory

    async def execute(self, filters: AnalyticsFilters, limit: int) -> list[TopEmoteDTO]:
        async with self._unit_of_work_factory() as unit_of_work:
            return await unit_of_work.analytics.get_top_emotes(filters, limit)
