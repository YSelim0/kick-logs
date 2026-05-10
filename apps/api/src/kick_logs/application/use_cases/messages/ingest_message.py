from collections.abc import Callable
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any

from kick_logs.application.dto.messages import ChatMessageDTO, chat_message_to_dto
from kick_logs.application.exceptions import ChannelNotFoundError, MessageIngestionError
from kick_logs.application.ports.unit_of_work import UnitOfWork
from kick_logs.application.services.emote_parser import EmoteParser
from kick_logs.domain.entities import ChatMessage, Sender


@dataclass(frozen=True, slots=True)
class _NormalizedKickMessage:
    kick_message_id: str
    chatroom_id: int
    content: str
    message_type: str
    message_created_at: datetime
    sender_kick_user_id: int
    sender_username: str
    sender_slug: str
    sender_color: str | None
    sender_badges: list[dict[str, Any]]
    sender_profile_image_url: str | None
    sender_payload: dict[str, Any]
    reply_metadata: dict[str, Any]
    thread_parent_id: str | None


class IngestMessageUseCase:
    def __init__(
        self,
        unit_of_work_factory: Callable[[], UnitOfWork],
        emote_parser: EmoteParser | None = None,
    ) -> None:
        self._unit_of_work_factory = unit_of_work_factory
        self._emote_parser = emote_parser or EmoteParser()

    async def execute(self, payload: dict[str, Any]) -> ChatMessageDTO:
        normalized = self._normalize_payload(payload)

        async with self._unit_of_work_factory() as unit_of_work:
            existing_message = await unit_of_work.messages.get_by_kick_message_id(
                normalized.kick_message_id
            )
            if existing_message is not None:
                return chat_message_to_dto(existing_message)

            channel = await unit_of_work.channels.get_by_chatroom_id(normalized.chatroom_id)
            if channel is None or channel.id is None:
                raise ChannelNotFoundError("Message channel is not followed.")

            sender = await self._upsert_sender(unit_of_work, normalized)
            if sender.id is None:
                raise MessageIngestionError("Sender could not be persisted.")

            message = await unit_of_work.messages.add(
                ChatMessage(
                    kick_message_id=normalized.kick_message_id,
                    channel_id=channel.id,
                    sender_id=sender.id,
                    chatroom_id=normalized.chatroom_id,
                    content=normalized.content,
                    message_type=normalized.message_type,
                    sender_username_snapshot=normalized.sender_username,
                    sender_slug_snapshot=normalized.sender_slug,
                    sender_color_snapshot=normalized.sender_color,
                    sender_badges=normalized.sender_badges,
                    emotes=self._emote_parser.parse(normalized.content),
                    reply_metadata=normalized.reply_metadata,
                    thread_parent_id=normalized.thread_parent_id,
                    raw_payload=payload,
                    message_created_at=normalized.message_created_at,
                )
            )
            await unit_of_work.commit()

        return chat_message_to_dto(message)

    async def _upsert_sender(
        self,
        unit_of_work: UnitOfWork,
        normalized: _NormalizedKickMessage,
    ) -> Sender:
        sender = await unit_of_work.senders.get_by_kick_user_id(normalized.sender_kick_user_id)
        if sender is None:
            sender = await unit_of_work.senders.get_by_slug(normalized.sender_slug)

        if sender is None:
            return await unit_of_work.senders.add(
                Sender(
                    kick_user_id=normalized.sender_kick_user_id,
                    username=normalized.sender_username,
                    slug=normalized.sender_slug,
                    profile_image_url=normalized.sender_profile_image_url,
                    last_seen_color=normalized.sender_color,
                    raw_profile_payload=normalized.sender_payload,
                )
            )

        sender.kick_user_id = normalized.sender_kick_user_id
        sender.username = normalized.sender_username
        sender.slug = normalized.sender_slug
        sender.profile_image_url = normalized.sender_profile_image_url
        sender.last_seen_color = normalized.sender_color
        sender.raw_profile_payload = normalized.sender_payload
        return await unit_of_work.senders.update(sender)

    def _normalize_payload(self, payload: dict[str, Any]) -> _NormalizedKickMessage:
        sender_payload = self._read_mapping(payload, "sender")
        identity_payload = sender_payload.get("identity")
        identity = identity_payload if isinstance(identity_payload, dict) else {}
        metadata = self._read_optional_mapping(payload.get("metadata"))

        sender_username = self._read_required_text(sender_payload, "username")
        sender_slug = self._normalize_slug(sender_payload.get("slug")) or sender_username.lower()

        return _NormalizedKickMessage(
            kick_message_id=self._read_required_text(payload, "id"),
            chatroom_id=self._read_required_int(payload, "chatroom_id"),
            content=str(payload.get("content") or ""),
            message_type=self._clean_text(payload.get("type")) or "message",
            message_created_at=self._parse_datetime(payload.get("created_at")),
            sender_kick_user_id=self._read_required_int(sender_payload, "id"),
            sender_username=sender_username,
            sender_slug=sender_slug,
            sender_color=self._clean_text(identity.get("color")),
            sender_badges=self._read_badges(identity.get("badges")),
            sender_profile_image_url=self._read_profile_image_url(sender_payload),
            sender_payload=sender_payload,
            reply_metadata=metadata,
            thread_parent_id=self._clean_text(
                payload.get("thread_parent_id") or metadata.get("thread_parent_id")
            ),
        )

    def _read_mapping(self, payload: dict[str, Any], key: str) -> dict[str, Any]:
        value = payload.get(key)
        if not isinstance(value, dict):
            raise MessageIngestionError(f"Message payload missing `{key}` object.")
        return value

    def _read_optional_mapping(self, value: Any) -> dict[str, Any]:
        return value if isinstance(value, dict) else {}

    def _read_required_text(self, payload: dict[str, Any], key: str) -> str:
        value = self._clean_text(payload.get(key))
        if value is None:
            raise MessageIngestionError(f"Message payload missing `{key}`.")
        return value

    def _read_required_int(self, payload: dict[str, Any], key: str) -> int:
        value = payload.get(key)
        if value is None:
            raise MessageIngestionError(f"Message payload missing `{key}`.")
        try:
            parsed = int(value)
        except (TypeError, ValueError) as exc:
            raise MessageIngestionError(f"Message payload has invalid `{key}`.") from exc
        if parsed < 1:
            raise MessageIngestionError(f"Message payload has invalid `{key}`.")
        return parsed

    def _parse_datetime(self, value: Any) -> datetime:
        if isinstance(value, datetime):
            parsed = value
        elif isinstance(value, str) and value.strip():
            try:
                parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
            except ValueError as exc:
                raise MessageIngestionError("Message payload has invalid `created_at`.") from exc
        else:
            parsed = datetime.now(UTC)

        if parsed.tzinfo is None:
            return parsed.replace(tzinfo=UTC)
        return parsed

    def _read_badges(self, value: Any) -> list[dict[str, Any]]:
        if not isinstance(value, list):
            return []
        return [badge for badge in value if isinstance(badge, dict)]

    def _read_profile_image_url(self, sender_payload: dict[str, Any]) -> str | None:
        for key in ("profile_image_url", "profile_pic", "profilepic"):
            value = self._clean_text(sender_payload.get(key))
            if value is not None:
                return value
        return None

    def _normalize_slug(self, value: Any) -> str | None:
        cleaned = self._clean_text(value)
        return cleaned.lower() if cleaned else None

    def _clean_text(self, value: Any) -> str | None:
        if value is None:
            return None
        cleaned = str(value).strip()
        return cleaned or None
