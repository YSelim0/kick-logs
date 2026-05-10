from datetime import datetime
from typing import Any

from pydantic import BaseModel

from kick_logs.application.dto.messages import MessageSearchItemDTO, MessageSearchPageDTO
from kick_logs.domain.value_objects.pagination import MessageCursor


class MessageEmoteResponse(BaseModel):
    id: str
    name: str
    token: str
    image_url: str


class MessageSenderResponse(BaseModel):
    id: int
    kick_user_id: int
    username: str
    slug: str
    profile_image_url: str | None


class MessageChannelResponse(BaseModel):
    id: int
    slug: str
    display_name: str
    profile_image_url: str | None
    banner_image_url: str | None


class MessageResponse(BaseModel):
    id: int
    kick_message_id: str
    chatroom_id: int
    content: str
    message_type: str
    sender_username_snapshot: str
    sender_slug_snapshot: str
    sender_color_snapshot: str | None
    sender_badges: list[dict[str, Any]]
    emotes: list[MessageEmoteResponse]
    reply_metadata: dict[str, Any]
    thread_parent_id: str | None
    message_created_at: datetime
    ingested_at: datetime
    sender: MessageSenderResponse
    channel: MessageChannelResponse

    @classmethod
    def from_dto(cls, item: MessageSearchItemDTO) -> "MessageResponse":
        return cls(
            id=item.message.id,
            kick_message_id=item.message.kick_message_id,
            chatroom_id=item.message.chatroom_id,
            content=item.message.content,
            message_type=item.message.message_type,
            sender_username_snapshot=item.message.sender_username_snapshot,
            sender_slug_snapshot=item.message.sender_slug_snapshot,
            sender_color_snapshot=item.message.sender_color_snapshot,
            sender_badges=item.message.sender_badges,
            emotes=[MessageEmoteResponse(**emote) for emote in item.message.emotes],
            reply_metadata=item.message.reply_metadata,
            thread_parent_id=item.message.thread_parent_id,
            message_created_at=item.message.message_created_at,
            ingested_at=item.message.ingested_at,
            sender=MessageSenderResponse(
                id=item.sender.id,
                kick_user_id=item.sender.kick_user_id,
                username=item.sender.username,
                slug=item.sender.slug,
                profile_image_url=item.sender.profile_image_url,
            ),
            channel=MessageChannelResponse(
                id=item.channel.id,
                slug=item.channel.slug,
                display_name=item.channel.display_name,
                profile_image_url=item.channel.profile_image_url,
                banner_image_url=item.channel.banner_image_url,
            ),
        )


class MessageSearchResponse(BaseModel):
    items: list[MessageResponse]
    next_cursor: str | None

    @classmethod
    def from_dto(cls, page: MessageSearchPageDTO) -> "MessageSearchResponse":
        return cls(
            items=[MessageResponse.from_dto(item) for item in page.items],
            next_cursor=cls._cursor_to_text(page.next_cursor),
        )

    @staticmethod
    def _cursor_to_text(cursor: MessageCursor | None) -> str | None:
        if cursor is None:
            return None
        return f"{cursor.message_created_at.isoformat()}|{cursor.message_id}"
