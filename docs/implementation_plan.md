# Kick Logs Go + ClickHouse Rewrite Implementation Plan

This is the active implementation plan for rewriting the backend and listener in Go while moving
message, raw-event, and analytics storage to ClickHouse.

The completed Python/FastAPI/PostgreSQL plans are archived under `docs/archive/`. They remain useful
as product and contract history, but they are no longer active implementation scope for this branch.

Status: planning started on branch `feat/go-clickhouse-rewrite`.

## Primary Goal

Rebuild the backend and listener in Go without breaking the current frontend or public API
contracts.

The rewrite must preserve:

- endpoint paths
- query parameter names and behavior
- request body fields
- response body fields
- status codes used by the frontend
- auth cookie behavior
- public versus admin-only access boundaries
- current search, analytics, profile, export, and data-management behavior

The frontend should continue to work against the Go API with minimal or no feature-level changes.

## Storage Decision

The rewrite uses two local stores:

- ClickHouse for data-plane workloads:
  - chat messages
  - raw Kick events
  - raw-event processing history
  - message search and export
  - analytics and profile aggregations
- SQLite for control-plane workloads:
  - admin users
  - followed channel configuration
  - sender profile cache
  - retention settings
  - worker heartbeat state
  - migration bookkeeping

This keeps ClickHouse focused on append-heavy log and analytics work while avoiding the need to use
ClickHouse as a transactional admin/auth database. SQLite does not add another database server to
the self-hosted runtime.

## Target Runtime

The final Compose runtime should contain:

- `clickhouse`: ClickHouse database service.
- `api`: Go HTTP API service.
- `listener`: Go Kick listener service.
- `web`: existing Next.js frontend service.

During rewrite and parity work, the Python backend and PostgreSQL service may remain available as a
reference runtime. PostgreSQL is removed from the default runtime only after migration and contract
parity are accepted.

## Target Backend Layout

Use a Go workspace under `apps/api-go`.

```text
apps/api-go/
  go.mod
  go.sum
  cmd/
    api/
      main.go
    listener/
      main.go
    migrate/
      main.go
  internal/
    app/
      bootstrap.go
    config/
      config.go
    domain/
      auth.go
      channel.go
      message.go
      raw_event.go
      sender.go
    ports/
      auth_repository.go
      channel_repository.go
      message_repository.go
      analytics_repository.go
      data_management_repository.go
      kick_resolver.go
      token_service.go
    usecase/
      auth/
      channels/
      messages/
      listener/
      analytics/
      profiles/
      data_management/
      operations/
    infra/
      clickhouse/
      sqlite/
      kick/
      auth/
      migrations/
    http/
      middleware/
      routes/
      schemas/
      server.go
    worker/
      listener_service.go
  tests/
    contract/
    fixtures/
```

Dependency direction:

```text
http -> usecase -> domain
infra -> ports/domain
worker -> usecase
cmd -> app/config/http/worker
```

Rules:

- Domain packages do not import HTTP, ClickHouse, SQLite, JWT, websocket, or framework packages.
- Use cases depend on ports/interfaces, not concrete database implementations.
- HTTP schemas are explicit structs that match existing JSON contracts.
- Infrastructure packages own SQL, external Kick calls, JWT implementation, password hashing, and
  websocket clients.

## API Contract Inventory

The Go API must preserve the current backend surface:

```text
GET  /health

POST /auth/login
POST /auth/logout
GET  /auth/me

GET  /messages
GET  /messages/export

GET    /admin/channels
POST   /admin/channels
DELETE /admin/channels/{channel_id}

GET  /admin/users
POST /admin/users

GET /admin/operations/summary

GET  /admin/data-management/summary
PUT  /admin/data-management/retention-settings
POST /admin/data-management/cleanup/preview
POST /admin/data-management/cleanup/confirm

GET /analytics/overview
GET /analytics/message-volume
GET /analytics/top-senders
GET /analytics/top-channels
GET /analytics/top-emotes

GET /users/{slug}/analytics
GET /channels/{slug}/analytics
```

