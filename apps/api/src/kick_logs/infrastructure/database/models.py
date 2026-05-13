from datetime import datetime
from typing import Any

from sqlalchemy import (
    BigInteger,
    Boolean,
    DateTime,
    ForeignKey,
    Identity,
    Index,
    Integer,
    String,
    Text,
    UniqueConstraint,
    func,
    text,
)
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.orm import Mapped, mapped_column, relationship

from kick_logs.infrastructure.database.base import Base


class TimestampMixin:
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        server_default=func.now(),
        nullable=False,
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        server_default=func.now(),
        onupdate=func.now(),
        nullable=False,
    )


class UserModel(TimestampMixin, Base):
    __tablename__ = "users"
    __table_args__ = (UniqueConstraint("email", name="uq_users_email"),)

    id: Mapped[int] = mapped_column(BigInteger, Identity(), primary_key=True)
    email: Mapped[str] = mapped_column(String(320), index=True, nullable=False)
    password_hash: Mapped[str] = mapped_column(String(255), nullable=False)
    role: Mapped[str] = mapped_column(String(32), nullable=False)
    is_active: Mapped[bool] = mapped_column(
        Boolean,
        default=True,
        server_default="true",
        nullable=False,
    )


class ChannelModel(TimestampMixin, Base):
    __tablename__ = "channels"
    __table_args__ = (
        UniqueConstraint("kick_channel_id", name="uq_channels_kick_channel_id"),
        UniqueConstraint("kick_chatroom_id", name="uq_channels_kick_chatroom_id"),
        UniqueConstraint("slug", name="uq_channels_slug"),
    )

    id: Mapped[int] = mapped_column(BigInteger, Identity(), primary_key=True)
    kick_channel_id: Mapped[int | None] = mapped_column(BigInteger)
    kick_chatroom_id: Mapped[int | None] = mapped_column(BigInteger)
    slug: Mapped[str] = mapped_column(String(120), index=True, nullable=False)
    display_name: Mapped[str] = mapped_column(String(160), nullable=False)
    profile_image_url: Mapped[str | None] = mapped_column(String(1000))
    banner_image_url: Mapped[str | None] = mapped_column(String(1000))
    is_enabled: Mapped[bool] = mapped_column(
        Boolean,
        default=True,
        server_default="true",
        nullable=False,
    )
    raw_payload: Mapped[dict[str, Any]] = mapped_column(JSONB, default=dict, nullable=False)

    messages: Mapped[list["ChatMessageModel"]] = relationship(back_populates="channel")


class SenderModel(TimestampMixin, Base):
    __tablename__ = "senders"
    __table_args__ = (
        UniqueConstraint("kick_user_id", name="uq_senders_kick_user_id"),
        UniqueConstraint("slug", name="uq_senders_slug"),
    )

    id: Mapped[int] = mapped_column(BigInteger, Identity(), primary_key=True)
    kick_user_id: Mapped[int] = mapped_column(BigInteger, nullable=False)
    username: Mapped[str] = mapped_column(String(160), index=True, nullable=False)
    slug: Mapped[str] = mapped_column(String(160), index=True, nullable=False)
    profile_image_url: Mapped[str | None] = mapped_column(String(1000))
    last_seen_color: Mapped[str | None] = mapped_column(String(32))
    raw_profile_payload: Mapped[dict[str, Any]] = mapped_column(JSONB, default=dict, nullable=False)

    messages: Mapped[list["ChatMessageModel"]] = relationship(back_populates="sender")


