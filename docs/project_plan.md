# Kick Logs Project Plan

## Summary

Kick Logs is a self-hosted application for collecting public Kick chat messages from followed
channels, storing them durably, and searching historical chat through a web UI.

The default runtime is Go + NATS JetStream + ClickHouse + SQLite:

- Go API serves the existing HTTP contract.
- Go listener subscribes to Kick chat streams and publishes reached raw events to JetStream.
- Go processor consumes JetStream batches and writes raw events plus normalized messages to ClickHouse.
- NATS JetStream is the durable live chat ingestion backlog for issue #23.
- ClickHouse stores chat messages, raw Kick events, exports, analytics, profile aggregates, and
  public request-form history.
- SQLite stores admin users, followed channels, sender profile cache, retention settings,
  heartbeats, webhook inbox/registry state, legacy queue compatibility tables, and migration
  metadata.
- Next.js serves the public/search/admin web UI.

Default local startup:

```powershell
docker compose up --build -d
```

## Product Goals

- Track one or more Kick channels.
- Add/disable followed channels from an admin panel.
- Persist useful message data, including normalized fields and raw payloads.
- Search messages with optional filters:
  - sender nickname
  - channel nickname/slug
  - message content
  - start datetime
  - end datetime
  - reply-only
  - emote-only
- Show results with infinite scroll.
- Render reply context and inline emotes in search results.
- Export filtered messages as JSON or CSV.
- Provide public analytics, user profile pages, and channel profile pages.
- Provide a public request form for channel tracking requests and general feedback.
- Provide admin operations and data-management panels.
- Let admins review, status-change, note, and archive public requests.
- Run locally through Docker Compose.

## Runtime Architecture

- `apps/api-go`: Go backend, listener, processor, migrator, ClickHouse/SQLite repositories.
- `apps/web`: Next.js frontend using pnpm, Tailwind, shadcn/ui, and lucide-react.
- `nats_data`: Docker volume that stores JetStream durable raw-event backlog.
- `clickhouse`: default data-plane database.
- `api_go_data`: Docker volume that stores SQLite control-plane data, legacy queue tables, and
  webhook inbox rows.
- `clickhouse_data`: Docker volume that stores ClickHouse data.

Detailed structure lives in `docs/architecture.md`.

## Auth And Admin

- Login is required only for admin/backend management flows.
- Public search, analytics, user profiles, and channel profiles do not require login.
- Roles:
  - `super_admin`
  - `admin`
- Default super admin:
  - email: `admin@kicklogs.local`
  - password: `admin123`
- Default credentials are overridable with:
  - `DEFAULT_SUPER_ADMIN_EMAIL`
  - `DEFAULT_SUPER_ADMIN_PASSWORD`
- Passwords are hashed, never stored as plain text.
- Super admin can create new admin users.
- Admin dashboard route: `/admin`.

## Search Behavior

Search route: `/search`.

API:

```text
GET /messages?sender=&channel=&q=&start=&end=&cursor=&limit=&reply_only=&emote_only=
GET /messages/export?format=json|csv&sender=&channel=&q=&start=&end=&reply_only=&emote_only=&limit=
```

Filter semantics:

- All filters are optional.
- Non-empty filters combine with `AND`.
- Empty `sender` searches all users.
- Empty `channel` searches all channels.
- Empty `q` searches all message contents.
- `sender` uses case-insensitive exact matching against sender username/slug snapshots.
- `channel` and `q` use case-insensitive contains matching.
- `start` and `end` filter by message timestamp.
- `reply_only=true` narrows results to reply messages.
- `emote_only=true` narrows results to messages with at least one parsed emote.
- Results are ordered newest-first.
- Infinite scroll uses cursor pagination based on `message_created_at|message_id`.
- Export uses the same filters and clamps rows to `MESSAGE_EXPORT_MAX_ROWS`.
- Bare `/search` does not fetch latest messages until the user submits a search.
- The search UI defaults missing date inputs to the last 7 days through today; users can clear
  either field to omit that filter.

## Kick Payload And Rendering

Live chat payload fields observed from Pusher:

