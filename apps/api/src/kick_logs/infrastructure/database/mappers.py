from typing import Any

from kick_logs.domain.entities import Channel, ChatMessage, Emote, Sender, User
from kick_logs.domain.value_objects.roles import UserRole
from kick_logs.infrastructure.database.models import (
    ChannelModel,
    ChatMessageModel,
    SenderModel,
    UserModel,
)


def user_to_domain(model: UserModel) -> User:
    return User(
        id=model.id,
        email=model.email,
        password_hash=model.password_hash,
        role=UserRole(model.role),
        is_active=model.is_active,
        created_at=model.created_at,
        updated_at=model.updated_at,
    )


def channel_to_domain(model: ChannelModel) -> Channel:
    return Channel(
        id=model.id,
        kick_channel_id=model.kick_channel_id,
        kick_chatroom_id=model.kick_chatroom_id,
        slug=model.slug,
        display_name=model.display_name,
        profile_image_url=model.profile_image_url,
        banner_image_url=model.banner_image_url,
        is_enabled=model.is_enabled,
        raw_payload=model.raw_payload or {},
        created_at=model.created_at,
        updated_at=model.updated_at,
    )


def sender_to_domain(model: SenderModel) -> Sender:
    return Sender(
        id=model.id,
        kick_user_id=model.kick_user_id,
        username=model.username,
        slug=model.slug,
        profile_image_url=model.profile_image_url,
        last_seen_color=model.last_seen_color,
        raw_profile_payload=model.raw_profile_payload or {},
        created_at=model.created_at,
        updated_at=model.updated_at,
    )


def _emote_to_dict(emote: Emote) -> dict[str, str]:
    return emote.to_dict()


def _emote_from_dict(payload: dict[str, Any]) -> Emote:
    return Emote(
        kick_emote_id=str(payload["id"]),
        name=str(payload["name"]),
        token=str(payload["token"]),
    )


def chat_message_to_domain(model: ChatMessageModel) -> ChatMessage:
    return ChatMessage(
        id=model.id,
        kick_message_id=model.kick_message_id,
        channel_id=model.channel_id,
        sender_id=model.sender_id,
        chatroom_id=model.chatroom_id,
        content=model.content,
        message_type=model.message_type,
        sender_username_snapshot=model.sender_username_snapshot,
        sender_slug_snapshot=model.sender_slug_snapshot,
        sender_color_snapshot=model.sender_color_snapshot,
        sender_badges=model.sender_badges or [],
        emotes=[_emote_from_dict(emote) for emote in model.emotes or []],
        reply_metadata=model.reply_metadata or {},
        thread_parent_id=model.thread_parent_id,
        raw_payload=model.raw_payload or {},
        message_created_at=model.message_created_at,
        ingested_at=model.ingested_at,
    )


def user_to_model(entity: User) -> UserModel:
    return UserModel(
        email=entity.email,
        password_hash=entity.password_hash,
        role=entity.role.value,
        is_active=entity.is_active,
    )


def channel_to_model(entity: Channel) -> ChannelModel:
    return ChannelModel(
        kick_channel_id=entity.kick_channel_id,
        kick_chatroom_id=entity.kick_chatroom_id,
        slug=entity.slug,
        display_name=entity.display_name,
        profile_image_url=entity.profile_image_url,
        banner_image_url=entity.banner_image_url,
        is_enabled=entity.is_enabled,
        raw_payload=entity.raw_payload,
    )


def sender_to_model(entity: Sender) -> SenderModel:
    return SenderModel(
        kick_user_id=entity.kick_user_id,
        username=entity.username,
        slug=entity.slug,
        profile_image_url=entity.profile_image_url,
        last_seen_color=entity.last_seen_color,
        raw_profile_payload=entity.raw_profile_payload,
    )


def chat_message_to_model(entity: ChatMessage) -> ChatMessageModel:
    return ChatMessageModel(
        kick_message_id=entity.kick_message_id,
        channel_id=entity.channel_id,
        sender_id=entity.sender_id,
        chatroom_id=entity.chatroom_id,
        content=entity.content,
        message_type=entity.message_type,
        sender_username_snapshot=entity.sender_username_snapshot,
        sender_slug_snapshot=entity.sender_slug_snapshot,
        sender_color_snapshot=entity.sender_color_snapshot,
        sender_badges=entity.sender_badges,
        emotes=[_emote_to_dict(emote) for emote in entity.emotes],
        reply_metadata=entity.reply_metadata,
        thread_parent_id=entity.thread_parent_id,
        raw_payload=entity.raw_payload,
        message_created_at=entity.message_created_at,
        ingested_at=entity.ingested_at,
    )
