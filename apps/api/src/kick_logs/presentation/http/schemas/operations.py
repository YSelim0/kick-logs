from datetime import datetime

from pydantic import BaseModel

from kick_logs.application.dto.operations import OperationsSummaryDTO


class OperationsCountsResponse(BaseModel):
    channels: int
    enabled_channels: int
    senders: int
    messages: int
    raw_events: int


class OperationsStorageTableResponse(BaseModel):
    table_name: str
    total_bytes: int


class OperationsStorageResponse(BaseModel):
    database_bytes: int
    tables: list[OperationsStorageTableResponse]


class OperationsTimestampsResponse(BaseModel):
    latest_message_at: datetime | None
    latest_raw_event_received_at: datetime | None
    latest_raw_event_processed_at: datetime | None
    oldest_pending_raw_event_received_at: datetime | None


class ListenerHeartbeatResponse(BaseModel):
    service_name: str
    last_seen_at: datetime | None
    is_fresh: bool
    stale_after_seconds: int
    seconds_since_last_seen: int | None


class OperationsSummaryResponse(BaseModel):
    counts: OperationsCountsResponse
    raw_event_status_counts: dict[str, int]
    storage: OperationsStorageResponse
    timestamps: OperationsTimestampsResponse
    listener: ListenerHeartbeatResponse

    @classmethod
    def from_dto(cls, dto: OperationsSummaryDTO) -> "OperationsSummaryResponse":
        if dto.storage is None or dto.timestamps is None or dto.listener is None:
            raise ValueError("Operations summary DTO is incomplete.")

        return cls(
            counts=OperationsCountsResponse(
                channels=dto.counts.channels,
                enabled_channels=dto.counts.enabled_channels,
                senders=dto.counts.senders,
                messages=dto.counts.messages,
                raw_events=dto.counts.raw_events,
            ),
            raw_event_status_counts=dto.raw_event_status_counts,
            storage=OperationsStorageResponse(
                database_bytes=dto.storage.database_bytes,
                tables=[
                    OperationsStorageTableResponse(
                        table_name=table.table_name,
                        total_bytes=table.total_bytes,
                    )
                    for table in dto.storage.tables
                ],
            ),
            timestamps=OperationsTimestampsResponse(
                latest_message_at=dto.timestamps.latest_message_at,
                latest_raw_event_received_at=dto.timestamps.latest_raw_event_received_at,
                latest_raw_event_processed_at=dto.timestamps.latest_raw_event_processed_at,
                oldest_pending_raw_event_received_at=(
                    dto.timestamps.oldest_pending_raw_event_received_at
                ),
            ),
            listener=ListenerHeartbeatResponse(
                service_name=dto.listener.service_name,
                last_seen_at=dto.listener.last_seen_at,
                is_fresh=dto.listener.is_fresh,
                stale_after_seconds=dto.listener.stale_after_seconds,
                seconds_since_last_seen=dto.listener.seconds_since_last_seen,
            ),
        )
