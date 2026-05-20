# Living Brain

This file is the active project memory. Keep it updated whenever project behavior, architecture,
implementation details, or working assumptions change.

## Current State

- Branch: `feat/issue-9-ingestion-batching`.
- Active plan: GitHub issue #9, stabilize high-volume Kick chat ingestion. Plan lives in
  `docs/implementation_plan.md`; phase task files live under `docs/tasks/issue_09_*`.
- Default runtime is:
  - `clickhouse`
  - `api` built from `apps/api-go`
  - `listener` built from `apps/api-go`
  - `web`
- Python/FastAPI/PostgreSQL runtime code has been removed from the repo.
- `migrate-go` is under the `tools` profile and owns legacy PostgreSQL to SQLite/ClickHouse
  import when `POSTGRES_SOURCE_DSN` points at an external/restored source database.
- The local legacy PostgreSQL Docker volume was intentionally not deleted during cleanup.
- Local data migration completed successfully into fresh ClickHouse/SQLite targets with:
  - `admin_users`: 2
  - `followed_channels`: 7
  - `sender_profiles`: 8570
  - `retention_settings`: 1
  - `worker_heartbeats`: 1
  - `chat_messages`: 123790
  - `raw_kick_events`: 121664
  - `raw_event_attempts`: 121664

## Default Data Stores

- SQLite stores control-plane data:
  - admin users
  - followed channels
  - sender profile cache
  - retention settings
  - worker heartbeats
  - raw-event work queue (`raw_event_queue`)
  - schema/data migration metadata
- ClickHouse stores data-plane rows:
  - chat messages
  - raw Kick events
  - raw-event processing attempts
- `chat_messages` is denormalized for search/export/analytics/profile paths. Hot read paths should
  not join back to SQLite.

## API Contract

The Go API preserves the existing frontend surface:

```text
GET  /health
POST /auth/login
POST /auth/logout
GET  /auth/me
GET  /messages
GET  /messages/export
GET  /admin/channels
POST /admin/channels
DELETE /admin/channels/{channel_id}
GET  /admin/users
POST /admin/users
GET  /admin/operations/summary
GET  /admin/data-management/summary
PUT  /admin/data-management/retention-settings
POST /admin/data-management/cleanup/preview
POST /admin/data-management/cleanup/confirm
GET  /analytics/overview
GET  /analytics/message-volume
GET  /analytics/top-senders
GET  /analytics/top-channels
GET  /analytics/top-emotes
GET  /users/{slug}/analytics
GET  /channels/{slug}/analytics
```

Public routes remain unauthenticated. Admin routes require the HttpOnly JWT session cookie and an
admin/super-admin role.

## Completed Go Rewrite Phases

- Phase 1: contract inventory.
- Phase 2: Go workspace/tooling.
- Phase 3: SQLite/ClickHouse schema and repositories.
- Phase 4: auth/admin API parity.
- Phase 5: message search/export parity.
- Phase 6: listener ingestion parity.
- Phase 7: analytics/profile parity.
- Phase 8: PostgreSQL data migration.
- Phase 9: cutover smoke/docs.
- Phase 9 also fixed SQLite sender-profile upsert to handle live listener races on
  `sender_profiles.kick_user_id` and `sender_profiles.slug`.
- Completed Go rewrite plan, task files, and contract inventory now live under
  `docs/archive/go_rewrite/`.

## Listener Rules

- The listener loads enabled channels from SQLite.
- It resolves missing Kick metadata before subscription.
- It subscribes to `chatrooms.{chatroom_id}.v2` plus channel-level streams.
- Once a Kick websocket chat event reaches the process, persist the raw event to ClickHouse
  archive **and** enqueue a tracking row into SQLite `raw_event_queue` before acknowledging the
  event. Issue #9 phase 1 moved the work queue out of ClickHouse so the worker hot path no
  longer runs heavy `raw_event_attempts` JOIN queries.
- Workers list pending rows and claim them from SQLite, then load the raw payload from
  ClickHouse by id, normalize, and insert the visible chat message.
- Raw-event processing is at-least-once and idempotent; visible messages dedupe by
  `kick_message_id`.
- Listener heartbeat state is stored in SQLite `worker_heartbeats`.
- At startup the listener backfills any unprocessed ClickHouse raw events into the queue and
  resets stale `claimed` rows older than `RawEventProcessingTimeout` back to `pending`; a
  background loop repeats the stale-claim sweep.
- Channel changes should take effect through periodic reconnect/resync without manual restart.

## Search Behavior

- `/search` is public.
- Bare `/search` does not automatically query the API; it shows
  `Arama yapmak için yukarıdaki formu kullanın.` until the user submits the form.
- Search filters are optional and combine with `AND`.
- `sender` uses case-insensitive exact matching against sender username/slug snapshots.
- `channel` and `q` use case-insensitive contains matching.
- Date inputs default to the last 7 days through now in the UI; users can clear them to omit date
  filters.
- `reply_only=true` returns reply messages only.
- `emote_only=true` returns messages containing parsed emotes.
- Infinite scroll uses the `message_created_at|message_id` cursor.
- JSON/CSV export uses the last submitted filters.

## UI Direction

- UI design source: `docs/design/design.md`.
- Dark-only palette:
  - `#26001B`
  - `#810034`
  - `#FF005C`
  - `#FFF600`
  - black
  - white
- Primary buttons should prefer `#FFF600`.
- Do not use blur, glow, colored lighting, or atmospheric background effects.
- Search results render dense rows inside one shared outer list container, not per-message modal or
  card components.
- Sender avatars are circular.
- Emotes render inline where they appear in message content.
- Reply rows show replied-to sender/content above the current message in muted gray text.
- Public profile links convert `_` to `-` in route slugs while keeping visible usernames unchanged.

## Locked Product Decisions

- Seed default super admin:
  - email: `admin@kicklogs.local`
  - password: `admin123`
- Default credentials are overridable by environment variables.
- `/` is the compact public landing page.
- `/search` is public historical message search.
- `/admin` is authenticated backend management.
- `/users/[slug]` and `/channels/[slug]` are public profile/analytics pages.
- Followed-channel deletion disables the channel and preserves historical data.
- Store useful normalized fields, parsed emotes, reply metadata, raw payload JSON, sender badges,
  and profile image URLs when available.
- Render emotes with `https://files.kick.com/emotes/{id}/fullsize` and fall back to text.

## Data Management

- Retention settings support `null`, `30`, and `90` days.
- Cleanup targets old messages, old raw events, a channel, or a sender.
- Cleanup requires preview plus exact confirmation text.
- Go cleanup uses ClickHouse mutations with synchronous completion requested by the API. Logical
  deletion completes before the API returns, but physical disk reclamation can lag behind ClickHouse
  background merges.

## Development Rules

- Every agent must read `AGENTS.md` and context files before making changes.
- Every implementation agent must read `docs/implementation_plan.md` before changing files.
- When an active task file exists under `docs/tasks/`, read the matching task file and stay inside
  that scope unless the user explicitly changes the plan.
- Archived MVP/post-MVP/Go rewrite files are historical context only.
- Keep documentation and context current with implementation changes.
- Update `docs/context/recent_changes.md` with a short latest-change handoff after each meaningful
  change.
- User manually pushes commits.
