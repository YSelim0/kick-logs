# Go Rewrite Phase 8: PostgreSQL Data Migration

## Scope

Build the one-time idempotent migrator from the existing PostgreSQL database to ClickHouse and
SQLite.

This phase owns migration commands, validation, and operational documentation for preserving
existing data.

## Out Of Scope

- Do not delete PostgreSQL data.
- Do not remove Python services.
- Do not switch the default runtime yet.
- Do not change frontend behavior.

## Checklist

- [x] Add PostgreSQL source connection configuration for the migrator.
- [x] Add migration command flags for dry-run, execute, batch size, and validation-only mode.
- [x] Migrate users to SQLite `admin_users`.
- [x] Verify existing password hashes are Go-compatible before accepting migration.
- [x] Migrate channels to SQLite `followed_channels`.
- [x] Migrate senders to SQLite `sender_profiles`.
- [x] Migrate retention settings to SQLite.
- [x] Migrate useful heartbeat state to SQLite if present.
- [x] Migrate chat messages to ClickHouse `chat_messages`.
- [x] Migrate raw Kick events to ClickHouse `raw_kick_events`.
- [x] Map raw event status/attempt fields into ClickHouse `raw_event_attempts`.
- [x] Serialize JSONB payloads into valid JSON strings.
- [x] Normalize timestamps to UTC.
- [x] Preserve source IDs where current API exposes IDs.
- [x] Make the migration safe to rerun.
- [x] Record migration run metadata.
- [x] Add count validation.
- [x] Add representative sample validation by ID and `kick_message_id`.
- [x] Add clear failure messages for unsupported source data.

## Tests And Checks

- [x] Migration test with empty source database.
- [x] Migration test with representative users, channels, senders, messages, raw events, replies,
      and emotes.
- [x] Migration rerun test proves idempotency.
- [x] Validation fails when destination counts or samples do not match.
- [x] Migrated search results match equivalent source fixture behavior.
- [x] Migrated admin login works with existing admin hash.

## Acceptance Criteria

- [x] Existing local PostgreSQL data can be copied into ClickHouse and SQLite without manual SQL.
- [x] Migration can be rerun safely.
- [x] The migrator reports enough validation detail to decide whether cutover is safe.
- [x] PostgreSQL remains untouched after migration.

## Commit Boundary

Commit migration tooling after tests prove idempotency and validation behavior.