Compatibility requirements:

- `POST /auth/login` sets the same HttpOnly cookie name and returns the same user shape.
- `POST /auth/logout` deletes the same cookie path.
- Admin routes require a valid admin or super admin session.
- Super admin-only behavior remains required for admin user creation.
- Public search, analytics, user profile, and channel profile routes remain unauthenticated.
- CORS behavior remains compatible with the current Next.js dev server and Docker runtime.
- Validation failures should keep the same practical frontend behavior even if exact error text
  differs where the frontend does not depend on it.

## Search Contract

The Go implementation must preserve current search behavior:

- All filters are optional.
- Non-empty filters combine with `AND`.
- Empty filters are allowed and return latest messages.
- `sender` uses case-insensitive exact matching against sender username and sender slug snapshots.
- `channel` matches channel slug/display fields with the current behavior.
- `q` matches message content with the current behavior.
- `start` and `end` filter by message creation time.
- `reply_only=true` returns reply messages only.
- `emote_only=true` returns messages with at least one parsed emote.
- Results are ordered newest-first.
- Cursor pagination uses the current text cursor shape: `message_created_at|message_id`.
- Export supports JSON and CSV and clamps to `MESSAGE_EXPORT_MAX_ROWS`.

ClickHouse query implementation should use stored helper columns for performance:

- normalized sender username and slug
- normalized channel slug and display name
- emote count
- reply flag or message type
- message creation timestamp
- lower-cased content helper if needed for contains search

## ClickHouse Schema Direction

Use migrations to create ClickHouse tables. The exact SQL can evolve during implementation, but the
schema must support the existing API without expensive cross-store joins on the hot search path.

### `chat_messages`

Purpose: durable normalized message store for search, export, analytics, and profile pages.

Required fields:

- `id Int64`
- `kick_message_id String`
- `chatroom_id Int64`
- `channel_id Int64`
- `channel_slug String`
- `channel_display_name String`
- `channel_profile_image_url Nullable(String)`
- `channel_banner_image_url Nullable(String)`
- `sender_id Int64`
- `sender_kick_user_id Int64`
- `sender_username String`
- `sender_slug String`
- `sender_profile_image_url Nullable(String)`
- `content String`
- `content_lower String`
- `message_type LowCardinality(String)`
- `sender_color Nullable(String)`
- `sender_badges_json String`
- `emotes_json String`
- `emote_ids Array(String)`
- `emote_names Array(String)`
- `emote_tokens Array(String)`
- `emote_image_urls Array(String)`
- `emote_count UInt16`
- `reply_metadata_json String`
- `thread_parent_id Nullable(String)`
- `raw_payload_json String`
- `message_created_at DateTime64(3, 'UTC')`
- `ingested_at DateTime64(3, 'UTC')`

Engine direction:

- `ReplacingMergeTree(ingested_at)`
- partition by message month
- order by `(message_created_at, kick_message_id)`
- use deterministic id generation for new rows where practical
- Go ingestion remains idempotent by checking `kick_message_id` before insert and relying on
  ReplacingMergeTree as a second layer of duplicate tolerance

### `raw_kick_events`

Purpose: store received websocket events before normalization so received events are not lost.

Required fields:

- `id Int64`
- `event_name String`
- `kick_message_id Nullable(String)`
- `chatroom_id Nullable(Int64)`
- `kick_channel_id Nullable(Int64)`
- `channel_id Nullable(Int64)`
- `payload_json String`
- `metadata_json String`
- `received_at DateTime64(3, 'UTC')`

Engine direction:

- append-focused MergeTree
- partition by receive month
- order by `(received_at, id)`

### `raw_event_attempts`

Purpose: append processing attempts instead of mutating the raw event row.

Required fields:

