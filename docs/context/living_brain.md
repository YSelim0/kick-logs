# Living Brain

This file is the active project memory. Keep it updated whenever project behavior, architecture,
implementation details, or working assumptions change.

## Current State

- Branch: `feat/go-clickhouse-rewrite`.
- Active Go rewrite plan status: Phase 9 cutover smoke/docs is complete.
- Default runtime is:
  - `clickhouse`
  - `api` built from `apps/api-go`
  - `listener` built from `apps/api-go`
  - `web`
- Python/FastAPI/PostgreSQL remains in-repo as a reference and rollback runtime through the
  `python-reference` Compose profile:
  - `postgres`
  - `api-python`
  - `listener-python`
- `migrate-go` is under the `tools` profile and owns PostgreSQL to SQLite/ClickHouse migration.
- PostgreSQL source data and volumes are not removed by the cutover.

## Default Data Stores

- SQLite stores control-plane data:
  - admin users
  - followed channels
  - sender profile cache
  - retention settings
  - worker heartbeats
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

## Listener Rules

- The listener loads enabled channels from SQLite.
- It resolves missing Kick metadata before subscription.
- It subscribes to `chatrooms.{chatroom_id}.v2` plus channel-level streams.
- Once a Kick websocket chat event reaches the process, persist the raw event to ClickHouse before
  normalization, sender enrichment, or visible message insertion.
- Raw-event processing is at-least-once and idempotent; visible messages dedupe by
  `kick_message_id`.
- Listener heartbeat state is stored in SQLite `worker_heartbeats`.
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
- Every implementation agent must read `docs/implementation_plan.md` and the matching active task
  file before changing files.
- Active Go rewrite task files are scoped handoff contracts; do not implement work from another
  phase unless the user explicitly changes the plan.
- Archived MVP/post-MVP files are historical context only.
- Keep documentation and context current with implementation changes.
- Update `docs/context/recent_changes.md` with a short latest-change handoff after each meaningful
  change.
- User manually pushes commits.
