from dataclasses import dataclass, field
from typing import Any

from kick_logs.domain.exceptions import DomainError


@dataclass(slots=True)
class Sender:
    kick_user_id: int
    username: str
    slug: str
    id: int | None = None
    profile_image_url: str | None = None
    last_seen_color: str | None = None
    raw_profile_payload: dict[str, Any] = field(default_factory=dict)

    def __post_init__(self) -> None:
        normalized_slug = self.slug.strip().lower()
        if self.kick_user_id < 1:
            raise DomainError("Kick user id must be positive.")
        if not self.username.strip():
            raise DomainError("Sender username is required.")
        if not normalized_slug:
            raise DomainError("Sender slug is required.")
        self.slug = normalized_slug
