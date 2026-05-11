from dataclasses import dataclass, field
from datetime import datetime
from typing import Any

from kick_logs.domain.exceptions import DomainError
from kick_logs.domain.value_objects.raw_event_status import RawEventStatus


@dataclass(slots=True)
class RawKickEvent:
    event_name: str
    payload: dict[str, Any]
    status: RawEventStatus = RawEventStatus.PENDING
    id: int | None = None
    kick_message_id: str | None = None
    chatroom_id: int | None = None
    kick_channel_id: int | None = None
    channel_id: int | None = None
    attempts: int = 0
    received_at: datetime | None = None
    processing_started_at: datetime | None = None
    processed_at: datetime | None = None
    last_error: str | None = None
    created_at: datetime | None = None
    updated_at: datetime | None = None
    metadata: dict[str, Any] = field(default_factory=dict)

    def __post_init__(self) -> None:
        if not self.event_name.strip():
            raise DomainError("Raw event name is required.")
        if not isinstance(self.payload, dict):
            raise DomainError("Raw event payload must be an object.")
        if self.attempts < 0:
            raise DomainError("Raw event attempts cannot be negative.")
        if self.chatroom_id is not None and self.chatroom_id < 1:
            raise DomainError("Raw event chatroom id must be positive.")
        if self.kick_channel_id is not None and self.kick_channel_id < 1:
            raise DomainError("Raw event Kick channel id must be positive.")
        if self.channel_id is not None and self.channel_id < 1:
            raise DomainError("Raw event channel id must be positive.")

    @classmethod
    def pending(
        cls,
        *,
        event_name: str,
        payload: dict[str, Any],
        kick_message_id: str | None,
        chatroom_id: int | None,
        kick_channel_id: int | None = None,
        channel_id: int | None = None,
        metadata: dict[str, Any] | None = None,
    ) -> "RawKickEvent":
        return cls(
            event_name=event_name,
            payload=payload,
            status=RawEventStatus.PENDING,
            kick_message_id=kick_message_id,
            chatroom_id=chatroom_id,
            kick_channel_id=kick_channel_id,
            channel_id=channel_id,
            metadata=metadata or {},
        )
