# Go Rewrite Phase 3: Storage Schema

## Scope

Add ClickHouse and SQLite storage foundations for the rewrite.

This phase owns schema migrations, database clients, repository foundations, and storage-level
tests. It does not need to expose full product APIs yet.

## Out Of Scope

- Do not implement full auth/admin API routes.
- Do not implement message search API behavior yet.
- Do not implement live Kick listener behavior yet.
- Do not migrate production data yet.

## Checklist

- [x] Add ClickHouse service to Compose for rewrite development.
- [x] Add SQLite database path configuration.
- [x] Add ClickHouse migration runner.
- [x] Add SQLite migration runner.
- [x] Create ClickHouse `chat_messages` table.
- [x] Create ClickHouse `raw_kick_events` table.
- [x] Create ClickHouse `raw_event_attempts` table.
- [x] Add ClickHouse helper columns needed for fast search: - normalized sender fields - normalized channel fields - lower-cased content helper - emote count - emote arrays - message type
- [x] Create SQLite `admin_users` table.
- [x] Create SQLite `followed_channels` table.
- [x] Create SQLite `sender_profiles` table.
- [x] Create SQLite `retention_settings` table.
- [x] Create SQLite `worker_heartbeats` table.
- [x] Create SQLite migration bookkeeping tables.
- [x] Add repository interfaces and initial concrete storage implementations.
- [x] Add seed logic for the default super admin in SQLite.
- [x] Add database size/table size query support where available.

## Tests And Checks

- [x] SQLite migrations apply from an empty database.
- [x] ClickHouse migrations apply from an empty database.
- [x] Migrations are idempotent or safely versioned.
- [x] Repository tests can insert and fetch admin users.
- [x] Repository tests can insert and fetch followed channels.
- [x] Repository tests can insert and search basic message rows.
- [x] Repository tests can insert raw events and processing attempts.

## Acceptance Criteria

- [x] A fresh local environment can create both stores without manual SQL.
- [x] Storage fields can represent every current API response field.
- [x] Message rows are denormalized enough for search responses without per-row cross-store joins.
- [x] Default super admin seeding works against SQLite.

## Commit Boundary

Commit this phase after migrations and storage tests pass.