class ChatMessageModel(Base):
    __tablename__ = "chat_messages"
    __table_args__ = (
        UniqueConstraint("kick_message_id", name="uq_chat_messages_kick_message_id"),
        Index("ix_chat_messages_message_created_at", "message_created_at"),
        Index("ix_chat_messages_message_created_at_id", "message_created_at", "id"),
        Index("ix_chat_messages_channel_id", "channel_id"),
        Index("ix_chat_messages_sender_id", "sender_id"),
        Index("ix_chat_messages_chatroom_id", "chatroom_id"),
    )

    id: Mapped[int] = mapped_column(BigInteger, Identity(), primary_key=True)
    kick_message_id: Mapped[str] = mapped_column(String(160), nullable=False)
    channel_id: Mapped[int] = mapped_column(ForeignKey("channels.id"), nullable=False)
    sender_id: Mapped[int] = mapped_column(ForeignKey("senders.id"), nullable=False)
    chatroom_id: Mapped[int] = mapped_column(BigInteger, nullable=False)
    content: Mapped[str] = mapped_column(Text, nullable=False)
    message_type: Mapped[str] = mapped_column(String(64), nullable=False)
    sender_username_snapshot: Mapped[str] = mapped_column(String(160), nullable=False)
    sender_slug_snapshot: Mapped[str] = mapped_column(String(160), nullable=False)
    sender_color_snapshot: Mapped[str | None] = mapped_column(String(32))
    sender_badges: Mapped[list[dict[str, Any]]] = mapped_column(JSONB, default=list, nullable=False)
    emotes: Mapped[list[dict[str, Any]]] = mapped_column(JSONB, default=list, nullable=False)
    reply_metadata: Mapped[dict[str, Any]] = mapped_column(JSONB, default=dict, nullable=False)
    thread_parent_id: Mapped[str | None] = mapped_column(String(160))
    raw_payload: Mapped[dict[str, Any]] = mapped_column(JSONB, default=dict, nullable=False)
    message_created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    ingested_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        server_default=func.now(),
        nullable=False,
    )

    channel: Mapped[ChannelModel] = relationship(back_populates="messages")
    sender: Mapped[SenderModel] = relationship(back_populates="messages")


class RawKickEventModel(TimestampMixin, Base):
    __tablename__ = "raw_kick_events"
    __table_args__ = (
        Index(
            "uq_raw_kick_events_kick_message_id_present",
            "kick_message_id",
            unique=True,
            postgresql_where=text("kick_message_id IS NOT NULL"),
        ),
        Index("ix_raw_kick_events_status_received_at", "status", "received_at"),
        Index("ix_raw_kick_events_chatroom_id", "chatroom_id"),
        Index("ix_raw_kick_events_received_at", "received_at"),
    )

    id: Mapped[int] = mapped_column(BigInteger, Identity(), primary_key=True)
    event_name: Mapped[str] = mapped_column(String(255), nullable=False)
    kick_message_id: Mapped[str | None] = mapped_column(String(160))
    chatroom_id: Mapped[int | None] = mapped_column(BigInteger)
    kick_channel_id: Mapped[int | None] = mapped_column(BigInteger)
    channel_id: Mapped[int | None] = mapped_column(ForeignKey("channels.id"))
    payload: Mapped[dict[str, Any]] = mapped_column(JSONB, nullable=False)
    status: Mapped[str] = mapped_column(String(32), nullable=False)
    attempts: Mapped[int] = mapped_column(Integer, default=0, server_default="0", nullable=False)
    received_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True),
        server_default=func.now(),
        nullable=False,
    )
    processing_started_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    processed_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True))
    last_error: Mapped[str | None] = mapped_column(Text)
    event_metadata: Mapped[dict[str, Any]] = mapped_column(
        "metadata",
        JSONB,
        default=dict,
        server_default=text("'{}'::jsonb"),
        nullable=False,
    )


class WorkerHeartbeatModel(TimestampMixin, Base):
    __tablename__ = "worker_heartbeats"
    __table_args__ = (Index("ix_worker_heartbeats_last_seen_at", "last_seen_at"),)

    service_name: Mapped[str] = mapped_column(String(80), primary_key=True)
    last_seen_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False)
    heartbeat_metadata: Mapped[dict[str, Any]] = mapped_column(
        "metadata",
        JSONB,
        default=dict,
        server_default=text("'{}'::jsonb"),
        nullable=False,
    )
