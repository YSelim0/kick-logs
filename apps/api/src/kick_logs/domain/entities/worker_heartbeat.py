from dataclasses import dataclass, field
from datetime import datetime
from typing import Any

from kick_logs.domain.exceptions import DomainError


@dataclass(slots=True)
class WorkerHeartbeat:
    service_name: str
    last_seen_at: datetime
    metadata: dict[str, Any] = field(default_factory=dict)
    created_at: datetime | None = None
    updated_at: datetime | None = None

    def __post_init__(self) -> None:
        if not self.service_name.strip():
            raise DomainError("Worker heartbeat service name is required.")
        if not isinstance(self.metadata, dict):
            raise DomainError("Worker heartbeat metadata must be an object.")
