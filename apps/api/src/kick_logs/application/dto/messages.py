from dataclasses import dataclass
from datetime import datetime
from typing import Any

from kick_logs.domain.entities import ChatMessage


@dataclass(frozen=True, slots=True)
class ChatMessageDTO:
    id: int
    kick_message_id: str
    channel_id: int
    sender_id: int
    chatroom_id: int
    content: str
    message_type: str
    sender_username_snapshot: str
    sender_slug_snapshot: str
    sender_color_snapshot: str | None
    sender_badges: list[dict[str, Any]]
    emotes: list[dict[str, str]]
    reply_metadata: dict[str, Any]
    thread_parent_id: str | None
    raw_payload: dict[str, Any]
    message_created_at: datetime
    ingested_at: datetime


def chat_message_to_dto(message: ChatMessage) -> ChatMessageDTO:
    if message.id is None:
        raise ValueError("Message id is required for API responses.")
    return ChatMessageDTO(
        id=message.id,
        kick_message_id=message.kick_message_id,
        channel_id=message.channel_id,
        sender_id=message.sender_id,
        chatroom_id=message.chatroom_id,
        content=message.content,
        message_type=message.message_type,
        sender_username_snapshot=message.sender_username_snapshot,
        sender_slug_snapshot=message.sender_slug_snapshot,
        sender_color_snapshot=message.sender_color_snapshot,
        sender_badges=message.sender_badges,
        emotes=[emote.to_dict() for emote in message.emotes],
        reply_metadata=message.reply_metadata,
        thread_parent_id=message.thread_parent_id,
        raw_payload=message.raw_payload,
        message_created_at=message.message_created_at,
        ingested_at=message.ingested_at,
    )
