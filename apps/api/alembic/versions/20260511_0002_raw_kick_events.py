"""Add durable raw Kick event inbox.

Revision ID: 20260511_0002
Revises: 20260510_0001
Create Date: 2026-05-11 00:00:00
"""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op
from sqlalchemy.dialects import postgresql

revision: str = "20260511_0002"
down_revision: str | None = "20260510_0001"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "raw_kick_events",
        sa.Column("id", sa.BigInteger(), sa.Identity(), primary_key=True),
        sa.Column("event_name", sa.String(length=255), nullable=False),
        sa.Column("kick_message_id", sa.String(length=160), nullable=True),
        sa.Column("chatroom_id", sa.BigInteger(), nullable=True),
        sa.Column("kick_channel_id", sa.BigInteger(), nullable=True),
        sa.Column("channel_id", sa.BigInteger(), nullable=True),
        sa.Column(
            "payload",
            postgresql.JSONB(astext_type=sa.Text()),
            nullable=False,
        ),
        sa.Column("status", sa.String(length=32), nullable=False),
        sa.Column("attempts", sa.Integer(), nullable=False, server_default="0"),
        sa.Column(
            "received_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.func.now(),
        ),
        sa.Column("processing_started_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("processed_at", sa.DateTime(timezone=True), nullable=True),
        sa.Column("last_error", sa.Text(), nullable=True),
        sa.Column(
            "metadata",
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
        sa.ForeignKeyConstraint(
            ["channel_id"],
            ["channels.id"],
            name="fk_raw_kick_events_channel_id",
        ),
    )
    op.create_index(
        "uq_raw_kick_events_kick_message_id_present",
        "raw_kick_events",
        ["kick_message_id"],
        unique=True,
        postgresql_where=sa.text("kick_message_id IS NOT NULL"),
    )
    op.create_index(
        "ix_raw_kick_events_status_received_at",
        "raw_kick_events",
        ["status", "received_at"],
    )
    op.create_index("ix_raw_kick_events_chatroom_id", "raw_kick_events", ["chatroom_id"])
    op.create_index("ix_raw_kick_events_received_at", "raw_kick_events", ["received_at"])


def downgrade() -> None:
    op.drop_index("ix_raw_kick_events_received_at", table_name="raw_kick_events")
    op.drop_index("ix_raw_kick_events_chatroom_id", table_name="raw_kick_events")
    op.drop_index("ix_raw_kick_events_status_received_at", table_name="raw_kick_events")
    op.drop_index("uq_raw_kick_events_kick_message_id_present", table_name="raw_kick_events")
    op.drop_table("raw_kick_events")
