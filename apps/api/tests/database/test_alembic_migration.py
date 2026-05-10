from pathlib import Path


def test_initial_migration_contains_required_postgres_features() -> None:
    migration = Path("alembic/versions/20260510_0001_initial_schema.py").read_text(encoding="utf-8")

    assert "CREATE EXTENSION IF NOT EXISTS pg_trgm" in migration
    assert "postgresql.JSONB" in migration
    assert "uq_chat_messages_kick_message_id" in migration
    assert "ix_chat_messages_content_trgm" in migration
