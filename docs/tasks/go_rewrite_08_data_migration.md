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

- [ ] Add PostgreSQL source connection configuration for the migrator.
- [ ] Add migration command flags for dry-run, execute, batch size, and validation-only mode.
- [ ] Migrate users to SQLite `admin_users`.
- [ ] Verify existing password hashes are Go-compatible before accepting migration.
- [ ] Migrate channels to SQLite `followed_channels`.
- [ ] Migrate senders to SQLite `sender_profiles`.
- [ ] Migrate retention settings to SQLite.
- [ ] Migrate useful heartbeat state to SQLite if present.
- [ ] Migrate chat messages to ClickHouse `chat_messages`.
- [ ] Migrate raw Kick events to ClickHouse `raw_kick_events`.
- [ ] Map raw event status/attempt fields into ClickHouse `raw_event_attempts`.
- [ ] Serialize JSONB payloads into valid JSON strings.
- [ ] Normalize timestamps to UTC.
- [ ] Preserve source IDs where current API exposes IDs.
- [ ] Make the migration safe to rerun.
- [ ] Record migration run metadata.
- [ ] Add count validation.
- [ ] Add representative sample validation by ID and `kick_message_id`.
- [ ] Add clear failure messages for unsupported source data.

## Tests And Checks

- [ ] Migration test with empty source database.
- [ ] Migration test with representative users, channels, senders, messages, raw events, replies,
      and emotes.
- [ ] Migration rerun test proves idempotency.
- [ ] Validation fails when destination counts or samples do not match.
- [ ] Migrated search results match equivalent source fixture behavior.
- [ ] Migrated admin login works with existing admin hash.

## Acceptance Criteria

- [ ] Existing local PostgreSQL data can be copied into ClickHouse and SQLite without manual SQL.
- [ ] Migration can be rerun safely.
- [ ] The migrator reports enough validation detail to decide whether cutover is safe.
- [ ] PostgreSQL remains untouched after migration.

## Commit Boundary

Commit migration tooling after tests prove idempotency and validation behavior.
