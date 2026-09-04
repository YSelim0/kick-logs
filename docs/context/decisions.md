# Decisions

## 2026-09-04 (one-off message import tool)

- **The import tool is append-only and requires an explicit confirm phrase to write.** The user
  provided a 429-message JSON/CSV export pair to backfill into ClickHouse and asked for a safe
  dry-run first. `-dry-run` defaults to `true`; a real write additionally requires
  `-confirm=IMPORT-CHAT-MESSAGES` exactly, mirroring the existing admin cleanup pattern
  (preview + exact confirmation text) rather than trusting a single `-dry-run=false` flag alone.
- **Existing rows are matched by `kick_message_id` and are never updated**, only skipped. The tool
  never issues a ClickHouse `ALTER`/update against `chat_messages`; it only `INSERT`s rows whose
  `kick_message_id` was confirmed absent via `ExistingKickMessageIDs`, consistent with the
  project's existing dedup-by-`kick_message_id` contract for live ingestion.
- **JSON is the canonical input; CSV is verification-only.** The user's two export files are the
  same 429-message dataset in different shapes. The JSON shape (`{items, count, max_rows,
  truncated}`) carries structured `reply_metadata`, sender/channel objects, and emote objects the
  flat CSV does not, so it is the only accepted `-input`. `-verify-csv` cross-checks the
  `kick_message_id` set for peace of mind but never feeds data into the import itself.
- **Row id reuses the live listener's deterministic hash** (`fnv64a(kick_message_id)` masked into
  an `int63`) so a backfilled row's id matches what the live pipeline would have produced for the
  same `kick_message_id`, instead of minting a new unrelated id scheme for imported data. The
  function is duplicated into the new `cmd/importmessages` package rather than importing
  `internal/usecase/listener`, which would pull in unrelated live-ingestion dependencies for a
  one-off CLI tool.

## 2026-06-02 (channel/user top-list aggregate hardening)

- **Channel index and admin channel counts must avoid window scans.** `/channels` search and the
  `/admin/channels` message-count column both depend on `TopChannels`. The old implementation used
  `row_number() OVER (PARTITION BY kick_message_id ...)` before grouping, which could hit
  ClickHouse memory limits for high-volume channels such as `hype`.
- **Analytics aggregates now dedupe through `ReplacingMergeTree FINAL`.** New live messages use a
  deterministic id derived from `kick_message_id`, and local data has no same-message rows with
  divergent ids. `Overview`, `MessageVolume`, `TopSenders`, `TopChannels`, and `TopEmotes` now read
  `chat_messages FINAL` instead of materializing a per-message window or full-table
  `GROUP BY kick_message_id`. This keeps duplicate redeliveries from inflating counts while avoiding
  the expensive sort/window and large hash-aggregate shapes.
- **Parameterized `/channels` and `/users` searches still bypass cache.** The fix is query-shape
  hardening, not unbounded query-param caching.

## 2026-06-02 (Kick recent-message backfill)

- **Kick Pusher alone is not a complete capture source.** For `gokhanoner`, an independent Pusher
  subscriber and the local listener saw the same small set of message ids, while Kick's numeric
  `/api/v2/channels/{kick_channel_id}/messages?sort=desc` endpoint exposed additional recent
  messages visible in the web chat. The missing rows were therefore upstream of `/search`,
  JetStream, processor, and ClickHouse normalization.
- **The listener must poll recent messages in addition to Pusher.** The listener now polls the
  numeric channel messages endpoint for every enabled followed channel and publishes returned
  messages through the same raw-event envelope and JetStream PubAck path as Pusher.
- **Recent polling uses existing message-id idempotency.** Poll envelopes use `raw_event_id =
kick:{message_id}`, so overlap between Pusher and polling is safe. JetStream duplicate PubAck is
  not counted as a new captured event, and visible ClickHouse rows still dedupe by
  `kick_message_id`.
- **Recent polling also keeps a short in-memory seen set.** Quiet channels can return the same last
  messages for a long time, eventually outliving JetStream's duplicate window. The listener keeps
  successfully published recent raw event ids in memory for 1 hour so the endpoint does not inflate
  `raw_kick_events` and attempt history by republishing the same page every poll tick.
- **Kick recent payloads are normalized before capture envelope creation.** The endpoint returns
  `metadata` as `null`, an object, or a JSON string, and returns the numeric channel id in `chat_id`.
  The client converts metadata to an object and injects the followed channel's `kick_chatroom_id`
  into `chatroom_id` so the existing processor normalizer can handle the payload unchanged.
- **Poll cadence is configurable and enabled by default.** Defaults:
  `LISTENER_RECENT_MESSAGE_POLL_ENABLED=true` and
  `LISTENER_RECENT_MESSAGE_POLL_INTERVAL_SECONDS=10`. Recent fetches run with bounded concurrency
  (`LISTENER_RECENT_MESSAGE_POLL_CONCURRENCY=8` by default) because polling enabled channels
  sequentially can outlive the recent-message page window on active chats.

## 2026-06-02 (Kick Pusher capture health)

- **Kick/Pusher protocol heartbeats must be answered.** The listener websocket client treats
  `pusher:ping` as a protocol message and replies with `pusher:pong`; otherwise the server can mark
  the connection unhealthy or close it, creating capture gaps before JetStream or ClickHouse see an
  event.
- **Chat capture can subscribe to both current and legacy chatroom channel names.** The listener
  subscribes to `chatrooms.{chatroom_id}.v2` and `chatrooms.{chatroom_id}`. Duplicate delivery is
  acceptable because raw stream publish uses Kick message id based message ids and visible
  `chat_messages` reads/writes dedupe by `kick_message_id`.
