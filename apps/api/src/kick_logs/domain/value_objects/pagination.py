from dataclasses import dataclass
from datetime import datetime

from kick_logs.domain.exceptions import DomainError


@dataclass(frozen=True, slots=True)
class MessageCursor:
    message_created_at: datetime
    message_id: int


@dataclass(frozen=True, slots=True)
class CursorPagination:
    limit: int = 50
    cursor: MessageCursor | None = None

    def __post_init__(self) -> None:
        if self.limit < 1:
            raise DomainError("Pagination limit must be at least 1.")
        if self.limit > 100:
            raise DomainError("Pagination limit must not exceed 100.")
