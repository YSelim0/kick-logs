from dataclasses import dataclass
from datetime import datetime
from typing import Literal

from kick_logs.domain.exceptions import DomainError

AnalyticsBucket = Literal["hour", "day"]


def _clean_optional_text(value: str | None) -> str | None:
    if value is None:
        return None
    cleaned = value.strip()
    return cleaned or None


@dataclass(frozen=True, slots=True)
class AnalyticsFilters:
    start: datetime | None = None
    end: datetime | None = None
    channel: str | None = None
    sender: str | None = None

    def __post_init__(self) -> None:
        object.__setattr__(self, "channel", _clean_optional_text(self.channel))
        object.__setattr__(self, "sender", _clean_optional_text(self.sender))

        if self.start and self.end and self.start > self.end:
            raise DomainError("Analytics start datetime must be before end datetime.")
