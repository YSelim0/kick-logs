"""Add data retention settings.

Revision ID: 20260515_0004
Revises: 20260513_0003
Create Date: 2026-05-15 00:00:00.000000

"""

from collections.abc import Sequence

import sqlalchemy as sa
from alembic import op

revision: str = "20260515_0004"
down_revision: str | None = "20260513_0003"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "data_retention_settings",
        sa.Column("id", sa.Integer(), nullable=False),
        sa.Column("message_retention_days", sa.Integer(), nullable=True),
        sa.Column("raw_event_retention_days", sa.Integer(), nullable=True),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            nullable=False,
        ),
        sa.CheckConstraint("id = 1", name="ck_data_retention_settings_singleton"),
        sa.CheckConstraint(
            "message_retention_days is null or message_retention_days in (30, 90)",
            name="ck_data_retention_settings_message_days",
        ),
        sa.CheckConstraint(
            "raw_event_retention_days is null or raw_event_retention_days in (30, 90)",
            name="ck_data_retention_settings_raw_event_days",
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    op.execute(
        "insert into data_retention_settings "
        "(id, message_retention_days, raw_event_retention_days) values (1, null, null)"
    )


def downgrade() -> None:
    op.drop_table("data_retention_settings")
