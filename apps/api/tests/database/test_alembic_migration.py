from pathlib import Path


def test_initial_migration_contains_required_postgres_features() -> None:
    migration = Path("alembic/versions/20260510_0001_initial_schema.py").read_text(encoding="utf-8")

    assert "CREATE EXTENSION IF NOT EXISTS pg_trgm" in migration
    assert "postgresql.JSONB" in migration
    assert "uq_chat_messages_kick_message_id" in migration
    assert "ix_chat_messages_content_trgm" in migration


def test_raw_event_migration_contains_durable_inbox_indexes() -> None:
    migration = Path("alembic/versions/20260511_0002_raw_kick_events.py").read_text(
        encoding="utf-8"
    )

    assert "raw_kick_events" in migration
    assert "postgresql.JSONB" in migration
    assert "uq_raw_kick_events_kick_message_id_present" in migration
    assert "ix_raw_kick_events_status_received_at" in migration
