# Kick Logs Architecture

## Overview

Kick Logs is a self-hosted Kick chat logging application. The default runtime is now:

- Go HTTP API
- Go Kick listener worker
- NATS JetStream for the durable raw-event ingestion backlog
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
processor
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
- `raw_event_queue` as legacy/migration state after the JetStream cutover
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

NATS JetStream is the durable live chat ingestion backlog:

- Stream: `KICK_RAW_EVENTS`
- Subject: `kick.raw.chat`
- Consumer: `kick-raw-event-processor`
- Retention: work-queue style backlog for unacked raw chat events
- Storage: file-backed JetStream volume

Live chat capture publishes reached raw events to JetStream before normalization. SQLite raw queue
tables remain legacy/migration state until a later cleanup.

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

GET    /admin/watched-senders
POST   /admin/watched-senders
DELETE /admin/watched-senders/{sender_id}

GET /admin/notification-settings
PUT /admin/notification-settings

GET  /admin/users
POST /admin/users

GET /admin/operations/summary

GET  /admin/data-management/summary
PUT  /admin/data-management/retention-settings
POST /admin/data-management/cleanup/preview
POST /admin/data-management/cleanup/confirm
POST /admin/data-management/import/preview
POST /admin/data-management/import/confirm

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

The Go listener follows the durable-capture rule:

1. Load enabled followed channels from SQLite.
2. Resolve missing Kick channel metadata.
3. Subscribe to Kick Pusher streams.
4. Poll Kick's numeric channel recent-messages endpoint as a short backfill source.
5. Serialize reached `App\Events\ChatMessageEvent` payloads into raw chat envelopes.
6. Publish those envelopes to NATS JetStream.
7. Wait for JetStream PubAck before counting an event as captured.
8. Record listener heartbeat in SQLite.

The listener does not normalize chat messages and no longer opens ClickHouse in the JetStream path.
It keeps the Kick websocket open while the followed-channel set is unchanged and reconnects only on
websocket failure or an actual enabled-channel set change. Recent-message polling uses
`/api/v2/channels/{kick_channel_id}/messages?sort=desc`, injects the followed channel's
`kick_chatroom_id`, and publishes through the same JetStream envelope/dedupe path as Pusher. Recent
fetches use bounded concurrency so active chats do not outpace the endpoint page while the listener
walks every enabled channel. A short in-memory seen set prevents quiet channels from republishing
the same recent endpoint rows on every poll tick. Visible message normalization and ClickHouse batch
writes belong to the processor service.

## Processor

The Go processor owns the live chat normalization and ClickHouse write path:

1. Pull a batch from the durable JetStream consumer.
2. Insert the raw event archive rows into ClickHouse.
3. Normalize valid chat payloads into `chat_messages`.
4. Insert visible messages in a batch.
5. Insert raw-event attempt audit rows.
6. ACK processed events only after required ClickHouse writes succeed.
7. TERM terminal invalid/ignored payloads only after diagnostic attempt rows are durable.
8. NACK transient failures so JetStream redelivers them.

The processor uses at-least-once delivery and relies on deterministic message identity plus
read-side dedupe to keep public search/profile results stable under redelivery.

### Watched-Sender Email Notification (optional)

After a `chat_messages` batch insert succeeds, the processor checks each message's sender username
against an in-memory watchlist (`internal/usecase/watchlist`) and, on a match, fires an SMTP alert
email (`internal/infra/notify`) in its own goroutine so a slow/blocked mail server cannot delay
JetStream ack of durably-stored messages. A per-sender cooldown (default 10 minutes) suppresses
repeat emails from an active chatter. The feature is disabled unless `SMTP_HOST` and
`NOTIFY_EMAIL_TO` are both configured.

The watched-username list itself is admin-managed, not an env var: `watched_senders` (SQLite,
control-plane) holds the list, `GET/POST/DELETE /admin/watched-senders` (admin-authenticated) manage
it from the `/admin/notifications` panel, and the processor polls
`WatchedSenderRepository.ListUsernames` every `WATCHLIST_REFRESH_INTERVAL_SECONDS` (default 30) to
push the current list into the in-memory watchlist via `WatchlistService.SetUsernames`. Adding or
removing a watched username from the admin panel takes effect on the next poll tick, with no
processor restart.

The per-sender cooldown is admin-managed the same way: `notification_settings` (SQLite,
single-row control-plane table) holds `cooldown_seconds`, `GET/PUT /admin/notification-settings`
(admin-authenticated) manage it from the same `/admin/notifications` panel, and the processor polls
`NotificationSettingsRepository.GetNotificationSettings` on the same interval to push the current
value into `WatchlistService.SetCooldown`. `NOTIFY_EMAIL_COOLDOWN_SECONDS` only seeds the row the
first time the app starts; after that it has no effect and the SQLite value is authoritative.

## Data Management

Admin data-management endpoints operate against SQLite settings and ClickHouse rows.

Operations/data-management views distinguish JetStream live backlog, legacy SQLite runtime state,
and ClickHouse history:

- JetStream pending and ack-pending counts represent the active live chat backlog.
- `raw_event_queue` row counts are legacy/migration state after the JetStream cutover, not the
  active live chat queue.
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
