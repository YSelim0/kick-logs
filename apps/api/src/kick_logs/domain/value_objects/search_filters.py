from dataclasses import dataclass
from datetime import datetime

from kick_logs.domain.exceptions import DomainError


def _clean_optional_text(value: str | None) -> str | None:
    if value is None:
        return None
    cleaned = value.strip()
    return cleaned or None


@dataclass(frozen=True, slots=True)
class MessageSearchFilters:
    sender: str | None = None
    channel: str | None = None
    q: str | None = None
    start: datetime | None = None
    end: datetime | None = None
    reply_only: bool = False
    emote_only: bool = False

    def __post_init__(self) -> None:
        object.__setattr__(self, "sender", _clean_optional_text(self.sender))
        object.__setattr__(self, "channel", _clean_optional_text(self.channel))
        object.__setattr__(self, "q", _clean_optional_text(self.q))

        if self.start and self.end and self.start > self.end:
            raise DomainError("Search start datetime must be before end datetime.")

    @property
    def has_any_filter(self) -> bool:
        return any(
            (
                self.sender,
                self.channel,
                self.q,
                self.start,
                self.end,
                self.reply_only,
                self.emote_only,
            )
        )
