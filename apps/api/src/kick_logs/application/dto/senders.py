from dataclasses import dataclass, field
from typing import Any


@dataclass(frozen=True, slots=True)
class ResolvedSenderProfileDTO:
    slug: str
    username: str | None = None
    profile_image_url: str | None = None
    raw_payload: dict[str, Any] = field(default_factory=dict)
