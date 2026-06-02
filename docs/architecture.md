# Kick Logs Architecture

## Overview

Kick Logs is a self-hosted Kick chat logging application. The default runtime is now:

- Go HTTP API
- Go Kick listener worker
- NATS JetStream for the issue #23 durable raw-event ingestion backlog
- ClickHouse for chat messages, raw Kick events, exports, analytics, and profile aggregates
- SQLite for admin/control-plane state
- Next.js web UI

## Runtime Services

Default `docker compose up --build -d` services:

```text
clickhouse
nats
api
listener
web
```

Tool services:

```text
migrate-go # tools profile
```

The API is exposed on `http://localhost:8000`.

## Monorepo Layout

```text
kick-logs/
  apps/
    api-go/       # default Go backend/listener/migrator
    web/          # Next.js frontend
  docs/
  compose.yaml
  README.md
```

## Go Backend Layout

```text
apps/api-go/
  cmd/
    api/
    listener/
    migrate/
  internal/
    app/
    config/
    domain/
    ports/
    usecase/
      analytics/
      auth/
      channels/
      data_management/
      datamigration/
      listener/
      messages/
      operations/
      profiles/
    infra/
      auth/
      clickhouse/
      data_management/
      kick/
      migrations/
      operations/
      postgres/
      sqlite/
    http/
      middleware/
      routes/
      schemas/
```

Dependency direction:

```text
http -> usecase -> domain
infra -> ports/domain
cmd -> app/config/http/worker
```

Rules:

- Domain code does not import HTTP, SQLite, ClickHouse, JWT, websocket, or Kick clients.
- Use cases depend on ports/interfaces.
- Infrastructure owns SQL, external Kick calls, JWT, password hashing, websocket clients, and
  migration adapters.
- HTTP schemas are explicit Go structs matching the frontend contract.

## Storage

SQLite stores control-plane data:

- `admin_users`
- `followed_channels`
- `sender_profiles` as a best-effort cache
- `retention_settings`
- `worker_heartbeats`
- `raw_event_queue` for active pending/claimed/failed listener work
- `kick_webhook_events` as a short-retention webhook inbox
- `kick_event_subscriptions`
- schema/data migration bookkeeping

ClickHouse stores data-plane rows:

- `chat_messages`
- `raw_kick_events`
- `raw_event_attempts`
- `channel_subscription_periods`

`chat_messages` is denormalized. It includes sender/channel snapshots, normalized helper columns,
reply metadata, emote arrays/image URLs, badges, raw payload JSON, message timestamps, and ingestion
timestamps. Search, export, analytics, and profile pages should not join back to SQLite on hot paths.

NATS JetStream is being introduced by issue #23 as the durable live chat ingestion backlog:

- Stream: `KICK_RAW_EVENTS`
- Subject: `kick.raw.chat`
- Consumer: `kick-raw-event-processor`
- Retention: work-queue style backlog for unacked raw chat events
- Storage: file-backed JetStream volume

After the cutover, live chat capture should publish reached raw events to JetStream before
normalization. SQLite raw queue tables remain legacy/migration state until a later cleanup.

## API Surface

The Go API preserves the existing frontend contract:

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
GET /channels/{slug}/subscription-summary

POST /webhooks/kick