- **Operations needs capture-side counters.** Processor pending/ack metrics only prove that received
  events are being drained. Listener heartbeat metadata now carries `captured_raw_events` so an
  operator can distinguish "Kick websocket did not deliver enough events" from "processor/ClickHouse
  fell behind."

## 2026-06-02 (global analytics cache)

- **Global public analytics may be stale for up to 1 hour.** Landing-page statistics are
  informational, not live operational state, so `overview`, day-bucket volume, and top-list
  analytics can be served from an in-memory API cache.
- **Parameterized analytics searches bypass cache.** Requests with `q`, `sender`, or `channel` are
  not cached so `/users` and `/channels` index searches do not create unbounded key churn.
- **Stale cached analytics is preferable to blank analytics.** If a cached global aggregate is older
  than the 1-hour fresh TTL but still within the 24-hour stale window, the API can return it when a
  ClickHouse refresh fails.
- **Operations data remains uncached.** Admin Operations must reflect current listener, processor,
  JetStream, and ClickHouse health. `/admin/channels` may tolerate cached global message counts
  because those counts are informational table decoration.
- **Landing must render partial analytics.** One failed analytics endpoint must not hide successful
  overview, volume, sender, channel, or emote data from the user.
- **Profile analytics may also be stale for up to 1 hour.** Public
  `/users/{slug}/analytics` and `/channels/{slug}/analytics` are statistical profile pages, so their
  ClickHouse aggregate response is cached by exact profile slug with the same bounded in-memory
  cache shape.
- **Profile analytics must degrade partially.** A single channel/user aggregate failure, for
  example a top-emotes memory-limit error, should not turn the whole profile into a 500. The API can
  return the identity plus available sections and leave failed sections empty.
- **Active subscription counts are never part of analytics cache.** Public
  `/channels/{slug}/subscription-summary` stays uncached and queries active periods on every
  request because the value can be used as live broadcaster-facing subscriber state.

## 2026-06-02 (issue #23 — JetStream durable ingestion)

- **NATS JetStream is the long-term live chat ingestion queue.** The previous in-process buffered
  writer plus SQLite `raw_event_queue` hardening is no longer considered the final design because it
  still keeps intentional drop behavior and SQLite hot-path pressure in the architecture.
- **Listener capture must be durable before success accounting.** A reached Kick
  `ChatMessageEvent` is counted as captured only after JetStream PubAck. Publish failure is an
  operational failure/backpressure condition, not a successful ingest.
- **SQLite is control-plane only after cutover.** Admin users, followed channels, sender cache,
  retention, heartbeats, webhook registry/inbox, and migration metadata remain in SQLite.
  `raw_event_queue` and `raw_event_claims` become legacy migration/compatibility tables, not active
  live chat queue state.
- **Processor owns ClickHouse batch writes.** Processor workers pull JetStream batches, insert raw
  archives and normalized `chat_messages` into ClickHouse, and ACK only after durable writes
  succeed. Transient ClickHouse failures must leave messages unacked for redelivery.
- **At-least-once plus idempotency is the contract.** Duplicate internal delivery is expected and
  must not create duplicate visible search/profile rows. Visible message identity stays based on
  Kick message id when available, with deterministic fallback identity for malformed/incomplete raw
  diagnostics.
- **Raw capture precedes strict normalization.** Parser/normalizer code must not discard a reached
  chat event before raw capture. Invalid or incomplete payloads are captured first and then marked
  terminal ignored/invalid by the processor.

## 2026-06-01 (issue #23 — sender profile cache is best-effort)

These decisions describe the short-lived SQLite queue hardening phase and are superseded for live
chat ingestion by the 2026-06-02 JetStream decision above. They still apply to legacy queue code,
sender profile cache behavior, and webhook inbox maintenance where noted.

- **Sender profile cache writes must not block chat ingestion.** The listener now treats
  `sender_profiles` as an optional cache. If SQLite upsert fails while processing a raw chat event,
  the listener logs a warning and continues building the visible `chat_messages` row from the sender
  snapshot already present in the raw Kick payload.
- **Visible message identity comes from the raw payload first.** If the cache write fails, the
  message still stores `sender_kick_id`, username, slug, color, badges, and profile-image hints from
  the payload. `normalizeMessagePayload` falls back to the Kick user id when no SQLite sender row id
  is available.
- **Sender profile cache writes are TTL-gated in memory.** The listener attempts at most one
  `sender_profiles` write per Kick user id every 10 minutes. The first observed message for a sender
  still writes immediately; repeated messages inside the TTL use the payload snapshot and avoid a
  SQLite write. Failed cache writes also consume the TTL window to prevent a busy SQLite cache from
  being hammered during high-volume chat.
- **Processed raw-event queue rows are pruned from SQLite (legacy queue mode).** The
  `raw_event_queue` table is not permanent history. In the superseded SQLite queue mode, after the
  worker wrote the processed ClickHouse attempt, `MarkProcessed` deleted the queue row. Active live
  chat ingestion now uses JetStream instead of this SQLite queue.
- **Attempt history is required before queue acknowledgement.** If `InsertAttemptsBatch` fails, the
  listener releases claimed queue rows back to pending instead of deleting them. `chat_messages`
  inserts are idempotent by `kick_message_id`, so retrying after an attempt-write failure preserves
  the audit/backfill guard without duplicating visible messages.
- **Permanent invalid raw events use terminal `ignored` attempts.** Malformed payloads, missing
  message ids, invalid sender/message shape, and known-unfollowed channel payloads are not retried
  until max attempts. They get a ClickHouse `raw_event_attempts.status = 'ignored'` record and are
  removed from the legacy SQLite queue in the superseded queue mode. JetStream-era operations treat
  `ignored`/`invalid` as terminal statuses alongside `processed`.
