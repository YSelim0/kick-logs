from dataclasses import dataclass
from datetime import datetime
from typing import Literal

CleanupTarget = Literal["old_messages", "old_raw_events", "channel", "sender"]


@dataclass(frozen=True, slots=True)
class RetentionSettingsDTO:
    message_retention_days: int | None
    raw_event_retention_days: int | None
    updated_at: datetime | None = None


@dataclass(frozen=True, slots=True)
class UpdateRetentionSettingsDTO:
    message_retention_days: int | None
    raw_event_retention_days: int | None


@dataclass(frozen=True, slots=True)
class DataManagementCountsDTO:
    channels: int
    senders: int
    messages: int
    raw_events: int


@dataclass(frozen=True, slots=True)
class DataManagementTableDTO:
    table_name: str
    total_bytes: int
    row_count: int


@dataclass(frozen=True, slots=True)
class DataManagementSummaryDTO:
    counts: DataManagementCountsDTO
    database_bytes: int
    tables: list[DataManagementTableDTO]
    retention_settings: RetentionSettingsDTO


@dataclass(frozen=True, slots=True)
class DataCleanupRequestDTO:
    target: CleanupTarget
    channel_slug: str | None = None
    sender: str | None = None


@dataclass(frozen=True, slots=True)
class DataCleanupCriteriaDTO:
    target: CleanupTarget
    cutoff_at: datetime | None = None
    channel_slug: str | None = None
    sender: str | None = None
    retention_days: int | None = None


@dataclass(frozen=True, slots=True)
class DataCleanupCountsDTO:
    messages: int = 0
    raw_events: int = 0

    @property
    def total(self) -> int:
        return self.messages + self.raw_events


@dataclass(frozen=True, slots=True)
class DataCleanupPreviewDTO:
    target: CleanupTarget
    affected: DataCleanupCountsDTO
    confirmation_text: str
    can_execute: bool
    cutoff_at: datetime | None = None
    channel_slug: str | None = None
    sender: str | None = None
    retention_days: int | None = None
    reason: str | None = None


@dataclass(frozen=True, slots=True)
class DataCleanupResultDTO:
    target: CleanupTarget
    deleted: DataCleanupCountsDTO
    confirmation_text: str
    cutoff_at: datetime | None = None
    channel_slug: str | None = None
    sender: str | None = None
    retention_days: int | None = None