GET  /admin/webhooks/health
POST /admin/webhooks/sync
```

Public routes:

- `/messages`
- `/messages/export`
- `/analytics/*`
- `/users/{slug}/analytics`
- `/channels/{slug}/analytics`
- `/channels/{slug}/subscription-summary`
- `/webhooks/kick` (no auth, signature-verified)

Prediction is client-side only. The browser calls Kick's public endpoints
(`https://kick.com/api/v2/channels/{slug}` and `.../predictions/latest`) directly; the frontend
normalizes the response and derives totals/point-share/winner. The Go API does not serve prediction
data and no prediction data is stored.

Admin routes require the HttpOnly JWT session cookie and an admin/super-admin role.

## Search Contract

Endpoint:

```text
GET /messages?sender=&channel=&q=&start=&end=&cursor=&limit=&reply_only=&emote_only=
GET /messages/export?format=json|csv&sender=&channel=&q=&start=&end=&reply_only=&emote_only=&limit=
```

Rules:

- All filters are optional.
- Non-empty filters combine with `AND`.
- Empty `sender` searches all senders.
- Empty `channel` searches all channels.
- Empty `q` searches all contents.
- `sender` uses case-insensitive exact matching against sender username/slug snapshots.
- `channel` and `q` use case-insensitive contains matching.
- `start` and `end` filter by `message_created_at`.
- `reply_only=true` restricts results to reply messages.
- `emote_only=true` restricts results to messages with parsed emotes.
- Results are newest-first.
- Cursor pagination uses `message_created_at|message_id`.

## Listener

The Go listener follows the durable-inbox rule:

1. Load enabled followed channels from SQLite.
2. Resolve missing Kick channel metadata.
3. Subscribe to Kick Pusher streams.
4. Persist received `App\Events\ChatMessageEvent` payloads to ClickHouse `raw_kick_events`.
5. Enqueue active work in SQLite `raw_event_queue`.
6. Normalize queue items into visible `chat_messages`.
7. Upsert sender profile cache in SQLite only as a throttled best-effort cache.
8. Append raw-event attempt history in ClickHouse.
9. Delete processed queue rows from SQLite after attempt history is durable.
10. Record listener heartbeat in SQLite.

The listener reconnects/resyncs periodically so admin channel changes take effect without a manual
restart. Visible message inserts remain idempotent by `kick_message_id`. Permanent invalid payloads
write terminal ignored attempts and leave the active queue instead of retrying forever.

## Data Management

Admin data-management endpoints operate against SQLite settings and ClickHouse rows.

Operations/data-management views distinguish active SQLite runtime state from ClickHouse history:

- `raw_event_queue` row counts represent active pending/claimed/failed listener work, not all-time
  processed event history.
- `raw_kick_events` and `raw_event_attempts` remain the durable raw-event archive/audit history.
- processed/ignored webhook inbox rows are pruned from SQLite after the short retention window,
  while normalized subscription periods stay in ClickHouse.

Retention settings:

- `null`: keep forever
- `30`: keep 30 days
- `90`: keep 90 days

Cleanup preview returns affected counts and the required confirmation text. Cleanup confirmation
executes ClickHouse mutations with synchronous completion requested by the API. Logical rows are
removed before the API returns, but physical disk reclamation can lag behind ClickHouse background
merges.

## Data Migration

`migrate-go` owns legacy PostgreSQL to SQLite/ClickHouse migration. PostgreSQL is not part of the
current Compose runtime; restore or expose the old database separately and pass
`POSTGRES_SOURCE_DSN` when running the data migrator:

```powershell
docker compose up -d clickhouse
$env:POSTGRES_SOURCE_DSN = "postgresql://kick_logs:kick_logs@host.docker.internal:5432/kick_logs"
docker compose --profile tools run --rm migrate-go -target=data -dry-run
docker compose --profile tools run --rm migrate-go -target=data -execute
docker compose --profile tools run --rm migrate-go -target=data -validation-only
```

The migrator is read-only against PostgreSQL. It preserves source IDs where API responses expose
them, validates Go-compatible bcrypt admin hashes, canonicalizes JSON payloads, normalizes
timestamps to UTC, validates counts/samples, and records migration metadata in SQLite.

## Frontend Architecture

Next.js App Router remains the frontend runtime:

```text
apps/web/src/
  app/
  components/
  features/
  lib/
  types/
```

Frontend rules:

- `/` is the public landing page.
- `/search` is public and uses `/messages`.
- `/admin` is authenticated and manages channels, admin users, operations, and data management.
- `/users/[slug]` and `/channels/[slug]` are public profile/analytics pages.
- `lib/api-client.ts` owns base URL, credentials, and response handling.
- UI work must follow `docs/design/design.md`.

## Verification

Primary Go checks:

```powershell
cd apps/api-go
go test ./...
go vet ./...
```

Frontend checks:

```powershell
pnpm --filter @kick-logs/web test
pnpm --filter @kick-logs/web typecheck
pnpm --filter @kick-logs/web lint
pnpm --filter @kick-logs/web build
pnpm format:check
```

Runtime smoke:

```powershell
docker compose up --build -d
docker compose ps
```
