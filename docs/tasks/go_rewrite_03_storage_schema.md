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

- [ ] Add ClickHouse service to Compose for rewrite development.
- [ ] Add SQLite database path configuration.
- [ ] Add ClickHouse migration runner.
- [ ] Add SQLite migration runner.
- [ ] Create ClickHouse `chat_messages` table.
- [ ] Create ClickHouse `raw_kick_events` table.
- [ ] Create ClickHouse `raw_event_attempts` table.
- [ ] Add ClickHouse helper columns needed for fast search: - normalized sender fields - normalized channel fields - lower-cased content helper - emote count - emote arrays - message type
- [ ] Create SQLite `admin_users` table.
- [ ] Create SQLite `followed_channels` table.
- [ ] Create SQLite `sender_profiles` table.
- [ ] Create SQLite `retention_settings` table.
- [ ] Create SQLite `worker_heartbeats` table.
- [ ] Create SQLite migration bookkeeping tables.
- [ ] Add repository interfaces and initial concrete storage implementations.
- [ ] Add seed logic for the default super admin in SQLite.
- [ ] Add database size/table size query support where available.

## Tests And Checks

- [ ] SQLite migrations apply from an empty database.
- [ ] ClickHouse migrations apply from an empty database.
- [ ] Migrations are idempotent or safely versioned.
- [ ] Repository tests can insert and fetch admin users.
- [ ] Repository tests can insert and fetch followed channels.
- [ ] Repository tests can insert and search basic message rows.
- [ ] Repository tests can insert raw events and processing attempts.

## Acceptance Criteria

- [ ] A fresh local environment can create both stores without manual SQL.
- [ ] Storage fields can represent every current API response field.
- [ ] Message rows are denormalized enough for search responses without per-row cross-store joins.
- [ ] Default super admin seeding works against SQLite.

## Commit Boundary

Commit this phase after migrations and storage tests pass.