- **Webhook inbox has short terminal retention.** `kick_webhook_events` remains the idempotent
  receiver inbox, but processed/ignored rows older than 7 days are pruned by the webhook processor.
  Pending and failed webhook rows are never pruned by this path because they still represent work or
  operator-visible failures. Subscription periods remain durable in ClickHouse.
- **Admin Operations separates active backlog from history.** Raw-event archive counts shown in
  admin Operations come from ClickHouse history. JetStream pending + ack-pending is the active live
  chat backlog; SQLite `raw_event_queue` is surfaced only as legacy/migration state. Terminal
  ignored/invalid raw events are excluded from failed-event diagnostics.
- **Admin Data Management summary must stay read-only after setup.** The summary endpoint should not
  perform opportunistic SQLite writes during normal reads. Retention settings are created only when
  missing, because live listener/webhook writes can otherwise make a harmless admin read fail with
  `SQLITE_BUSY`.
- **Webhook inbox pruning is periodic maintenance, not per-tick work.** Terminal inbox cleanup is
  throttled so the processor does not compete with live webhook inserts/listener SQLite writes every
  5 seconds.

## 2026-06-01 (issue #22 — Kick webhook subscription tracking)

- **`broadcaster_user_id` stored as `int64` with 0 = unresolved**, same pattern as `kick_channel_id`
  in `followed_channels`. No pointer type; 0 is the sentinel for "not yet resolved via sync".
- **`kick_webhook_events` uses `INSERT OR IGNORE`** for idempotency so duplicate deliveries from
  Kick are silently dropped at the DB level without an extra SELECT round-trip.
- **Webhook endpoint excluded from rate limiting.** `POST /webhooks/kick` has no matching rate-limit
  policy. Security is RSA-SHA256 signature verification (fail-closed: 503 if public key missing,
  401 if signature invalid). Treating it like user traffic would throttle Kick's delivery retries.
- **Kick webhook signature message is `messageID + "." + timestamp + "." + raw body`.** The
  signature is base64 RSA-SHA256 over that message. `infra/kick/webhook_verifier.go` owns the
  verification logic.
- **`channel_subscription_periods` uses `ReplacingMergeTree(ingested_at)` ORDER BY `id`.** The
  deterministic `id` (`messageID` for new/renewal, `messageID_gifteeUserID` for gifts) makes
  re-processing idempotent. Queries use `FINAL` to deduplicate on read.
- **Subscription period `expires_at` is always set before storage.** The fallback (`created_at + 30d`)
  is applied during normalization (Phase 5), not at query time. The ClickHouse column is NOT NULL.
- **`kick_subscription_id` preserved on upsert conflict.** The `ON CONFLICT DO UPDATE` SQL uses
  `CASE WHEN excluded.kick_subscription_id != '' THEN ... ELSE existing END` so a sync error
  upsert does not wipe a previously recorded Kick subscription ID.
- **Subscription sync is non-blocking.** `SyncAll` returns void; per-channel errors are stored in
  the registry and surfaced through `GET /admin/webhooks/health`. The API never fails to start
  due to Kick API unavailability.
- **Processor stays in the API process for now.** The background worker runs in a goroutine inside
  `cmd/api`. Splitting into a separate binary is deferred until live load shows it is necessary;
  idempotency guarantees the transition is safe.
- **`ResolveBroadcasterUserID` tries the official public API first, then the web API fallback.**
  The official path is `GET /public/v1/channels?slug=...`; if that shape or availability changes,
  the existing `kick.com/api/v2/channels/{slug}` fallback preserves channel setup. All broadcaster-ID
  resolution logic is isolated in `infra/kick/event_subscription_client.go`.
- **Kick event subscription create uses batch `events`, not a single `type` field.** The request is
  `POST /public/v1/events/subscriptions` with `broadcaster_user_id`, `method: webhook`, and
  `events: [{name, version: 1}]`. Missing events are created per channel in one request to reduce
  Kick API rate-limit pressure.
- **Disabled channels are ignored by the webhook processor.** A stale remote Kick subscription can
  still deliver events if it predates the local registry or failed deletion; the processor must not
  count subscription periods for disabled channels.

## 2026-05-31 (issue #20 — rate limiting follow-up: real IP + origin lockdown)

- **Container ports bind to loopback by default.** `compose.yaml` now publishes api/web/ClickHouse
  ports as `${API_BIND_HOST:-127.0.0.1}` / `WEB_BIND_HOST` / `CH_BIND_HOST`. Previously they bound to
  `0.0.0.0`, so the app was reachable directly on the host IP, bypassing the reverse proxy and letting
  a client forge the client-IP header to defeat every IP-based rate limit (and hit ClickHouse reads
  directly). Self-hosters without a local reverse proxy set the bind host to `0.0.0.0`.
- **Client IP header is configurable.** New `RATE_LIMIT_CLIENT_IP_HEADER` (default `CF-Connecting-IP`)
  replaces the hardcoded header, so a plain-nginx self-hoster can use `X-Real-IP`/`X-Forwarded-For`.
  The resolved value is validated with `net.ParseIP` (first entry for XFF) so a garbage/forged header
  falls back to `RemoteAddr` instead of becoming a unique key.
- **Store keys are namespaced by policy name.** The middleware prefixes the limiter key with the
  policy `Name`. Two policies with identical rate params and key derivation (analytics and
  profile-analytics, both 60/min burst 15 keyed by IP) previously shared one GCRA bucket, causing
  unexpected 429s for a dashboard hitting both; they are now independent.
