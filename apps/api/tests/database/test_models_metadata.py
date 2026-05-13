from sqlalchemy import BigInteger, DateTime
from sqlalchemy.dialects.postgresql import JSONB

from kick_logs.infrastructure.database import Base
from kick_logs.infrastructure.database.models import (
    ChannelModel,
    ChatMessageModel,
    RawKickEventModel,
    SenderModel,
    UserModel,
    WorkerHeartbeatModel,
)


def test_metadata_contains_core_tables() -> None:
    assert {
        "users",
        "channels",
        "senders",
        "chat_messages",
        "raw_kick_events",
        "worker_heartbeats",
    }.issubset(Base.metadata.tables)


def test_jsonb_payload_columns_are_present() -> None:
    assert isinstance(ChannelModel.__table__.c.raw_payload.type, JSONB)
    assert isinstance(SenderModel.__table__.c.raw_profile_payload.type, JSONB)
    assert isinstance(ChatMessageModel.__table__.c.sender_badges.type, JSONB)
    assert isinstance(ChatMessageModel.__table__.c.emotes.type, JSONB)
    assert isinstance(ChatMessageModel.__table__.c.reply_metadata.type, JSONB)
    assert isinstance(ChatMessageModel.__table__.c.raw_payload.type, JSONB)
    assert isinstance(RawKickEventModel.__table__.c.payload.type, JSONB)
    assert isinstance(RawKickEventModel.__table__.c.metadata.type, JSONB)
    assert isinstance(WorkerHeartbeatModel.__table__.c.metadata.type, JSONB)


def test_timestamp_columns_are_timezone_aware() -> None:
    for model in (UserModel, ChannelModel, SenderModel, RawKickEventModel, WorkerHeartbeatModel):
        assert isinstance(model.__table__.c.created_at.type, DateTime)
        assert model.__table__.c.created_at.type.timezone is True
        assert isinstance(model.__table__.c.updated_at.type, DateTime)
        assert model.__table__.c.updated_at.type.timezone is True

    assert isinstance(ChatMessageModel.__table__.c.message_created_at.type, DateTime)
    assert ChatMessageModel.__table__.c.message_created_at.type.timezone is True
    assert isinstance(RawKickEventModel.__table__.c.received_at.type, DateTime)
    assert RawKickEventModel.__table__.c.received_at.type.timezone is True
    assert isinstance(WorkerHeartbeatModel.__table__.c.last_seen_at.type, DateTime)
    assert WorkerHeartbeatModel.__table__.c.last_seen_at.type.timezone is True


def test_primary_keys_use_bigint_identity() -> None:
    for model in (UserModel, ChannelModel, SenderModel, ChatMessageModel, RawKickEventModel):
        assert isinstance(model.__table__.c.id.type, BigInteger)


def test_deduplication_constraints_are_named() -> None:
    constraint_names = {
        constraint.name
        for table in Base.metadata.tables.values()
        for constraint in table.constraints
        if constraint.name
    }

    assert "uq_users_email" in constraint_names
    assert "uq_channels_slug" in constraint_names
    assert "uq_senders_kick_user_id" in constraint_names
    assert "uq_chat_messages_kick_message_id" in constraint_names
