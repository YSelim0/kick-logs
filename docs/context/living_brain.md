# Living Brain

This file is the active project memory. Keep it updated whenever project behavior, architecture,
implementation details, or working assumptions change.

## Current State

- Branch: `feat/issue-22-kick-subscription-webhooks`.
- Active plan: Kick webhook subscription tracking (see `docs/implementation_plan.md`, issue #22).
  Phase 6 (backend query APIs) complete. Phase 7 next (docs and smoke).
- Earlier: channels/users index search pages — `/channels` and `/users` search-first index pages
  added 2026-05-24.
- Responsive polish pass is current through 2026-05-22: profile rows, admin navigation, admin
  channel/user tables, operations dashboard, and data-management panels have mobile-specific
  layouts.
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

## Webhook Subscription Pipeline (issue #22, Phase 6 complete)

- `domain.KickWebhookEvent` — webhook inbox model; status: `pending/processed/failed/ignored`
- `domain.KickEventSubscription` — Kick event subscription registry; status: `active/deleted/error`
- `domain.ChannelSubscriptionPeriod` — ClickHouse normalized subscription period; deterministic `id`
- `domain.ChannelSubscriptionSummary` — active count aggregate returned by public API
- `domain.KickAPIEventSub` — lightweight type for Kick API event subscription responses
- `FollowedChannel.BroadcasterUserID int64` — Kick broadcaster user ID (0 = not yet resolved)
- Port: `KickWebhookEventRepository` (SQLite inbox), `KickEventSubscriptionRepository` (SQLite registry)
- Port: `SubscriptionPeriodRepository` (ClickHouse), `KickEventSubscriptionClient` (Phase 3 impl)
- Port: `FollowedChannelRepository.GetByBroadcasterUserID` added
- API contract additions: `GET /channels/{slug}/subscription-summary` (public), `GET /admin/webhooks/health` (admin), `POST /admin/webhooks/sync` (admin)
- SQLite migrations v5 (broadcaster_user_id), v6 (kick_webhook_events), v7 (kick_event_subscriptions)
- ClickHouse migration v5 (channel_subscription_periods, ReplacingMergeTree ORDER BY id)
- Active count query uses `FINAL` + `countDistinctIf(subscriber_kick_user_id, expires_at > now())`

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

## Search Index Pages

- `/users` and `/channels` are search-first index pages (no data on initial load).
- Submit-only: clicking the `Ara` button or pressing Enter fires the request. Typing alone never
  triggers an API call. Decision driven by ClickHouse `LIKE '%…%'` cost under heavy ingest load.
- Submit button requires minimum 2-character trimmed query and is disabled while loading.
- Results call `GET /analytics/top-senders?q=…&limit=20` and
  `GET /analytics/top-channels?q=…&limit=20`.
- The `q=` text search parameter is a backend free-text LIKE filter on username/slug (senders)
  and slug/display-name (channels). It applies before GROUP BY in the ClickHouse query.
- Empty-state message quotes the last submitted query.
- SiteHeader `ActiveRoute` supports `"channels"` and `"users"` to highlight the current nav pill.

## Prediction Feature

- `/prediction` is a search-first page (submit-only, ≥2 chars). Submit navigates to
  `/prediction/{slug}` (lowercased); it fetches no data itself.
- `/prediction/{slug}` fetches the prediction client-side and renders summary, donut +
  grouped-bar charts, outcome cards (winner = `KAZANAN` + accent border), and a horizontal top-users
  chart. Loading / not-found (404) / error states are handled; a refresh button re-runs the fetch.
- After the first successful prediction load, `/prediction/{slug}` re-fetches every 5
  seconds in the background for as long as the page is open, including after `LOCKED`,
  `CANCELED`/`CANCELLED`, or `RESOLVED`. Existing content remains mounted to avoid chart flash.
- Prediction is client-side only (issue #19, supersedes the earlier backend-proxy decision). The
  browser fetches Kick's undocumented public endpoints directly:
  `GET https://kick.com/api/v2/channels/{slug}` for a cached per-slug channel check, then
  `.../predictions/latest` for every live refresh. `features/prediction/kick-prediction-client.ts`
  validates the response shape, normalizes the snake_case payload into the existing `Prediction`
  shape, and derives total points, total votes, per-outcome point share, and the winner flag. No
  backend route, no persistence, no migration. Accepted risk: if Kick changes CORS/shape/blocking the
  page fails with its error state and there is no backend fallback; all direct-Kick logic stays behind
  `getPrediction(slug)` so a proxy can be restored without UI changes.
- Error mapping (thrown as `ApiClientError`): channel 404 or null prediction → 404 (not-found state);
  blocked/non-2xx/network/malformed JSON or malformed prediction shape → non-404 (error state).
- Charts use `recharts` (the repo's only charting dep). Categorical palette is tokenized (accent,
  warning, text-secondary, danger, border-strong, accent-hover) because the base palette has no
  blue/purple. State pills: RESOLVED=accent, LOCKED=warning, CANCELED/CANCELLED=warning,
  ACTIVE=neutral.
- SiteHeader `ActiveRoute` extended with `"prediction"`; a `Prediction` nav link points to
  `/prediction`.

## Rate Limiting

- Port: `internal/ports/ratelimit.go` — `RateLimiter` interface, `RateLimitResult` struct.
- Infra: `internal/infra/ratelimit/gcra.go` — GCRA via `throttled/throttled/v2`, `memstore`
  backend, lazy per-config GCRA limiter creation with `sync.Map`.
- Middleware: `internal/http/middleware/ratelimit.go` — `DefaultPolicies`, `ClientIP`, `RateLimit`.
- Config env vars: `RATE_LIMIT_ENABLED` (default `true`), `RATE_LIMIT_STORE_MAX_KEYS` (default
  `65536`), `RATE_LIMIT_TRUST_PROXY` (default `true`), `RATE_LIMIT_CLIENT_IP_HEADER` (default
  `CF-Connecting-IP`). All four are wired through `compose.yaml` and `.env.example`.
- Real client IP: when `RATE_LIMIT_TRUST_PROXY=true`, read `RATE_LIMIT_CLIENT_IP_HEADER`
  (`CF-Connecting-IP` behind Cloudflare; `X-Real-IP`/`X-Forwarded-For` behind a plain nginx proxy).
  The value is validated with `net.ParseIP` (X-Forwarded-For: first entry) so a forged/garbage header
  cannot inject an arbitrary key; otherwise it falls back to `RemoteAddr`.
- The middleware prefixes each store key with the policy `Name`, so two policies sharing the same
  rate params + key (analytics and profile-analytics, both 60/min burst 15 keyed by IP) get separate
  buckets instead of colliding.
- Trust-proxy is only safe when the app cannot be reached while bypassing the trusted proxy:
  `compose.yaml` binds api/web/ClickHouse host ports to `127.0.0.1` by default (`API_BIND_HOST` /
  `WEB_BIND_HOST` / `CH_BIND_HOST`), and the origin should be firewalled to the CDN. See
  `docs/operations/reverse_proxy_and_origin.md`.
- Policy table (most specific first):
  - `POST /admin/data-management/cleanup/confirm` → admin user ID, 3/min burst 1
  - `POST /auth/login` → IP, 20/10min burst 5; + IP+email in handler, 8/10min burst 3
  - `GET /messages/export` → IP, 3/min burst 2
  - `GET /messages` → IP, 20/min burst 10
  - `GET /analytics/*`, profile analytics → IP, 60/min burst 15
  - `POST|PUT|DELETE /admin/*` → admin user ID, 30/min burst 10
  - `GET /admin/*` → admin user ID, 120/min burst 30
  - `/health`, OPTIONS → unlimited (no matching policy)
- `POST /webhooks/kick` → no matching policy (rate-limit exempt; security is RSA-SHA256 signature + idempotent inbox insert)
- Admin keying: `TokenService.GetUserID(cookie)` (no DB hit), IP fallback on failure.
- Login dual-key: attacker cannot lock victim email by hammering from one IP — key includes
  attacker IP, not victim.
- Fail-open: limiter errors log a warning and pass the request through.

## API Contract

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
GET  /channels/{slug}/subscription-summary
POST /webhooks/kick
GET  /admin/webhooks/health
POST /admin/webhooks/sync
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
- Kick event subscription create uses `events: [{name, version: 1}]` with `method: webhook`; delete
  uses `DELETE /public/v1/events/subscriptions?id=<subscription_id>`.
- Webhook public key is auto-fetched from `GET /public/v1/public-key` when credentials exist.
- Webhook processor ignores events for disabled channels so stale remote subscriptions cannot pollute
  active subscriber counts.
- It subscribes to `chatrooms.{chatroom_id}.v2` plus channel-level streams.
- Once a Kick websocket chat event reaches the process, submit it to the in-memory buffered
  writer. The writer flushes batches to ClickHouse archive using `InsertEventsBatch` and then
  enqueues the same batch into SQLite `raw_event_queue` in one logical step before treating
  the batch as acknowledged. Buffered writes flush on `LISTENER_RAW_EVENT_WRITE_BATCH_SIZE`
  events or `LISTENER_RAW_EVENT_WRITE_FLUSH_INTERVAL_MS` whichever first. Phase 1 moved the
  work queue out of ClickHouse so the worker hot path no longer runs heavy
  `raw_event_attempts` JOIN queries.
- Workers list pending rows and claim them from SQLite, then load each raw payload from
  ClickHouse by id, normalize in memory, and write the entire tick's chat messages and
  raw-event attempts as one batch each. ClickHouse insert failure releases every claim back to
  pending and the next tick retries.
- A single `CircuitBreaker` is shared by the buffered writer and the worker loop. Consecutive
  ClickHouse failures open the breaker; while open, every listener goroutine that calls
  `Wait` sleeps for the current backoff window before the next attempt. Successful operations
  close the breaker and reset the backoff.
- Listener heartbeat metadata JSON carries buffered-writer stats and circuit-breaker state so
  the admin operations summary can surface ingestion health (queue backlog, oldest pending
  age, writer buffer depth, drops, flushes, ClickHouse insert failures, breaker state) without
  any cross-process call into the listener.
- Synthetic ingestion load harness lives at `apps/api-go/cmd/loadgen`. It seeds channels with
  slugs `loadgen-*`, emits configurable events-per-second through the real listener service,
  and reports buffered-writer stats plus emitted counts. The procedure is documented in
  `docs/operations/load_test.md`. External durable queues are deferred until a live load run
  shows the in-process pipeline is insufficient.
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

- UI design source: `docs/design/design.md` (v2). `docs/design/design.pen` holds the approved
  v2 screen set.
- Dark-only palette (v2 tokens; legacy magenta is fully replaced):
  - accent `#00e701` (hover `#00c701`, muted `rgba(0,231,1,0.2)`)
  - `bg-page #0b0e0f`, `bg-panel #191b1f`, `bg-elevated #24272c`, `bg-deep #000000`
  - `border-subtle #24272c`, `border-strong #474f54`
  - `text-primary #ffffff`, `text-secondary #9ca3af`, `text-muted #474f54`,
    `text-on-accent #0b0e0f`
  - `danger #ff4d4f`, `warning #facc15`
- Primary buttons fill `accent` with `text-on-accent` foreground.
- Typography: Geist Sans for UI, Geist Mono for labels/timestamps/metric values, wired via the
  `geist` package and Tailwind `font-sans` / `font-mono`.
- Do not use blur, glow, colored lighting, or atmospheric background effects.
- Search results render dense rows inside one shared outer list container, not per-message modal or
  card components.
- Sender avatars are circular.
- Emotes render inline where they appear in message content.
- Reply rows show replied-to sender/content above the current message in muted gray text.
- Public profile links convert `_` to `-` in route slugs while keeping visible usernames unchanged.
- User profile identity avatars are fixed 72px circles and must keep `min-w-[72px]` on both image
  and fallback initials paths so mobile flex rows cannot squeeze the profile photo.
- Admin mobile layout uses a hamburger drawer for section navigation. Channel admin rows collapse
  into mobile cards, user admin rows stack role/status under email, and operations/data-management
  sections wrap controls instead of relying on desktop table widths.

## Locked Product Decisions

- Seed default super admin:
  - email: `admin@kicklogs.local`
  - password: `admin123`
- Default credentials are overridable by environment variables.
- `/` is the compact public landing page.
- `/search` is public historical message search.
- `/admin` is authenticated backend management.
- `/users` is the public search-first user index (submit-on-click/Enter, no auto-search).
- `/channels` is the public search-first channel index (submit-on-click/Enter, no auto-search).
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