- `raw_event_id Int64`
- `kick_message_id Nullable(String)`
- `status LowCardinality(String)`
- `attempt Int32`
- `started_at DateTime64(3, 'UTC')`
- `finished_at Nullable(DateTime64(3, 'UTC'))`
- `last_error Nullable(String)`

Statuses:

- `processing`
- `processed`
- `failed`

Operations summary can derive pending/processed/failed counts from raw events, chat messages, and
latest attempt state.

## SQLite Schema Direction

Use migrations to create SQLite tables.

### `admin_users`

Stores:

- id
- email
- password_hash
- role
- is_active
- created_at
- updated_at

Rules:

- Preserve existing default super admin behavior.
- Preserve migrated password hashes if they are bcrypt-compatible.
- If a migrated hash is not Go-compatible, document and fail migration with a clear error instead
  of silently resetting credentials.

### `followed_channels`

Stores:

- id
- kick_channel_id
- kick_chatroom_id
- slug
- display_name
- profile_image_url
- banner_image_url
- is_enabled
- raw_payload_json
- created_at
- updated_at

Rules:

- Admin channel add resolves Kick metadata before persisting.
- Re-adding a disabled channel enables it again.
- Delete route disables the channel, preserving historical message data.
- Listener periodically reloads enabled channels and resubscribes when the set changes.

### `sender_profiles`

Stores:

- id
- kick_user_id
- username
- slug
- profile_image_url
- last_seen_color
- raw_profile_payload_json
- created_at
- updated_at

Rules:

- Listener updates sender profile cache when richer metadata is available.
- Message rows still store sender snapshots in ClickHouse so search does not require SQLite joins.

### `retention_settings`

Stores:

- singleton id
- message_retention_days nullable
- raw_event_retention_days nullable
- updated_at

Allowed values remain:

- `null`
- `30`
- `90`

### `worker_heartbeats`

Stores:

- service_name
- last_seen_at
- metadata_json
- created_at
- updated_at

Purpose:

- admin operations can show listener freshness even when channels are quiet.

### `schema_migrations` and `data_migrations`

Migrations must track:

- applied schema migration versions
- source PostgreSQL migration run id
- source row counts
- destination row counts
- migration started/finished timestamps

## Listener and Ingestion Design

The Go listener must preserve the current reliability goal: once a websocket event reaches the
process, write the raw event durably before normalization or enrichment.

Flow:

1. Load enabled channels from SQLite.
2. Resolve missing Kick metadata before subscribing.
3. Connect to Kick Pusher websocket.
4. Subscribe to `chatrooms.{chatroom_id}.v2` and required channel-level streams.
5. Receive `App\Events\ChatMessageEvent`.
6. Parse minimal identifiers.
7. Insert raw event into ClickHouse.
8. Normalize event into sender, channel snapshot, message content, emotes, reply metadata, badges,
   and timestamps.
9. Upsert sender profile cache in SQLite.
10. Insert normalized message into ClickHouse idempotently.
11. Append processing attempt result.
12. Write listener heartbeat periodically.

Recovery behavior:

- On startup, scan raw events that do not have a matching normalized message and retry processing.
- Failed processing attempts remain visible in admin operations.
- Websocket failures reconnect with backoff.
- Channel subscription changes take effect through periodic resync without manual restart.

## Data Management Behavior

Admin data-management endpoints must keep the same request and response shape.

Summary:

- counts for channels, senders, messages, raw events
- database/table sizes where available
- retention settings

Cleanup preview:

- old messages
- old raw events
- specific channel
- specific sender

Cleanup confirm:

- requires exact confirmation text from preview
- deletes ClickHouse rows through controlled mutations
- does not delete SQLite metadata rows for channels/senders

ClickHouse delete note:

- Use preview counts before mutation as the response source.
- Use synchronous mutations where practical for admin-confirmed cleanup.
- Document that ClickHouse physical storage reclamation may lag behind logical deletion.

## Migration Plan

The one-time migrator reads PostgreSQL and writes to ClickHouse/SQLite.

Migration order:

1. Admin users to SQLite.
2. Channels to SQLite.
3. Senders to SQLite.
4. Retention settings to SQLite.
5. Worker heartbeat state to SQLite if useful.
6. Chat messages to ClickHouse.
7. Raw Kick events to ClickHouse.
8. Raw event status/attempt history to ClickHouse attempt rows.

Rules:

- Migration is idempotent.
- Source IDs are preserved where the current API exposes IDs.
- `kick_message_id` remains the message dedupe key.
- JSONB payloads are serialized as canonical JSON strings.
- Timestamps are normalized to UTC.
- Migration validates row counts and representative samples before reporting success.
- The migrator must not delete PostgreSQL data.

## Docker and Environment Plan

During transition:

- keep `postgres`, Python `api`, and Python `listener` available when needed for parity checks
- add `clickhouse`
- add Go API and listener build targets
- keep `web` pointed at the selected API base URL

Final default runtime:

```text
clickhouse
api
listener
web
```

Environment variables should include:

- ClickHouse connection DSN/host/user/password/database
- SQLite database path
- JWT secret, algorithm, expiry, cookie name, cookie secure flag, cookie same-site value
- default super admin email/password
- listener backoff/resync/heartbeat intervals
- export max rows
- CORS allowed origins

## Testing and Verification

Each phase must include tests that match its risk.

Required verification layers:

- Go unit tests for pure domain/use-case behavior.
- Go integration tests for SQLite repositories.
- Go integration tests for ClickHouse repositories.
- API contract tests against captured current backend fixtures.
- Migration tests using seeded PostgreSQL fixtures.
- Docker Compose smoke test against the Go runtime.
- Existing frontend checks after Go API parity.

Contract fixture coverage:

- successful login
- failed login
- current user
- admin users list/create
- followed channel list/add/delete
- message search with combinations of sender/channel/q/start/end
- reply-only and emote-only filters
- JSON export and CSV export
- analytics overview, volume, top senders, top channels, top emotes
- user profile analytics
- channel profile analytics
- operations summary
- data-management summary, retention update, cleanup preview, cleanup confirm

## Commit Strategy

Keep commits scoped to completed phase units.

Recommended commit boundaries:

- docs/archive and active rewrite plan
- contract inventory and fixtures
- Go workspace/tooling skeleton
- ClickHouse/SQLite migration foundation
- auth/admin API parity
- message search/export parity
- listener ingestion parity
- analytics/profile parity
- data migrator
- Compose cutover and smoke/docs

Do not combine unrelated implementation phases in one commit.

## Feature Map

| Phase | Task File                                            | Goal                                                          |
| ----- | ---------------------------------------------------- | ------------------------------------------------------------- |
| 1     | `docs/tasks/go_rewrite_01_contract_inventory.md`     | Capture current API contract and frontend dependencies        |
| 2     | `docs/tasks/go_rewrite_02_workspace_tooling.md`      | Create Go workspace, commands, config, and base runtime       |
| 3     | `docs/tasks/go_rewrite_03_storage_schema.md`         | Build ClickHouse and SQLite schemas and migrations            |
| 4     | `docs/tasks/go_rewrite_04_auth_admin_api.md`         | Rebuild auth, admin users, channels, and operations basics    |
| 5     | `docs/tasks/go_rewrite_05_messages_search_export.md` | Rebuild message search, pagination, filters, and export       |
| 6     | `docs/tasks/go_rewrite_06_listener_ingestion.md`     | Rebuild Kick listener, raw-event durability, parsing, retries |
| 7     | `docs/tasks/go_rewrite_07_analytics_profiles.md`     | Rebuild analytics, user profiles, and channel profiles        |
| 8     | `docs/tasks/go_rewrite_08_data_migration.md`         | Migrate PostgreSQL data to ClickHouse and SQLite              |
| 9     | `docs/tasks/go_rewrite_09_cutover_smoke_docs.md`     | Switch runtime to Go, smoke test, and update docs             |
