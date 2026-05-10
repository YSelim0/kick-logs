from datetime import UTC, datetime

import pytest

from kick_logs.domain.entities import Channel, ChatMessage, Emote, Sender, User
from kick_logs.domain.exceptions import DomainError
from kick_logs.domain.value_objects.roles import UserRole


def test_user_normalizes_email() -> None:
    user = User(email=" Admin@Example.COM ", password_hash="hash", role=UserRole.SUPER_ADMIN)

    assert user.email == "admin@example.com"


def test_channel_normalizes_slug() -> None:
    channel = Channel(slug=" Hype ", display_name="Hype")

    assert channel.slug == "hype"
    assert channel.is_enabled is True


def test_emote_builds_image_url() -> None:
    emote = Emote(kick_emote_id="37226", name="KEKW", token="[emote:37226:KEKW]")

    assert emote.image_url == "https://files.kick.com/emotes/37226/fullsize"
    assert emote.to_dict()["name"] == "KEKW"


def test_sender_requires_positive_kick_user_id() -> None:
    with pytest.raises(DomainError):
        Sender(kick_user_id=0, username="Yavuz", slug="yavuz")


def test_chat_message_requires_timezone_aware_created_at() -> None:
    with pytest.raises(DomainError):
        ChatMessage(
            kick_message_id="message-1",
            channel_id=1,
            sender_id=1,
            chatroom_id=10,
            content="hello",
            message_type="message",
            sender_username_snapshot="Yavuz",
            sender_slug_snapshot="yavuz",
            message_created_at=datetime(2026, 5, 10, 12, 0, 0),
        )


def test_chat_message_accepts_valid_payload() -> None:
    message = ChatMessage(
        kick_message_id="message-1",
        channel_id=1,
        sender_id=1,
        chatroom_id=10,
        content="hello",
        message_type="message",
        sender_username_snapshot="Yavuz",
        sender_slug_snapshot="yavuz",
        message_created_at=datetime(2026, 5, 10, 12, 0, 0, tzinfo=UTC),
    )

    assert message.kick_message_id == "message-1"
