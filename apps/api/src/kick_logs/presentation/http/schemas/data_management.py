from datetime import datetime

from pydantic import BaseModel, Field, field_validator

from kick_logs.application.dto.data_management import (
    CleanupTarget,
    DataCleanupCountsDTO,
    DataCleanupPreviewDTO,
    DataCleanupRequestDTO,
    DataCleanupResultDTO,
    DataManagementSummaryDTO,
    RetentionSettingsDTO,
    UpdateRetentionSettingsDTO,
)
from kick_logs.application.use_cases.data_management.cleanup_support import ALLOWED_RETENTION_DAYS


class RetentionSettingsRequest(BaseModel):
    message_retention_days: int | None
    raw_event_retention_days: int | None

    @field_validator("message_retention_days", "raw_event_retention_days")
    @classmethod
    def validate_retention_days(cls, value: int | None) -> int | None:
        if value is not None and value not in ALLOWED_RETENTION_DAYS:
            allowed = ", ".join(str(days) for days in sorted(ALLOWED_RETENTION_DAYS))
            raise ValueError(f"Retention days must be one of: {allowed}.")
        return value

    def to_dto(self) -> UpdateRetentionSettingsDTO:
        return UpdateRetentionSettingsDTO(
            message_retention_days=self.message_retention_days,
            raw_event_retention_days=self.raw_event_retention_days,
        )


class RetentionSettingsResponse(BaseModel):
    message_retention_days: int | None
    raw_event_retention_days: int | None
    updated_at: datetime | None

    @classmethod
    def from_dto(cls, dto: RetentionSettingsDTO) -> "RetentionSettingsResponse":
        return cls(
            message_retention_days=dto.message_retention_days,
            raw_event_retention_days=dto.raw_event_retention_days,
            updated_at=dto.updated_at,
        )


class DataManagementCountsResponse(BaseModel):
    channels: int
    senders: int
    messages: int
    raw_events: int


class DataManagementTableResponse(BaseModel):
    table_name: str
    total_bytes: int
    row_count: int


class DataManagementSummaryResponse(BaseModel):
    counts: DataManagementCountsResponse
    database_bytes: int
    tables: list[DataManagementTableResponse]
    retention_settings: RetentionSettingsResponse

    @classmethod
    def from_dto(cls, dto: DataManagementSummaryDTO) -> "DataManagementSummaryResponse":
        return cls(
            counts=DataManagementCountsResponse(
                channels=dto.counts.channels,
                senders=dto.counts.senders,
                messages=dto.counts.messages,
                raw_events=dto.counts.raw_events,
            ),
            database_bytes=dto.database_bytes,
            tables=[
                DataManagementTableResponse(
                    table_name=table.table_name,
                    total_bytes=table.total_bytes,
                    row_count=table.row_count,
                )
                for table in dto.tables
            ],
            retention_settings=RetentionSettingsResponse.from_dto(dto.retention_settings),
        )


class DataCleanupRequest(BaseModel):
    target: CleanupTarget
    channel_slug: str | None = Field(default=None, max_length=160)
    sender: str | None = Field(default=None, max_length=160)

    def to_dto(self) -> DataCleanupRequestDTO:
        return DataCleanupRequestDTO(
            target=self.target,
            channel_slug=self.channel_slug,
            sender=self.sender,
        )


class DataCleanupConfirmRequest(DataCleanupRequest):
    confirmation_text: str = Field(min_length=1, max_length=240)


class DataCleanupCountsResponse(BaseModel):
    messages: int
    raw_events: int
    total: int

    @classmethod
    def from_dto(cls, dto: DataCleanupCountsDTO) -> "DataCleanupCountsResponse":
        return cls(messages=dto.messages, raw_events=dto.raw_events, total=dto.total)


class DataCleanupPreviewResponse(BaseModel):
    target: CleanupTarget
    affected: DataCleanupCountsResponse
    confirmation_text: str
    can_execute: bool
    cutoff_at: datetime | None
    channel_slug: str | None
    sender: str | None
    retention_days: int | None
    reason: str | None

    @classmethod
    def from_dto(cls, dto: DataCleanupPreviewDTO) -> "DataCleanupPreviewResponse":
        return cls(
            target=dto.target,
            affected=DataCleanupCountsResponse.from_dto(dto.affected),
            confirmation_text=dto.confirmation_text,
            can_execute=dto.can_execute,
            cutoff_at=dto.cutoff_at,
            channel_slug=dto.channel_slug,
            sender=dto.sender,
            retention_days=dto.retention_days,
            reason=dto.reason,
        )


class DataCleanupResultResponse(BaseModel):
    target: CleanupTarget
    deleted: DataCleanupCountsResponse
    confirmation_text: str
    cutoff_at: datetime | None
    channel_slug: str | None
    sender: str | None
    retention_days: int | None

    @classmethod
    def from_dto(cls, dto: DataCleanupResultDTO) -> "DataCleanupResultResponse":
        return cls(
            target=dto.target,
            deleted=DataCleanupCountsResponse.from_dto(dto.deleted),
            confirmation_text=dto.confirmation_text,
            cutoff_at=dto.cutoff_at,
            channel_slug=dto.channel_slug,
            sender=dto.sender,
            retention_days=dto.retention_days,
        )
