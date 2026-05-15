from collections.abc import Callable

from kick_logs.application.dto.messages import MessageExportDTO, MessageSearchItemDTO
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.application.use_cases.messages.search_messages import SearchMessagesUseCase
from kick_logs.domain.value_objects.pagination import CursorPagination
from kick_logs.domain.value_objects.search_filters import MessageSearchFilters


class ExportMessagesUseCase:
    def __init__(self, unit_of_work_factory: Callable[[], UnitOfWork]) -> None:
        self._search_messages = SearchMessagesUseCase(unit_of_work_factory)

    async def execute(
        self,
        filters: MessageSearchFilters,
        *,
        max_rows: int,
    ) -> MessageExportDTO:
        safe_max_rows = max(1, max_rows)
        items: list[MessageSearchItemDTO] = []
        cursor = None

        while len(items) < safe_max_rows:
            page = await self._search_messages.execute(
                filters,
                CursorPagination(limit=min(100, safe_max_rows - len(items)), cursor=cursor),
            )
            items.extend(page.items)

            if page.next_cursor is None:
                return MessageExportDTO(
                    items=items,
                    count=len(items),
                    max_rows=safe_max_rows,
                    truncated=False,
                )

            cursor = page.next_cursor

        return MessageExportDTO(
            items=items[:safe_max_rows],
            count=min(len(items), safe_max_rows),
            max_rows=safe_max_rows,
            truncated=cursor is not None,
        )
