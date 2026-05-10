from dataclasses import dataclass, field
from datetime import UTC, datetime
from typing import Any

from kick_logs.domain.exceptions import DomainError


@dataclass(slots=True)
class Channel:
    slug: str
    display_name: str
    id: int | None = None
    kick_channel_id: int | None = None
    kick_chatroom_id: int | None = None
    profile_image_url: str | None = None
    banner_image_url: str | None = None
    is_enabled: bool = True
    raw_payload: dict[str, Any] = field(default_factory=dict)
    created_at: datetime | None = None
    updated_at: datetime | None = None

    def __post_init__(self) -> None:
        normalized_slug = self.slug.strip().lower()
        if not normalized_slug:
            raise DomainError("Channel slug is required.")
        if not self.display_name.strip():
            raise DomainError("Channel display name is required.")
        self.slug = normalized_slug

    def enable(self) -> None:
        self.is_enabled = True
        self.updated_at = datetime.now(UTC)

    def disable(self) -> None:
        self.is_enabled = False
        self.updated_at = datetime.now(UTC)
