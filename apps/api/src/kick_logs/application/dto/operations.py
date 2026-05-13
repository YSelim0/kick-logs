from dataclasses import dataclass, field
from datetime import datetime


@dataclass(frozen=True, slots=True)
class OperationsCountsDTO:
    channels: int
    enabled_channels: int
    senders: int
    messages: int
    raw_events: int


@dataclass(frozen=True, slots=True)
class OperationsStorageTableDTO:
    table_name: str
    total_bytes: int


@dataclass(frozen=True, slots=True)
class OperationsStorageDTO:
    database_bytes: int
    tables: list[OperationsStorageTableDTO]


@dataclass(frozen=True, slots=True)
class OperationsTimestampsDTO:
    latest_message_at: datetime | None
    latest_raw_event_received_at: datetime | None
    latest_raw_event_processed_at: datetime | None
    oldest_pending_raw_event_received_at: datetime | None


@dataclass(frozen=True, slots=True)
class ListenerHeartbeatDTO:
    service_name: str
    last_seen_at: datetime | None
    is_fresh: bool
    stale_after_seconds: int
    seconds_since_last_seen: int | None


@dataclass(frozen=True, slots=True)
class OperationsSummaryDTO:
    counts: OperationsCountsDTO
    raw_event_status_counts: dict[str, int] = field(default_factory=dict)
    storage: OperationsStorageDTO | None = None
    timestamps: OperationsTimestampsDTO | None = None
    listener: ListenerHeartbeatDTO | None = None