- **Origin firewall is required, not optional, when behind a CDN.** The origin IP is visible in DNS,
  so `https://<origin>` is hittable directly with a forged CDN header even after the loopback bind.
  Locking `80`/`443` to the CDN's published ranges (keeping SSH open) is the only fix. Documented
  generically in `docs/operations/reverse_proxy_and_origin.md`; the nginx `real_ip` snippet there is
  optional (the app reads the header directly).
- GCRA adapter now passes `context.Background()` to `RateLimitCtx` instead of `nil`. Login
  handler limiter-error logging was intentionally skipped (handler already fails open).

## 2026-05-31 (issue #20 — rate limiting)

- GCRA-based rate limiting added to all public and admin API endpoints via
  `github.com/throttled/throttled/v2` with an in-memory LRU `memstore` backend. A fixed-window
  counter was rejected: no separate burst control and double-counts across the window edge.
- `RateLimiter` interface added to `ports/ratelimit.go`. Implementation in
  `infra/ratelimit/gcra.go`. Middleware in `http/middleware/ratelimit.go`. Call sites depend only
  on the interface; swapping to a Redis backend later requires no middleware or policy changes.
- In-memory store now (`memstore`), Redis explicitly deferred — single instance, avoids new
  memory/failure pressure on the 4 GB VPS (see #15). `RATE_LIMIT_STORE_MAX_KEYS` (default 65536)
  bounds the LRU (~10 MB at ~150 bytes per GCRA entry).
- Global middleware + path match strategy: single middleware holds an ordered policy table; route
  files are untouched. First matching policy wins. `/health` and OPTIONS requests have no matching
  policy and are always unlimited.
- Real client IP via `CF-Connecting-IP` header under `RATE_LIMIT_TRUST_PROXY=true` (default),
  falling back to `RemoteAddr`. Trusted only because the origin is locked to Cloudflare IP ranges
  (infra prerequisite; VPS firewall + nginx `real_ip` are outside code scope).
- Login dual-key: middleware applies IP-only check (20/10min burst 5); login handler applies
  IP+email check (8/10min burst 3) after body parse. An attacker cannot lock a victim by email
  alone because the email key includes the attacker's own IP, not the victim's.
- Admin endpoints keyed by user ID extracted from JWT cookie via `TokenService.GetUserID()` (pure
  token parse, no DB hit), falling back to IP if cookie is missing or invalid. `cleanup/confirm`
  has a tighter independent policy (3/min burst 1) matched before the generic admin-write policy.
- `RATE_LIMIT_ENABLED=false` disables the middleware entirely. Limiter is nil in tests that do not
  exercise rate limiting so existing test helpers require no behavioral changes.
- Prediction endpoint excluded from all policies (moved client-side per #19).

## 2026-05-30 (issue #19 — prediction moved client-side)

- Prediction fetching and normalization moved from the Go API proxy to the browser. The public
  `GET /channels/{slug}/prediction` endpoint and all prediction-specific Go code (domain, port,
  `infra/kick` adapter, `usecase/predictions`, route, tests) are removed. This **supersedes the
  2026-05-26 backend-proxy decision below**.
- The browser calls Kick's public endpoints directly. It validates
  `GET https://kick.com/api/v2/channels/{slug}` once per slug/browser fetch context, then live polling
  calls only `.../predictions/latest`. `features/prediction/kick-prediction-client.ts` validates the
  latest-prediction response shape, normalizes the snake_case payload into the existing `Prediction`
  shape, and derives totals, per-outcome point share, and the winner flag (the same logic the Go use
  case did).
- The frontend contract is unchanged: `getPrediction(slug): Promise<Prediction>` stays the only
  function the page consumes; only its implementation changed. Failures are thrown as `ApiClientError`
  so the page keeps mapping `404` → not-found and anything else → error. Invalid JSON and invalid
  prediction object shapes are treated as malformed non-404 failures rather than rendering blank
  default data.
- Rationale: prediction polling (every 5s) no longer loads the Go API, so prediction is excluded from
  the rate-limit policy (#20), and the API stays focused on logged chat data/admin/analytics.
- Accepted risk: Kick's browser endpoints are undocumented; a CORS/shape/blocking change breaks the
  page with no backend fallback. Mitigation: all direct-Kick logic is isolated behind
  `getPrediction(slug)`, so a backend proxy or feature flag can be restored without UI changes.

## 2026-05-26 (prediction feature)

- Kick prediction data is fetched through a backend proxy (`GET /channels/{slug}/prediction`), not a
  direct browser fetch. A client-side request to `kick.com` hits CORS (no `Access-Control-Allow-Origin`
  for the web origin) and Kick's `Request blocked by security policy`. The backend owns the Kick call,
  browser-like headers, normalization, and error mapping — consistent with the existing `infra/kick`
  channel resolver.
- The prediction endpoint is live and stateless: every `/prediction/{slug}` view fetches the latest
  prediction on demand. No ClickHouse/SQLite tables, no migration, no write path. Persistence was
  explicitly deferred (the observed Kick endpoint exposes only top users per outcome, not full
  voters).
- Totals (total points, total votes), per-outcome point share, and the winner flag are derived in the
  `usecase/predictions` service, not trusted from Kick (the endpoint does not return them).
- `recharts` is adopted as the repo's charting dependency for the donut, grouped-bar, and horizontal
  top-users charts. Landing-page bars stay hand-rolled CSS; recharts is scoped to the prediction
  analysis route.
- Prediction charts use a green/purple pair for stronger contrast on dark panels. The first two
  outcome/series colors are `#22C55E` and `#C084FC`; fallback category colors cycle through white,
  `#FF005C`, `#474f54`, and `#26001B`. State pills map onto existing tokens:
  RESOLVED=accent green (`Sonuçlandı`), LOCKED=warning (`Kilitli`),
  CANCELED/CANCELLED=warning (`İptal`), ACTIVE=neutral (`Aktif`).
- `/prediction/{slug}` polls every 5 seconds after the first successful load and keeps polling while
  the page is open, including terminal states. This captures `LOCKED` → `RESOLVED` transitions that
  can happen after Kick locks voting. Polling is a background refresh that keeps existing content
  mounted to avoid chart flash.
- `/prediction` is search-first and submit-only (same rationale as `/users` and `/channels`):
  submitting navigates to `/prediction/{slug}` rather than rendering analytics in place.
- `SiteHeader` gains a `Prediction` nav link; `ActiveRoute` extended with `"prediction"`.

## 2026-05-24 (issue #15 memory stability)

- `/messages` search must keep ClickHouse wide columns out of the ranking/filtering phase. The
  repository first finds the page of deduped message IDs using narrow columns, then fetches
  `raw_payload_json`, emote arrays, and reply JSON only for those IDs. This keeps public search
  compatible with the 1.2 GiB ClickHouse cap; otherwise channel/date searches can exceed the cap
  before `LIMIT` is applied.
- Production VPS (4 GB RAM) locks up ~24h after boot from host RAM exhaustion → swap thrash, not
  disk. A full reboot restores it; disk-backed Docker volumes survive reboot, so the resettable
  cause is RAM. Remediation is bounding per-process memory, not adding disk.
- ClickHouse is capped with a mounted `clickhouse/config.d/memory.xml` override:
  `max_server_memory_usage` ~1.2 GiB (`1288490188`), `mark_cache_size` 256 MiB, and
  `uncompressed_cache_size` 0. Default `max_server_memory_usage` is ~90% of host RAM (~3.6 GB on a
  4 GB box), which assumes ClickHouse owns the whole machine and starves the Go api, Go listener,
  Next.js web, and the OS. `mark_cache` defaults to 5 GiB and is the largest idle consumer.
- Memory budget split for the 4 GB host (leaves OS headroom): `clickhouse 1.5G`, `web 768M`,
  `listener 512M`, `api 384M`. The ClickHouse `max_server_memory_usage` stays below its container
  `mem_limit` so caches and query memory fit inside the container.
- The `web` service runs a production build (`next start`) instead of `next dev`. `next dev` keeps
  HMR + continuous recompilation resident and grows memory over time, which is a significant
  standalone consumer on a 4 GB box. The Dockerfile is multi-stage (build then production-only
  runtime) and the dev bind mounts are removed so the built artifacts are not shadowed.
- `output: "standalone"` was rejected for now: it fails to build on the Windows dev host (EPERM on
  the symlink step that assembles `.next/standalone`), so it would break the local
  build-before-commit gate. `next start` already eliminates the `next dev` memory growth that
  causes the lockup; standalone is only an image-slimming optimization and can be revisited if the
  build runs on Linux/CI.

## 2026-05-24

- `/channels` and `/users` are search-first index pages with explicit submit. Initial load shows
  an idle prompt; the user types a query and clicks `Ara` or presses Enter to fire the request.
  Auto-search-while-typing was rejected because ClickHouse `LIKE '%…%'` over denormalized columns
  is expensive under heavy live ingestion; firing on every debounced keystroke would degrade the
  whole API under load.
- Submit button requires minimum 2-character trimmed query and is disabled while loading.
- Empty-results message quotes the last submitted query, not the current input value.
- Backend `q=` text search parameter added to `GET /analytics/top-senders` and
  `GET /analytics/top-channels`. The filter applies a LIKE `%…%` WHERE clause on
  `sender_username_lower / sender_slug_lower` (senders) and
  `channel_slug_lower / channel_display_name_lower` (channels) before GROUP BY.
- `AnalyticsFilter.Query` field carries the text search value from route to ClickHouse.
- `SiteHeader` now includes `Channels` and `Users` nav links; `ActiveRoute` extended with
  `"channels" | "users"` values.
- `AnalyticsQueryParams` frontend type now includes `q?: string`; `buildAnalyticsQuery` passes it
  through to the backend.
- Index page result rows link to the existing `/channels/[slug]` and `/users/[slug]` profile pages.
  User slug links convert `_` to `-` (same convention as profile pages).

## 2026-05-22

- User profile identity avatars keep fixed 72px dimensions plus `min-w-[72px]` on both image and
  fallback initials paths. Mobile flex rows must wrap text around the avatar rather than shrink
  the profile photo.
- Admin responsive views intentionally keep desktop and mobile row variants in the DOM behind
  breakpoint classes. Tests should query repeated labels with plural queries when asserting shared
  row data.
- Admin mobile UX uses a hamburger drawer for section navigation, card-style channel rows, stacked
  user rows, and wrapped operations/data-management panels instead of horizontal overflow.

## 2026-05-21

- Frontend v2 design tokens replace the legacy magenta palette repo-wide. The new tokens (Kick
  green accent `#00e701` on dark neutrals, Geist Sans + Geist Mono) live in
  `apps/web/src/app/globals.css` and `apps/web/tailwind.config.ts` and are wired through shadcn
  HSL aliases plus dedicated `bg-page`, `bg-panel`, `bg-elevated`, `border-strong`, `text-faint`,
  `accent`, `accent-hover`, `accent-muted`, `danger`, and `warning` utilities. Legacy `kick-*`
  color tokens are removed; routes not yet re-skinned will pick up the new colors until they get
  their own v2 layout pass.
- Geist fonts ship via the official `geist` npm package; `RootLayout` applies `GeistSans.variable`
  and `GeistMono.variable` so Tailwind `font-sans` / `font-mono` resolve through CSS variables.

## 2026-05-09

- Use a monorepo in `kick-logs`.
- Use Python for backend and listener.
- Use `uv` for Python project/dependency management.
- Use FastAPI for the backend API.
- Use PostgreSQL for persistence.
- Use SQLAlchemy 2.x async ORM with asyncpg for PostgreSQL access.
- Use Alembic for database migrations.
- Use pragmatic clean architecture for backend code.
- Keep domain entities independent from SQLAlchemy, FastAPI, Pydantic, and external clients.
- Use one Python backend package shared by API and listener entrypoints.
- Run PostgreSQL in Docker.
- Use Docker Compose as the default local runtime.
- Use a development Docker stack with hot reload.
- Use Next.js for frontend.
- Use pnpm as frontend package manager.
- Use Tailwind, shadcn/ui, and lucide-react for frontend UI.
- Defer UI implementation until the backend API is working end-to-end.
- Use `docs/design/design.md` as the source of truth for UI/UX decisions.
- Use a fixed dark-only UI theme.
- Use UI palette `#26001B`, `#810034`, `#FF005C`, `#FFF600`, black, and white.
- Prefer `#FFF600` for primary buttons.
- Do not use blur, glow, colored lighting, or atmospheric background effects.
- Keep UI typography compact; do not use landing-page-scale text in app screens.
- Keep button/control corner radii modest for a serious professional feel.
- Use the provided search UI reference as structural guidance only; do not copy the green visual style exactly.
- Design the `/search` screen first and wait for approval before designing admin panel screens.
- Use the user-provided logo asset where a product mark is needed.
- Search results use one shared outer list container with stacked message rows, not per-message modal/card components.
- Sender avatars in search results should be fully circular.
- Emotes should render inline where they appear in message content.
- `/search` is public and does not require login.
- `/admin` is an authenticated backend management dashboard for operational tasks such as managing followed channels.
- Implement admin authentication in MVP; production hardening can be refined later.
- Use Kick web Pusher chat events, not official Kick webhooks, for MVP ingestion.
- Use commit message format `feat(scope): title`.
- Store messages indefinitely.
- Implement full admin login in MVP.
- Seed default super admin with `admin@kicklogs.local` / `admin123`, overridable by env.
- Super admin can create new admin users.
- Use one listener worker for all enabled channels.
- Add date range filters to search.
- Use optional `AND` search filters; sender is exact case-insensitive, while channel and content are case-insensitive contains.
- Store raw Kick payloads and all useful normalized fields.
- Enrich sender profile images through Kick web endpoints when possible.
- Parse `[emote:id:name]` tokens and render image fallback URLs.
- Use `/` for the public landing page, `/search` for public message search, and `/admin` for
  authenticated backend management.
- Allow multi-agent development for non-overlapping work scopes.
- The original MVP used a sequential phase implementation plan; that completed plan now lives in `docs/archive/`.
- Active implementation agents must use the current `docs/implementation_plan.md` and matching active task file.
- Do not add placeholder `web` or `listener` services in Phase 1; add each service only in its owning phase.

## 2026-05-10

- Public `/search` date inputs default to the last 7 days: `Başlangıç` is current local date/time minus 7 days and `Bitiş` is current local date/time. Users can clear date fields to omit date filters.
- MVP started search-first at `/search`; post-MVP work can use `/` for compact landing content.

## 2026-05-12

- Bare `/search` page load does not automatically fetch latest messages; the result area stays idle with `Arama yapmak için yukarıdaki formu kullanın.` until the user submits a search.
- Explicitly submitting the search form with empty filters still fetches latest messages.
- `/search` date inputs stay as local `datetime-local` values in the UI/URL, but API requests convert them to UTC ISO strings; `end` includes the full selected minute.
- `/search` reply rows show the replied-to sender and replied-to message content above the current message in muted gray text.
- Reply rendering uses `message_type === "reply"`, `reply_metadata.original_sender.username`, and `reply_metadata.original_message.content`; long reply previews expose the full original content through a `title` attribute.
- Repository sponsorship uses Buy Me a Coffee account `yavuzselim` through GitHub `FUNDING.yml` and README links.
- The completed MVP implementation plan is archived under `docs/archive/`; active work uses the post-MVP feature plan in `docs/implementation_plan.md`.
- Post-MVP development is split into feature-scoped task files under `docs/tasks/post_mvp_*.md`.
- The selected post-MVP roadmap prioritizes admin operations, search improvements, analytics, landing analytics, user/channel profiles, and admin data management.

## 2026-05-13

- Public `/messages` sender filtering uses case-insensitive exact matching against sender username/slug snapshots; channel and content filters remain case-insensitive contains matching.
- Post-MVP Feature 1 stores listener heartbeat state in PostgreSQL instead of inferring
  liveness from message timestamps, because quiet channels can be healthy but produce no
  messages.
- Admin operations metrics are exposed through `GET /admin/operations/summary` and remain
  authenticated admin-only.
- Post-MVP Feature 2 will render URLs found inside message content as safe clickable links in
  `/search` result rows. Link rendering must not break inline emotes or matched-text
  highlighting.
- `/search` date presets update only the date fields and keep other filters intact.
- `/search` CSV/JSON export actions use the last submitted filters, not unsent form edits.
- `/search` keeps secondary controls compact: quick date ranges are a select, exports sit
  behind one square download icon, and reply/emote filters use explicit `Sadece ...` labels.
- `/search` export menu must close on outside click.
- `/search` keeps date controls on their own row; result-type filters sit to the left of the
  `İşlem` action group so the date row does not feel cramped.
- Analytics endpoints are public read-only contracts under `/analytics/*` for future landing,
  user profile, and channel profile screens.
- Analytics `sender` scope uses case-insensitive exact sender username/slug matching;
  analytics `channel` scope uses case-insensitive exact channel slug/display-name matching.

## 2026-05-14

- Public `/` is a compact landing page, not a redirect. It explains the self-hosted project and
  loads public analytics from Feature 3 endpoints.
- Landing message volume uses a recent day-bucket range, while overview/top-list cards summarize
  current stored data.
- Landing navigation links to `/search`, `/admin`, GitHub, and Buy Me a Coffee support.
- Landing design must stay dark, compact, product-focused, and avoid oversized hero treatment.
- Header brand/logo areas in `/search` and `/admin` navigate to `/`.
- Public user profiles live at `/users/[slug]` and use `GET /users/{slug}/analytics`.
- Search result sender names and avatars link to public user profiles when sender slug exists.
- `/search` reply preview sender names also link to `/users/[slug]`; when Kick reply metadata has
  no slug, the frontend derives a lowercase username fallback.
- `/users/[slug]` top identity blocks use the same rounded bordered panel treatment as the rest of
  the profile sections.
- Public sender profile URLs follow Kick's profile slug behavior: chat usernames can display with
  underscores, but profile routes convert `_` to `-`; backend profile/search lookups accept both
  forms so existing underscore-stored data keeps working.

## 2026-05-20

- Issue #9 phase 7 ships the synthetic ingestion load harness at `apps/api-go/cmd/loadgen` and
  a runbook at `docs/operations/load_test.md`. The harness reuses the production listener
  service wired against the real SQLite/ClickHouse stack, but replaces the Pusher client with a
  deterministic emitter, so it exercises buffered writer + worker batch + breaker on the live
  ingestion path. External durable queues (RabbitMQ/NATS/Kafka) remain explicitly deferred
  until the live load run confirms in-process changes are insufficient.
- Issue #9 phase 6 surfaces ingestion health on the admin operations summary. The listener
  heartbeat now embeds buffered-writer stats and circuit-breaker state in its metadata JSON; the
  API operations repository reads SQLite `raw_event_queue` for live queue depth and oldest
  pending age, and parses the heartbeat metadata to populate the new `ingestion` block on
  `GET /admin/operations/summary`. The `/admin` operations dashboard renders queue backlog,
  writer buffer, ClickHouse breaker state, and last flush cards plus warnings for an open
  breaker or non-zero writer drop count.
- Issue #9 phase 5 wraps ClickHouse access in the listener with bounded exponential backoff and
  a single shared `CircuitBreaker`. The breaker opens after a configurable consecutive failure
  threshold and holds the open state for the current backoff delay before allowing a probe call.
  All listener goroutines (buffered writer flush, worker batch insert) share the same breaker
  so opening it in one path pauses the others.
- Backoff defaults: 1s initial, 30s max, multiplier 2, full jitter (`d/2` to `3d/2`). Threshold
  default: 5 consecutive failures. Env knobs: `LISTENER_CLICKHOUSE_BACKOFF_INITIAL_MS`,
  `LISTENER_CLICKHOUSE_BACKOFF_MAX_MS`, `LISTENER_CLICKHOUSE_BACKOFF_MULTIPLIER`,
  `LISTENER_CLICKHOUSE_BREAKER_FAILURE_THRESHOLD`.
- Workers and the buffered writer call `breaker.Wait(ctx)` before each ClickHouse operation and
  call `RecordSuccess`/`RecordFailure` after. The worker also calls `Wait` before claiming and
  reading the queue, so the entire worker loop honors the breaker delay during an outage.
- Issue #9 phase 4 batches worker output writes. A single worker tick now claims pending queue
  rows, loads raw payloads through `RawEventRepository.GetByID`, normalizes them in memory, then
  writes one `InsertMessagesBatch` and one `InsertAttemptsBatch` per tick. Successful items are
  marked processed in the SQLite queue; failing items are marked failed with the cause.
- Tick-internal dedupe keeps a `seenKickMessageIDs` set so two queue rows carrying the same
  `kick_message_id` do not both insert a visible message in the same batch. ClickHouse
  `ExistsByKickMessageID` continues to dedupe across ticks.
- ClickHouse `InsertMessagesBatch` failure during a tick releases every claimed queue row back
  to pending so the next tick retries the same set. The attempt batch insert is best-effort; a
  failure logs an error but does not roll back the message batch because audit attempts are
  recoverable on the next attempt.
- Issue #9 phase 3 introduces a buffered websocket writer. The websocket callback now submits
  raw events to an in-memory channel; a dedicated goroutine flushes batches to ClickHouse using
  the phase 2 batch insert and then enqueues the same batch into the SQLite work queue. Batch
  size, flush interval, in-memory queue size, and max retries are env-tunable via
  `LISTENER_RAW_EVENT_WRITE_BATCH_SIZE` (default 500),
  `LISTENER_RAW_EVENT_WRITE_FLUSH_INTERVAL_MS` (default 500),
  `LISTENER_RAW_EVENT_WRITE_QUEUE_SIZE` (default 50000), and
  `LISTENER_RAW_EVENT_WRITE_MAX_RETRIES` (default 10).
- When the in-memory buffer is full the writer drops the oldest event, increments a counter,
  and logs a warning. Phase 6 will surface drop counters through the operations dashboard.
- ClickHouse batch failures retry with bounded exponential delay up to `MaxRetries`; the batch
  is dropped with a metric counter only after all retries fail. SQLite enqueue failures after a
  successful ClickHouse archive retry indefinitely until the SQLite queue accepts the rows so
  archived events do not silently fall out of the work queue.
- Issue #9 work moves the listener raw-event work queue out of ClickHouse into a new SQLite
  `raw_event_queue` table. ClickHouse keeps the archive role for `raw_kick_events`,
  `chat_messages`, and `raw_event_attempts`; SQLite owns pending list, attempts, claim
  ownership, and last error.
- Worker pending count and listing are read from SQLite, removing the heavy
  `raw_event_attempts` GROUP BY + LEFT JOIN query from the hot path.
- The websocket callback enqueues into SQLite in the same logical unit as the ClickHouse archive
  insert; failure to enqueue is treated as a callback error so the websocket does not silently
  drop the event.
- Workers fetch raw payloads from ClickHouse by id via `RawEventRepository.GetByID` and keep
  using the existing `markRawEventProcessed`/`markRawEventFailed` helpers for per-row CH attempt
  audit; batching of those writes is deferred to phase 4 of the issue #9 plan.
- Stale `claimed` rows older than `RawEventProcessingTimeout` are reset to `pending` at startup
  and on a periodic background loop, so workers that die mid-claim do not block the queue.

## 2026-05-16

- Go rewrite work lives under `apps/api-go`; after cutover the Go API/listener is the default
  runtime and Python is reference-only.
- Phase 2 Go workspace uses the Go standard library for the initial HTTP server, routing,
  middleware, config, and logging to avoid unnecessary early dependencies.
- Default Compose service `api` runs the Go API and maps to host `API_PORT` or `8000`.
- Go local build outputs and caches stay untracked and outside Docker build context under
  `apps/api-go/bin/`, `apps/api-go/.gocache/`, `apps/api-go/.gomodcache/`, and
  `apps/api-go/.cache/`.
- The Go rewrite uses SQLite for control-plane state and ClickHouse for message/raw-event data.
- SQLite stores admin users, followed channels, sender profile cache, retention settings, worker
  heartbeats, and migration bookkeeping.
- ClickHouse stores denormalized chat messages, raw Kick events, and raw-event processing attempts.
- Go rewrite migrations are run through `cmd/migrate`; Compose exposes that binary as the
  `migrate-go` service behind profile `tools`.
- Go rewrite default super-admin seeding happens in SQLite migration startup and stores a bcrypt
  hash, not the plain password.
- Go rewrite auth preserves the Python cookie contract and uses HS256 JWTs with `sub`, `iat`, and
  `exp` claims.
- Go rewrite API startup may apply SQLite and ClickHouse migrations for local developer ergonomics;
  `migrate-go` remains the explicit migration command for Compose setup.
- Go rewrite admin channel deletion remains disable-only to preserve historical chat data.
- Go rewrite public message search/export reads denormalized ClickHouse `chat_messages` directly;
  the hot search path must not join back to SQLite.
- Default Compose service `listener` runs the Go listener.
- Go listener ingestion keeps the durable-inbox rule: once a Kick websocket chat event reaches the
  process, persist the raw event to ClickHouse before normalization, sender enrichment, or visible
  message insertion.
- Go raw-event processing is at-least-once and idempotent: retries append attempt rows, while
  visible message writes dedupe by `kick_message_id`.
- Go listener heartbeat state remains in SQLite `worker_heartbeats` so operations health can be
  read without Docker logs.
- Go analytics/profile endpoints are public and keep reading denormalized ClickHouse
  `chat_messages`; they must not add hot-path joins back to SQLite for aggregate lists.
- Go user/channel profile identity comes from SQLite metadata, while profile analytics and latest
  messages come from ClickHouse.
- Go PostgreSQL data migration is one-way and read-only against PostgreSQL. It writes SQLite and
  ClickHouse, preserves source IDs where API rows expose IDs, validates counts/samples after
  execute, and records run metadata in SQLite `data_migration_runs`.
- Go migration rejects admin password hashes that `golang.org/x/crypto/bcrypt` cannot parse; it
  must not silently reset migrated credentials.
- Cutover keeps the service names `api` and `listener`, but those default services now point to
  the Go binaries from `apps/api-go`.
- ClickHouse is part of the default Compose runtime; PostgreSQL is not.
- SQLite control-plane data is stored in the `api_go_data` Docker volume.
- During cutover, Python/FastAPI/PostgreSQL stayed temporarily behind an explicit reference profile
  for rollback.
- `migrate-go` is a `tools` profile service, not a default runtime service.
- PostgreSQL volumes must not be removed automatically during cutover.
- Python source remained in the repository until the final cleanup decision.
- SQLite sender profile ingestion must tolerate concurrent live messages from the same sender; Go
  listener upsert uses `ON CONFLICT` for both `kick_user_id` and `slug` instead of pre-read then
  insert.
- Superseding the earlier cleanup decision: Python/FastAPI source is now removed from the repo.
- PostgreSQL is no longer a Compose service. Legacy PostgreSQL data can be imported only by running
  `migrate-go` with an explicit external/restored `POSTGRES_SOURCE_DSN`.
- The old PostgreSQL Docker volume is not removed automatically and was intentionally preserved
  during cleanup.
- Go CI is the backend validation source of truth. It runs on every push and pull request, along
  with the repository code-style workflow.
- Completed Go rewrite plan, task files, and API contract inventory are historical context under
  `docs/archive/go_rewrite/`.
