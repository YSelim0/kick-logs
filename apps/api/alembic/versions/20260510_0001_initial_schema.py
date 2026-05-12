"""Create initial persistence schema.

Revision ID: 20260510_0001
Revises:
Create Date: 2026-05-10 00:00:00
"""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op
from sqlalchemy.dialects import postgresql

revision: str = "20260510_0001"
down_revision: str | None = None
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.execute("CREATE EXTENSION IF NOT EXISTS pg_trgm")

    op.create_table(
        "users",
        sa.Column("id", sa.BigInteger(), sa.Identity(), primary_key=True),
        sa.Column("email", sa.String(length=320), nullable=False),
        sa.Column("password_hash", sa.String(length=255), nullable=False),
        sa.Column("role", sa.String(length=32), nullable=False),
        sa.Column("is_active", sa.Boolean(), nullable=False, server_default=sa.true()),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.func.now(),
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.func.now(),
        ),
        sa.UniqueConstraint("email", name="uq_users_email"),
    )

    op.create_table(
        "channels",
        sa.Column("id", sa.BigInteger(), sa.Identity(), primary_key=True),
        sa.Column("kick_channel_id", sa.BigInteger(), nullable=True),
        sa.Column("kick_chatroom_id", sa.BigInteger(), nullable=True),
        sa.Column("slug", sa.String(length=120), nullable=False),
        sa.Column("display_name", sa.String(length=160), nullable=False),
        sa.Column("profile_image_url", sa.String(length=1000), nullable=True),
        sa.Column("banner_image_url", sa.String(length=1000), nullable=True),
        sa.Column("is_enabled", sa.Boolean(), nullable=False, server_default=sa.true()),
        sa.Column(
            "raw_payload",
            postgresql.JSONB(astext_type=sa.Text()),
            nullable=False,
            server_default=sa.text("'{}'::jsonb"),
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.func.now(),
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.func.now(),
        ),
        sa.UniqueConstraint("kick_channel_id", name="uq_channels_kick_channel_id"),
        sa.UniqueConstraint("kick_chatroom_id", name="uq_channels_kick_chatroom_id"),
        sa.UniqueConstraint("slug", name="uq_channels_slug"),
    )

    op.create_table(
        "senders",
        sa.Column("id", sa.BigInteger(), sa.Identity(), primary_key=True),
        sa.Column("kick_user_id", sa.BigInteger(), nullable=False),
        sa.Column("username", sa.String(length=160), nullable=False),
        sa.Column("slug", sa.String(length=160), nullable=False),
        sa.Column("profile_image_url", sa.String(length=1000), nullable=True),
        sa.Column("last_seen_color", sa.String(length=32), nullable=True),
        sa.Column(
            "raw_profile_payload",
            postgresql.JSONB(astext_type=sa.Text()),
            nullable=False,
            server_default=sa.text("'{}'::jsonb"),
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.func.now(),
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.func.now(),
        ),
        sa.UniqueConstraint("kick_user_id", name="uq_senders_kick_user_id"),
        sa.UniqueConstraint("slug", name="uq_senders_slug"),
    )

    op.create_table(
        "chat_messages",
        sa.Column("id", sa.BigInteger(), sa.Identity(), primary_key=True),
        sa.Column("kick_message_id", sa.String(length=160), nullable=False),
        sa.Column("channel_id", sa.BigInteger(), nullable=False),
        sa.Column("sender_id", sa.BigInteger(), nullable=False),
        sa.Column("chatroom_id", sa.BigInteger(), nullable=False),
        sa.Column("content", sa.Text(), nullable=False),
        sa.Column("message_type", sa.String(length=64), nullable=False),
        sa.Column("sender_username_snapshot", sa.String(length=160), nullable=False),
        sa.Column("sender_slug_snapshot", sa.String(length=160), nullable=False),
        sa.Column("sender_color_snapshot", sa.String(length=32), nullable=True),
        sa.Column(
            "sender_badges",
            postgresql.JSONB(astext_type=sa.Text()),
            nullable=False,
            server_default=sa.text("'[]'::jsonb"),
        ),
        sa.Column(
            "emotes",
            postgresql.JSONB(astext_type=sa.Text()),
            nullable=False,
            server_default=sa.text("'[]'::jsonb"),
        ),
        sa.Column(
            "reply_metadata",
            postgresql.JSONB(astext_type=sa.Text()),
            nullable=False,
            server_default=sa.text("'{}'::jsonb"),
        ),
        sa.Column("thread_parent_id", sa.String(length=160), nullable=True),
        sa.Column(
            "raw_payload",
            postgresql.JSONB(astext_type=sa.Text()),
            nullable=False,
            server_default=sa.text("'{}'::jsonb"),
        ),
        sa.Column("message_created_at", sa.DateTime(timezone=True), nullable=False),
        sa.Column(
            "ingested_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.func.now(),
        ),
        sa.ForeignKeyConstraint(
            ["channel_id"],
            ["channels.id"],
            name="fk_chat_messages_channel_id",
        ),
        sa.ForeignKeyConstraint(["sender_id"], ["senders.id"], name="fk_chat_messages_sender_id"),
        sa.UniqueConstraint("kick_message_id", name="uq_chat_messages_kick_message_id"),
    )

    op.create_index("ix_users_email", "users", ["email"])
    op.create_index("ix_channels_slug", "channels", ["slug"])
    op.create_index("ix_senders_username", "senders", ["username"])
    op.create_index("ix_senders_slug", "senders", ["slug"])
    op.create_index("ix_chat_messages_message_created_at", "chat_messages", ["message_created_at"])
    op.create_index(
        "ix_chat_messages_message_created_at_id",
        "chat_messages",
        [sa.text("message_created_at DESC"), sa.text("id DESC")],
    )
    op.create_index("ix_chat_messages_channel_id", "chat_messages", ["channel_id"])
    op.create_index("ix_chat_messages_sender_id", "chat_messages", ["sender_id"])
    op.create_index("ix_chat_messages_chatroom_id", "chat_messages", ["chatroom_id"])

    op.execute(
        "CREATE INDEX ix_channels_slug_trgm ON channels USING gin (lower(slug) gin_trgm_ops)"
    )
    op.execute(
        "CREATE INDEX ix_channels_display_name_trgm "
        "ON channels USING gin (lower(display_name) gin_trgm_ops)"
    )
    op.execute(
        "CREATE INDEX ix_senders_username_trgm ON senders USING gin (lower(username) gin_trgm_ops)"
    )
    op.execute("CREATE INDEX ix_senders_slug_trgm ON senders USING gin (lower(slug) gin_trgm_ops)")
    op.execute(
        "CREATE INDEX ix_chat_messages_content_trgm "
        "ON chat_messages USING gin (lower(content) gin_trgm_ops)"
    )


def downgrade() -> None:
    op.execute("DROP INDEX IF EXISTS ix_chat_messages_content_trgm")
    op.execute("DROP INDEX IF EXISTS ix_senders_slug_trgm")
    op.execute("DROP INDEX IF EXISTS ix_senders_username_trgm")
    op.execute("DROP INDEX IF EXISTS ix_channels_display_name_trgm")
    op.execute("DROP INDEX IF EXISTS ix_channels_slug_trgm")
    op.drop_table("chat_messages")
    op.drop_table("senders")
    op.drop_table("channels")
    op.drop_table("users")
    op.execute("DROP EXTENSION IF EXISTS pg_trgm")