- `id`
- `chatroom_id`
- `content`
- `type`
- `created_at`
- `sender.id`
- `sender.username`
- `sender.slug`
- `sender.identity.color`
- `sender.identity.badges`
- `metadata.message_ref`
- `metadata.original_sender`
- `metadata.original_message`
- `thread_parent_id`

Reply rendering uses:

- `message_type === "reply"`
- `reply_metadata.original_sender.username`
- `reply_metadata.original_message.content`

Emotes arrive as tokens such as:

```text
[emote:37226:KEKW]
```

They are parsed into structured values and rendered with:

```text
https://files.kick.com/emotes/{id}/fullsize
```

If the image fails, the UI falls back to emote name/token text.

## Public Pages

- `/`: compact public landing page with self-hosted positioning and analytics blocks.
- `/request`: public request form for channel tracking requests and general feedback.
- `/search`: public historical message search.
- `/users/[slug]`: public sender profile with analytics and latest messages.
- `/channels/[slug]`: public channel profile with metadata, analytics, and latest messages.

Profile links follow Kick URL behavior: visible usernames may contain `_`, but profile route links
convert `_` to `-`.

## Admin Pages

`/admin` supports:

- login/logout session flow
- followed channel list
- add channel by slug/nickname
- disable channel without deleting historical data
- create admin user when current user is `super_admin`
- operations health, storage growth, raw event status, and listener freshness
- retention settings and guarded cleanup preview/confirm flows
- request-form review workflow with status, notes, and archive actions

Operations treats JetStream pending and ack-pending counts as the active live chat backlog. SQLite
raw-event queue depth is shown only as legacy/migration state. Processed raw-event history is read
from ClickHouse attempts, while terminal ignored events are excluded from retryable failed-event
actions.

## Data Management

Retention values:

- `null`: keep forever
- `30`: keep 30 days
- `90`: keep 90 days

Cleanup targets:

- old messages
- old raw events
- specific channel
- specific sender

Cleanup requires a dry-run preview and exact confirmation text before execution. ClickHouse cleanup
uses mutations; logical rows are removed before the API returns, while physical disk reclamation may
lag behind background merges.

The listener no longer uses SQLite as the live chat hot-path queue in the JetStream architecture.
Processed/ignored webhook inbox rows are pruned after the short retention window.

## Legacy Data Migration

PostgreSQL migration uses `migrate-go` when upgrading from an older deployment. PostgreSQL is not a
current runtime service; expose or restore the old database separately and provide
`POSTGRES_SOURCE_DSN`.

```powershell
docker compose up -d clickhouse
$env:POSTGRES_SOURCE_DSN = "postgresql://kick_logs:kick_logs@host.docker.internal:5432/kick_logs"
docker compose --profile tools run --rm migrate-go -target=data -dry-run
docker compose --profile tools run --rm migrate-go -target=data -execute
docker compose --profile tools run --rm migrate-go -target=data -validation-only
```

The migrator is read-only against PostgreSQL. PostgreSQL source data and dumps are not deleted by
the migration plan.

## Test Plan

Backend Go:

- config and route tests
- auth/session tests
- admin users/channels tests
- message search/export tests
- listener parsing and processing tests
- analytics/profile tests
- data-management tests
- legacy PostgreSQL migration tests
- ClickHouse repository integration tests when Docker is available

Frontend:

- search form and URL query behavior
- infinite scroll helpers
- emote/reply/link/highlight rendering
- auth guard and login flow
- admin channel/user/operations/data-management panels
- landing analytics, user profiles, and channel profiles

Docker:

- `docker compose up --build -d`
- `GET /health`
- default super admin login
- public search/export
- analytics/profile routes
- admin operations/data-management routes
- unauthenticated admin rejection

## Constraints

- Kick web endpoints and Pusher channels are undocumented and may change.
- Once a websocket event reaches the listener, raw payload persistence must happen before heavy
  normalization.
- Historical Python/PostgreSQL docs live under `docs/archive/`.
- User manually pushes commits.
- Keep implementation steps commit-sized.
