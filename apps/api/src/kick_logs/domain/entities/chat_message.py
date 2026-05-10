from dataclasses import dataclass, field
from datetime import UTC, datetime
from typing import Any

from kick_logs.domain.entities.emote import Emote
from kick_logs.domain.exceptions import DomainError


@dataclass(slots=True)
class ChatMessage:
    kick_message_id: str
    channel_id: int
    sender_id: int
    chatroom_id: int
    content: str
    message_type: str
    sender_username_snapshot: str
    sender_slug_snapshot: str
    message_created_at: datetime
    id: int | None = None
    sender_color_snapshot: str | None = None
    sender_badges: list[dict[str, Any]] = field(default_factory=list)
    emotes: list[Emote] = field(default_factory=list)
    reply_metadata: dict[str, Any] = field(default_factory=dict)
    thread_parent_id: str | None = None
    raw_payload: dict[str, Any] = field(default_factory=dict)
    ingested_at: datetime = field(default_factory=lambda: datetime.now(UTC))

    def __post_init__(self) -> None:
        if not self.kick_message_id.strip():
            raise DomainError("Kick message id is required.")
        if self.channel_id < 1:
            raise DomainError("Channel id must be positive.")
        if self.sender_id < 1:
            raise DomainError("Sender id must be positive.")
        if self.chatroom_id < 1:
            raise DomainError("Chatroom id must be positive.")
        if not self.sender_username_snapshot.strip():
            raise DomainError("Sender username snapshot is required.")
        if not self.sender_slug_snapshot.strip():
            raise DomainError("Sender slug snapshot is required.")
        if self.message_created_at.tzinfo is None:
            raise DomainError("Message created datetime must be timezone-aware.")
