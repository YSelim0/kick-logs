# Change Log

This is a living implementation log. Add new entries for each meaningful project change.

## 2026-06-02 (issue #23 — JetStream durable ingestion plan)

- Replaced the active issue #23 plan with a NATS JetStream durable ingestion plan.
- New target pipeline: `listener -> NATS JetStream -> processor -> ClickHouse`.
- Locked the core rule that reached Kick chat events must not be silently dropped by memory pressure,
  retry exhaustion, or planned reconnects.
- Documented that SQLite returns to control-plane ownership only after the cutover; old raw queue
  tables become legacy/migration state.
- Planned feature-sized phases for NATS foundation, raw event envelopes, listener publishing,
  processor workers, ClickHouse idempotency, old hot-path removal, operations visibility, and final
  verification.

## 2026-06-02 (issue #23 — NATS foundation)

- Added the NATS Go client dependency and a new `infra/natsstream` package using the current
  `jetstream` API.
- Added raw event stream domain/port types for publish, fetch, ack/nak/term, and stats without
  coupling use cases to NATS-specific types.
- Added config/env/Compose wiring for the JetStream raw event stream, subject, durable consumer, ack
  wait, fetch batch size, and fetch timeout.
- Added a `nats` Compose service with JetStream enabled and a persistent `nats_data` volume.
- Added focused tests for NATS config defaults/overrides and durable stream/consumer settings.

## 2026-06-02 (issue #23 — raw capture before normalization)

- Changed the Kick Pusher chat parser so `App\Events\ChatMessageEvent` payloads no longer need all
  normalized message fields before capture.
- Incomplete chat payloads are now accepted as raw chat events and can be classified later as
  terminal ignored/invalid by processing logic.
- Raw event ids now use a deterministic `kick:{message_id}` identity when Kick message id exists,
  with a hashed raw fallback for malformed/incomplete events without a message id.
- Listener capture now derives `chatroom_id` from the Pusher channel name when the payload omits it,
  which preserves channel metadata for incomplete events.

## 2026-06-02 (issue #23 — listener publishes to JetStream)

- Added a raw-event stream publisher dependency to the listener service.
- The live listener path now serializes the raw chat envelope and waits for JetStream PubAck before
  incrementing the captured-event counter.
- Publish failure now surfaces as a listener error instead of being counted as a successful capture.
- When a stream publisher is configured, the listener disables the legacy buffered writer, SQLite
  queue bootstrap, stale-claim recovery, and raw-event worker goroutines.
- The listener binary now opens NATS JetStream and no longer opens ClickHouse, so ClickHouse write
  pressure cannot stop the websocket capture service from publishing reached events into the
  durable stream.
- Docker Compose no longer makes the listener depend on ClickHouse health; listener depends on NATS
  and SQLite control-plane state only.

## 2026-06-02 (issue #23 — JetStream processor service)

- Added a `StreamProcessorService` that pulls raw chat events from the durable JetStream consumer.
- Processor batches ClickHouse writes in this order: raw archive rows, normalized `chat_messages`,
  and raw-event attempt audit rows.
- Processor ACKs stream messages only after required ClickHouse writes succeed.
- Terminal invalid payloads are recorded as ignored attempts and then terminated in JetStream.
- Transient ClickHouse failures NACK fetched messages so JetStream can redeliver them.
- Added `cmd/processor` and a default Docker Compose `processor` service.
- Added tests for successful ACK, transient NACK, and terminal invalid TERM behavior.

## 2026-06-01 (issue #23 — storage hot path hardening plan)

- Replaced the active implementation plan with the storage hot-path hardening plan for issue #23.
- Documented the production-safe target split:
  - ClickHouse remains the durable data-plane store for chat messages, raw Kick events, processing
    attempts, and subscription periods.
  - SQLite remains the control-plane store plus temporary queue/inbox state.
- Planned feature-sized implementation phases for:
  - best-effort sender profile cache writes
  - sender profile write throttling
  - deleting processed raw-event queue rows
  - terminal invalid raw-event handling
  - webhook inbox retention
  - admin operations clarification

## 2026-06-01 (issue #23 — sender profile cache best-effort)

- Changed listener raw-event processing so SQLite `sender_profiles` upsert failures no longer fail
  chat message normalization.
- The listener now logs sender cache upsert errors and continues with the sender snapshot from the
  raw Kick payload.
- Added an in-memory sender profile write gate so the listener attempts at most one cache upsert per
  Kick user id every 10 minutes.
- Added coverage proving a sender profile cache failure still produces a visible chat message and a
  processed raw-event attempt, plus coverage for repeated sender messages avoiding repeated cache
  writes.

## 2026-06-01 (issue #23 — raw event queue processed-row pruning)

- Changed SQLite `raw_event_queue.MarkProcessed` to delete processed queue rows instead of keeping a
  permanent `processed` row in SQLite.
- Added SQLite migration v8 to prune existing `processed` queue rows on deploy.
- Updated listener and SQLite queue tests so successful processing leaves no active queue row.
- Added coverage proving startup queue bootstrap skips raw events that already have a ClickHouse
  `processed` attempt, which prevents re-enqueue loops after queue pruning.

## 2026-06-01 (issue #23 — terminal invalid raw events)

- Added terminal invalid raw-event handling for permanently malformed payloads. These events now
  write ClickHouse attempts with status `ignored` and are removed from active SQLite queue work
  instead of retrying until max attempts.
- Updated raw-event backfill queries to treat `processed`, `ignored`, and `invalid` attempts as
  terminal statuses.
- Changed worker behavior so a failed `raw_event_attempts` batch insert releases queue claims back
  to pending instead of acknowledging/deleting queue rows without ClickHouse attempt history.
- Operations raw-event status counts now expose `ignored` and exclude terminal ignored/invalid rows
  from pending/failed calculations.

## 2026-06-01 (issue #23 — webhook inbox retention)

- Added `KickWebhookEventRepository.PruneTerminalBefore`.
- SQLite webhook inbox pruning deletes only `processed` and `ignored` rows whose `processed_at` is
  older than the retention cutoff.
- Webhook processor now prunes terminal inbox rows older than the default 7-day retention window
  after each processing tick.
- Added repository and processor tests proving processed/ignored old rows are pruned while failed
  and recent terminal rows remain.

## 2026-06-01 (issue #23 — admin operations clarification)

- Updated Operations dashboard copy so ClickHouse raw-event history is not presented as active
  SQLite queue backlog.
- The active listener backlog is labeled as `Aktif queue` under the Ingestion section, while the
  Raw Event metric is described as ClickHouse history.
- Failed raw-event modal copy now states that it shows retryable failed events and excludes terminal
  ignored events.
- Data Management table summaries now include `raw_event_queue`, `kick_webhook_events`,
  `kick_event_subscriptions`, and `schema_migrations` row counts so temporary/runtime SQLite state
  remains visible after processed rows are deleted.

## 2026-06-01 (issue #23 — docs and verification)

- Updated `docs/architecture.md` and `docs/project_plan.md` so the current storage split is explicit:
  ClickHouse owns durable data-plane history, while SQLite owns control-plane state and temporary
  active queue/inbox rows.
- Marked `docs/implementation_plan.md` as implemented on
  `feat/issue-23-storage-hot-path-hardening`.
- Refreshed context handoff docs for the completed storage hot-path hardening branch.

## 2026-06-01 (issue #23 — SQLite lock follow-up)

- Fixed admin Data Management summary returning 500 under concurrent SQLite writer pressure.
- `GetRetentionSettings` now reads existing settings first and only writes the default row when no
  row exists, keeping `/admin/data-management/summary` read-only after initial setup.
- Webhook processor terminal inbox pruning is throttled to once per hour instead of attempting a
  SQLite `DELETE` every 5-second processor tick. Failed prune attempts also wait for the next
  interval before retrying, avoiding repeated `SQLITE_BUSY` warnings.
- Added regression tests for Data Management summary during a concurrent SQLite write lock and for
  webhook prune throttling.

## 2026-06-01 (admin webhook status UI)

- Updated the Operations Webhooks panel so channel subscription health is summarized per channel
  instead of rendering one table row per Kick event subscription.
- Added three distinct channel summary states: `aktif`, `Aktif değil`, and `N Hata`. Missing or
  inactive event subscriptions without sync errors are no longer labeled as errors.
- Added a clickable details modal that shows channel metadata and each configured event's
  `aktif` / `aktif değil` / `hata` state, including sync error text when present.
- Updated frontend tests for active, inactive, and error webhook subscription states.

## 2026-06-01 (issue #22 — webhook sync contract fix)

- Fixed channel subscription summary counts. ClickHouse `countDistinctIf(...)` returns `UInt64`;
  scanning directly into Go `int64` caused the repository to error, and the route returned zero
  counts. `SubscriptionPeriodRepository.ActiveSummary` now scans unsigned counts and converts them
  safely. Verified `/channels/levo/subscription-summary` returns `active_count: 1` for the processed
  renewal webhook.
- Corrected the Kick event subscription client to match the current public API:
  - create uses batch `events: [{name, version: 1}]` and `method: webhook`
  - delete uses `DELETE /public/v1/events/subscriptions?id=<subscription_id>`
  - webhook public key auto-fetch uses `GET /public/v1/public-key`
- Corrected webhook signature verification from the earlier Ed25519 assumption to RSA-SHA256 over
  `message_id + "." + timestamp + "." + raw_body`.
- Reworked sync to create missing events per channel in one request, reconcile ambiguous responses
  with list-after-create, and clear previous error records once Kick reports the subscription active.
- Added processor protection for disabled channels so stale remote subscriptions cannot create active
  subscription periods locally.
- Added tests for the Kick event-subscription HTTP contract, RSA webhook signatures, batch sync, and
  disabled-channel webhook ignores.
- Verified locally: `go test ./...`, `go vet ./...`, Docker API rebuild, startup sync, manual sync,
  and `/admin/webhooks/health` for active channels.

## 2026-06-01 (issue #22 — Kick webhook subscription tracking, Phases 1–7)

Backend pipeline for tracking Kick subscription events via webhooks. Phase 8 (frontend) deferred.

- **Phase 2 — Storage foundation**: `followed_channels.broadcaster_user_id`; SQLite tables
  `kick_webhook_events` (inbox) and `kick_event_subscriptions` (registry); ClickHouse table
  `channel_subscription_periods` (`ReplacingMergeTree` ORDER BY deterministic `id`); all port
  interfaces; full repository test coverage.
- **Phase 3 — Kick API client and sync**: `EventSubscriptionClient` (OAuth2 client credentials,
  token cache, broadcaster_user_id resolve, event sub CRUD via `api.kick.com`); `kicksync.Service`
  (`SyncAll`, `EnsureChannelSubscriptions`, `RemoveChannelSubscriptions`); startup background sync;
  channel add/disable triggers goroutine sync; 5 unit tests.
- **Phase 4 — Webhook receiver**: `POST /webhooks/kick`; RSA-SHA256 signature verification
  (`infra/kick/WebhookVerifier`, PEM/base64 public key formats); `INSERT OR IGNORE` idempotency;
  fail-closed (503 with no key, 401 on bad sig); rate-limit exempt; 8 route tests.
- **Phase 5 — Processor and normalization**: `webhookprocessor.Service` (5s tick, background
  worker); normalizer handles `channel.subscription.new/renewal` (1 period) and
  `channel.subscription.gifts` (1 period/giftee); `expires_at` fallback `created_at + 30d`;
  `ErrIgnored` for unsupported/unfollowed events; 13 tests.
- **Phase 6 — Backend query APIs**: `GET /channels/{slug}/subscription-summary` (public active
  count); `GET /admin/webhooks/health` (inbox counts, sync status, config flags);
  `POST /admin/webhooks/sync` (manual sync trigger); auth-gated admin endpoints.
- **Phase 7 — Docs and smoke**: context, architecture, decisions, change log, operations runbook
  updated; `go test ./...`, `go vet ./...`, `pnpm format:check` verified green.

## 2026-05-31 (README refresh)

- Rewrote the root README as a shorter, product-focused public repo page:
  - added a compact hero with logo + title, slogan, demo GIF, badges, and repository links
  - removed the long API endpoint catalogue and most low-level operational detail
  - kept the project explanation, feature summary, contribution flow, and short self-host commands

## 2026-05-30 (client-side prediction review fixes)

- Hardened the client-side Kick prediction fetcher after review:
  - channel existence validation is cached per slug/browser fetch context, so after the first
    validation the 5-second refresh loop calls only `.../predictions/latest`
  - latest-prediction JSON is now runtime-validated before normalization; missing or wrong-typed
    required prediction/outcome/top-user fields throw `ApiClientError(502)` instead of rendering
    blank default data
  - added coverage for cached channel validation and malformed prediction/outcome shapes
- Refreshed prediction docs/context to state that prediction is browser-side, that the Go API no
  longer serves `GET /channels/{slug}/prediction`, and that malformed prediction shapes map to the
  error state.

## 2026-05-26 (prediction active polling and profile polish)

- `/prediction/{slug}` now refreshes the latest prediction every 5 seconds after the first
  successful load and keeps polling while the page is open, including after terminal states so
  `LOCKED` → `RESOLVED` transitions are still captured.
- Background refreshes are guarded against overlapping requests and keep the existing content
  mounted, avoiding chart/list flash during live updates.
- `predictionStateBadge` now labels both `CANCELED` and `CANCELLED` as `İptal`.
- Prediction chart containers now set positive Recharts initial dimensions and `min-w-0` wrappers
  to prevent `width(-1) and height(-1)` console warnings while the first ResizeObserver measurement
  is pending.
- Outcome cards now show an option-colored point-share progress bar between stats and top users,
  mute losing cards after a winner exists, and add a subtle hover background to top-user rows.
- Channel profile identity no longer renders internal channel/chatroom ids. User profile identity no
  longer renders the channel-count fragment before first/latest activity.

## 2026-05-26 (mobile header + active-route + text-visibility fixes)

- `site-header.tsx` → client component with hamburger menu for mobile. All nav links + Admin in dropdown panel below header. Desktop layout unchanged.
- `channel-profile-page.tsx` `activeRoute` fix: `"search"` → `"channels"`.
- `user-profile-page.tsx` `activeRoute` fix: `"search"` → `"users"`.
- Replaced all `text-secondary` usages with `text-muted-foreground` in channel/user profile and prediction analysis pages. Bug: `text-secondary` resolves to `#24272c` (background color) in the Tailwind config, not `#9ca3af` text color.
- `design.md` and `recent_changes.md` updated.
- Verification: lint, typecheck, 96 tests, build green.

## 2026-05-26 (prediction feature)

- Backend: added public `GET /channels/{slug}/prediction`.
  - `domain.Prediction`/`PredictionOutcome`/`PredictionTopUser`, `ports.PredictionFetcher` with
    `ErrPredictionNotFound` / `ErrPredictionChannelNotFound` / `ErrPredictionBlocked`.
  - `infra/kick/prediction_resolver.go` fetches Kick's undocumented
    `/api/v2/channels/{slug}/predictions/latest` with browser-like headers (user-agent, referer,
    accept-language, sec-ch-ua, sec-fetch-\*); maps 404 → channel-not-found, non-2xx / top-level
    `error` body → blocked, null prediction → not-found.
  - `usecase/predictions` validates the slug and derives total points/votes, per-outcome point
    share, and the winner flag.
  - Route maps errors to 404 (no prediction / channel) and 502 (blocked); wired through
    `routes.Dependencies`, `server.go`, and `cmd/api/main.go` (independent of ClickHouse).
  - Tests: `usecase/predictions` normalization + `internal/http` route success/404/404/502 cases.
- Frontend: added `/prediction` (search-first, submit navigates to `/prediction/{slug}`) and
  `/prediction/{slug}` (live fetch → summary card, `recharts` donut + grouped-bar + horizontal
  top-users charts, outcome cards with `KAZANAN` winner badge, loading/not-found/error states,
  refresh button).
  - Added `recharts` dependency; `Prediction` header nav link; `ActiveRoute` `"prediction"`;
    `Prediction*` response types; `features/prediction/` api + format helpers + components + tests.
  - Refined the analysis layout so outcome cards render directly after the summary and before the
    charts, with two equal-width columns on tablet/desktop.
  - Updated prediction chart colors to `#22C55E` + `#C084FC`, gave vote count and return rate
    separate Y axes, and moved legends outside fixed-height chart containers to avoid overflow.
- Docs: design, architecture, living brain, decisions, change log, recent changes updated.
- Verification: `go build/vet/test ./...` green; web `typecheck`/`lint`/`test` (20 files, 96 tests)
  /`build` green; `pnpm format:check` green.

## 2026-05-24 (issue #15 memory stability)

- Fixed `/messages` search under the new ClickHouse memory cap:
  - The old query selected wide response columns, including `raw_payload_json`, inside the same
    windowed `row_number()` subquery used for deduplication. With the 1.2 GiB ClickHouse cap,
    channel/date searches over a large week could hit `MEMORY_LIMIT_EXCEEDED` while reading
    `raw_payload_json` before the final `LIMIT 50`.
  - `MessageRepository.Search` now performs a narrow first query that applies filters and ranks only
    `id`, `kick_message_id`, timestamps, and deletion state, then fetches the full response columns
    only for the selected page of IDs.
  - Verified `go test ./...` from `apps/api-go`.
  - Rebuilt the local API container and verified the failing request now returns 200:
    `GET /messages?limit=50&channel=eray&start=2026-05-17T20:28:00.000Z&end=2026-05-24T20:28:59.999Z`.

- Capped ClickHouse server memory for the 4 GB production VPS:
  - Added `clickhouse/config.d/memory.xml` mounted read-only at
    `/etc/clickhouse-server/config.d/`: `max_server_memory_usage` 1288490188 (~1.2 GiB),
    `mark_cache_size` 268435456 (256 MiB), `uncompressed_cache_size` 0.
  - Added `mem_limit: 1536m` to the `clickhouse` service in `compose.yaml`.
  - Validated with `docker compose config --quiet`.
  - Fix: mount the override as a single file
    (`./clickhouse/config.d/memory.xml:/etc/clickhouse-server/config.d/memory.xml:ro`), not the
    whole `config.d` directory. A directory bind mount shadowed the image's own config.d files
    (including the one that sets `listen_host 0.0.0.0`), so ClickHouse only bound localhost — the
    healthcheck (local client) still passed, but the `api` and `listener` containers got
    `connection refused` on `clickhouse:9000` (listener crash-looped, analytics returned 500).
    Verified after the fix: listener connects and processes batches, `GET /health` and
    `GET /analytics/overview` return 200.

- Added memory limits and a restart policy to every long-running service in `compose.yaml`:
  - `restart: unless-stopped` on `clickhouse`, `api`, `listener`, `web` so a service that
    exceeds its limit is OOM-killed and restarted instead of taking down the host.
  - `mem_limit`: `clickhouse 1536m`, `web 768m`, `listener 512m`, `api 384m` (total < 4 GB with
    OS headroom). `migrate-go` (one-shot tools profile, run with `--rm`) is left unchanged.
  - Validated with `docker compose config --quiet`.

- Switched the `web` service from Next.js dev mode to a production build:
  - `apps/web/Dockerfile` is now multi-stage: a build stage installs the full toolchain and runs
    `pnpm build`; a runtime stage installs production-only dependencies and copies the compiled
    `.next`, `public`, and `next.config.mjs`, then runs `pnpm start` (`next start`).
  - `NEXT_PUBLIC_API_BASE_URL` is passed as a build `ARG`/`ENV` (and wired through
    `web.build.args` in `compose.yaml`) because `NEXT_PUBLIC_*` values are inlined into the client
    bundle at build time.
  - Removed the dev bind mounts (`./apps/web`, `web_node_modules`, `web_app_node_modules`,
    `web_next`) and dropped the now-unused volume definitions; removed the redundant runtime
    `NEXT_PUBLIC_API_BASE_URL` env.
  - Standalone output was evaluated but not adopted: `output: "standalone"` fails to build on the
    Windows dev host (EPERM on the symlink step). `next start` already removes the heavy
    `next dev` HMR/recompilation memory growth that was the actual fix.
  - Verified with `pnpm --filter @kick-logs/web build` and `docker compose build web`.

- Set `GOMEMLIMIT` on the Go services in `compose.yaml`:
  - `api`: `GOMEMLIMIT=307MiB` (~80% of its 384m `mem_limit`).
  - `listener`: `GOMEMLIMIT=410MiB` (~80% of its 512m `mem_limit`).
  - The Go runtime reads `GOMEMLIMIT` directly, so this disciplines the GC to reclaim before the
    container hits its hard limit, bounding heap spikes. Hardcoded (not an overridable env var)
    because the value is coupled to each service's `mem_limit`.
  - Validated with `docker compose config --quiet`.

- Tuned ingestion batching to reduce ClickHouse part pressure (env defaults in `compose.yaml`
  listener + `.env.example`):
  - Stage 1 raw writer: `LISTENER_RAW_EVENT_WRITE_BATCH_SIZE` 500 → 1000 and
    `LISTENER_RAW_EVENT_WRITE_FLUSH_INTERVAL_MS` 500 → 1500. Larger/less-frequent
    `raw_kick_events` inserts create fewer small parts → fewer background merges → less merge RAM.
    Flush kept at 1.5s (not higher) so the crash-loss window stays small with the 50k in-memory
    queue.
  - Stage 2 normalize: `LISTENER_RAW_EVENT_BATCH_SIZE` 100 → 500 so each worker tick inserts more
    `chat_messages` rows per batch → fewer parts there too.
  - `LISTENER_WORKER_COUNT` left at 4; reducing it would slow normalize and risk SQLite queue
    backlog. The durable SQLite queue means delayed normalization loses no data.
  - Validated with `docker compose config --quiet`.

- Added `docs/operations/vps_memory.md`: 4 GB memory budget table, the swap safety-net commands
  (P2 backstop), and a post-deploy verification checklist (`docker stats`, `free -m`, ClickHouse
  `system.metrics`/`parts`/`merges`).

Note: issue #15 P1 SQLite `raw_event_queue` pruning (delete-on-processed vs periodic cleanup) is
intentionally deferred and tracked separately for a later discussion; not part of this branch.

## 2026-05-24

- Channels/users index pages switched from debounced auto-search to explicit submit:
  - Removed debounce `useEffect` and `setTimeout` plumbing from `ChannelsIndexPage` and
    `UsersIndexPage`.
  - Added `<form onSubmit>` wrapping the input + a primary `Ara` submit button (accent green,
    `text-on-accent`). Button text becomes `Aranıyor…` while a request is in flight.
  - Added `submittedQuery` state so the empty-results message quotes the last submitted query
    rather than the current input value.
  - Added `MIN_QUERY_LENGTH = 2` validation; submit button is disabled below threshold.
  - Updated idle prompt copy: `Kanal/Kullanıcı adı veya slug girin ve Ara butonuna basın.`
  - Updated tests for both pages (11 channels + 12 users tests).
  - Updated `docs/design/design.md`, `docs/context/living_brain.md`, `docs/context/decisions.md`,
    `docs/context/recent_changes.md` to reflect the submit-only UX and the rationale
    (ClickHouse `LIKE '%…%'` cost under live ingestion load).
- Verification:
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web test`: 18 files, 89 tests passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

- Channels and users search index pages:
  - Added `Query string` field to `domain.AnalyticsFilter`.
  - Added `topSendersWhere` and `topChannelsWhere` helpers in ClickHouse analytics repository;
    both apply `LIKE '%…%'` WHERE clauses on denormalized lower-cased columns when `filter.Query`
    is set.
  - `parseAnalyticsFilter` in the HTTP analytics route reads `q=` query param and assigns it to
    `filter.Query`.
  - Added `analytics_q_param_test.go`: four Go HTTP tests verifying `q=` param is parsed and
    forwarded to the analytics repository for top-senders and top-channels endpoints.
  - Frontend `AnalyticsQueryParams` type extended with `q?: string`; `buildAnalyticsQuery` passes
    it to the backend.
  - New Next.js route pages: `app/channels/page.tsx` and `app/users/page.tsx`.
  - New feature components: `ChannelsIndexPage` and `UsersIndexPage` — search-first with 300ms
    debounce, idle/loading/empty/error states, v2 design tokens, profile page links.
  - `SiteHeader` nav updated: `Channels` and `Users` links added; `ActiveRoute` extended.
  - Test files committed: `channels-index-page.test.tsx`, `users-index-page.test.tsx`.
  - `@testing-library/user-event` added as a dev dependency.
  - Context and design docs updated.
- Verification:
  - `go test ./...`: passed
  - `go vet ./...`: passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web test`: 18 files, 83 tests passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

## 2026-05-22

- Responsive mobile polish follow-up:
  - added `min-w-[72px]` to the user profile image path as well as the fallback initials avatar
    so the circular profile photo keeps its intended 72px width on mobile flex layouts
  - kept the fallback initials avatar at the same fixed/min-width dimensions for parity with
    image-backed profiles
  - updated channel/user admin tests to expect the responsive desktop + mobile row variants that
    intentionally duplicate visible labels in the DOM behind breakpoint classes
  - refreshed context docs after recent responsive commits:
    - profile latest-message rows were tightened for mobile wrapping
    - admin sidebar now collapses into a mobile hamburger drawer
    - channel admin uses a mobile card layout
    - user admin uses mobile stacked rows
    - operations dashboard and data-management panels now wrap controls/metrics for smaller
      screens
- Verification:
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web test`: 16 files, 70 tests passed

## 2026-05-21

- Frontend v2 re-skin: user and channel profiles migrated to the v2 designs.
  - rebuilt `apps/web/src/features/user-profile/user-profile-page.tsx` against `ksyyS`
    (`User Profile / v2`) with shared v2 chrome, breadcrumb, identity panel, stats bar,
    3-column analytics grid, top channels/emotes, and dense latest-message rows with inline
    emotes and reply context chips
  - rebuilt `apps/web/src/features/channel-profile/channel-profile-page.tsx` against `WGYFT`
    (`Channel Profile / v2`) with shared v2 chrome, channel identity panel, `LOGGING` pill,
    stats bar, top users/emotes, and sender-color latest-message rows
  - aligned user-profile reply preview chips with `/search` so the `↳` marker appears and
    replied-to sender links stay readable without hover
  - updated profile tests for the v2 labels, CTAs, and link expectations
  - ignored `docs/design/screens/` so exported design/reference screenshots stay out of commits
  - updated implementation progress and living context for the completed `/search`, `/users/[slug]`,
    and `/channels/[slug]` v2 passes; `/admin` and `/login` remain
- Verification:
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web test`: 16 files, 70 tests passed
  - `git diff --check`: passed
  - `pnpm --filter @kick-logs/web lint`: initially failed on unused chart summary plumbing, fixed
    in the working tree

- Frontend v2 re-skin: landing page (`/`) migrated to the v2 design.
  - replaced the legacy magenta palette in `apps/web/src/app/globals.css` and
    `apps/web/tailwind.config.ts` with the v2 token set (Kick green `#00e701`, dark neutrals,
    border/text tiers, danger/warning) and removed all `kick-*` color tokens
  - added Geist Sans + Geist Mono via the `geist` npm package and wired the variables through
    `RootLayout` so Tailwind `font-sans` / `font-mono` resolve to them
  - rebuilt `apps/web/src/features/landing/landing-page.tsx` to match `mRzu8` (`Landing / v2`):
    compact hero with pill badge + 48px title + description + `Arama başlat`/`GitHub` CTAs,
    a 4-cell rounded `border` stats bar (TOPLAM MESAJ / KANAL / KULLANICI / EMOTE), and a 2×2
    analytics panel grid containing the 14-day message volume bar chart, top channels, top
    users, and top emotes lists
  - extracted the v2 global header into `apps/web/src/components/site-header.tsx` so future
    `/search` and `/admin` v2 passes can reuse it; landing consumes it with `activeRoute="search"`
  - rewrote `landing-page.test.tsx` for the new strings, panels, empty hints, and link set
    (Support link removed)
- Verification:
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web test`: 16 files, 70 tests passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm exec prettier --check` on touched files: passed
  - `go build ./...`, `go test ./...`, `go vet ./...`: passed

## 2026-05-16

- Finalized Go + ClickHouse runtime cleanup:
  - archived completed Go rewrite documents under `docs/archive/go_rewrite/`, including phase
    tasks and the API contract inventory
  - replaced the active implementation plan with a no-active-plan marker
  - removed the Python/FastAPI application from `apps/api`
  - removed PostgreSQL and Python runtime services/volumes from Docker Compose
  - retained `migrate-go` as a `tools` profile legacy PostgreSQL import command that requires an
    explicit `POSTGRES_SOURCE_DSN`
  - removed Python-specific Docker/Git ignore rules that only existed for the deleted app
  - replaced Python GitHub Actions validation with Go CI on every push and pull request
  - changed code-style GitHub Actions validation to run on every push and pull request
  - updated README, architecture, project plan, living context, decisions, and recent handoff docs
    for Go + ClickHouse + SQLite as the only current runtime
  - migrated local legacy data into fresh ClickHouse/SQLite targets:
    - `admin_users`: 2
    - `followed_channels`: 7
    - `sender_profiles`: 8570
    - `retention_settings`: 1
    - `worker_heartbeats`: 1
    - `chat_messages`: 123790
    - `raw_kick_events`: 121664
    - `raw_event_attempts`: 121664
  - preserved the legacy PostgreSQL volume intentionally
  - verified `go test ./...`
  - verified `go vet ./...`
  - verified live ClickHouse integration test with
    `KICK_LOGS_RUN_CLICKHOUSE_TESTS=1 go test ./internal/infra/clickhouse -run TestClickHouseMigrationsAndRepositories -count=1 -v`
  - verified `pnpm format:check`
  - verified `git diff --check`
  - verified `docker compose config --services`
  - verified `docker compose --profile tools config --services`
  - verified `docker compose up --build -d --remove-orphans` starts `clickhouse`, Go `api`, Go
    `listener`, and `web`
  - smoke verified `GET /health`, `GET /messages?limit=1`, default admin login, and
    `GET /admin/channels`

- Completed Go rewrite Phase 9 cutover:
  - added Go admin data-management API parity before switching the default runtime
  - implemented `GET /admin/data-management/summary`,
    `PUT /admin/data-management/retention-settings`,
    `POST /admin/data-management/cleanup/preview`, and
    `POST /admin/data-management/cleanup/confirm` in the Go API
  - switched Compose intent so default `api` and `listener` build from `apps/api-go`
  - made `clickhouse` part of the default Compose runtime
  - moved Python/FastAPI/PostgreSQL services behind the `python-reference` profile as
    `postgres`, `api-python`, and `listener-python`
  - moved `migrate-go` behind the `tools` profile
  - updated `.env.example`, README, architecture notes, project plan, implementation plan, and
    active context docs for Go + ClickHouse default runtime
  - PostgreSQL source data and volumes remain available for migration/rollback and are not deleted
  - fixed SQLite sender profile upsert to use atomic `ON CONFLICT` handling for live listener
    races on `kick_user_id` and `slug`
  - closed all checklist items in `docs/tasks/go_rewrite_09_cutover_smoke_docs.md`
  - verified `go test ./...`
  - verified `go vet ./...`
  - verified live ClickHouse integration test with
    `KICK_LOGS_RUN_CLICKHOUSE_TESTS=1 go test ./internal/infra/clickhouse -run TestClickHouseMigrationsAndRepositories -count=1 -v`
  - verified `docker compose up --build -d --remove-orphans` starts default `clickhouse`, Go
    `api`, Go `listener`, and `web`
  - smoke verified `GET /health`, default super-admin login, admin channel list/add/disable,
    listener heartbeat, live and fixture searchable messages, sender exact search, channel/content
    search, reply metadata, emote image metadata, JSON export, CSV export, landing/user/channel
    pages, analytics/profile APIs, admin operations, admin data-management summary, cleanup
    preview, public unauthenticated routes, and unauthenticated admin rejection

- Completed Go rewrite Phase 8 PostgreSQL data migration:
  - added `POSTGRES_SOURCE_DSN` config and PostgreSQL source adapter under
    `internal/infra/postgres`
  - added data migration use case under `internal/usecase/datamigration`
  - added `cmd/migrate -target=data` with `-dry-run`, `-execute`, `-validation-only`,
    `-batch-size`, `-sample-size`, and `-source-postgres-url`
  - added SQLite data migration writer for admin users, followed channels, sender profiles,
    retention settings, worker heartbeat state, and `data_migration_runs`
  - added ClickHouse data migration writer for idempotent chat message, raw event, and raw-event
    attempt inserts
  - added ClickHouse raw event `metadata_json` migration and repository mapping
  - preserved source IDs for migrated SQLite rows, ClickHouse messages, and raw events; raw-event
    attempts use deterministic migrated IDs to avoid duplicates on rerun
  - validated Go-compatible bcrypt admin password hashes before accepting migration
  - added count validation, sample validation, rerun/idempotency coverage, migrated-search fixture
    coverage, and migrated admin hash verification
  - updated README, architecture notes, implementation plan, Compose env, `.env.example`, and
    closed all checklist items in `docs/tasks/go_rewrite_08_data_migration.md`
  - verified `go test ./...`
  - verified `go vet ./...`
  - verified live ClickHouse integration test with
    `KICK_LOGS_RUN_CLICKHOUSE_TESTS=1 go test ./internal/infra/clickhouse -run TestClickHouseMigrationsAndRepositories -count=1 -v`
  - verified `docker compose --profile go-rewrite build migrate-go`
  - verified `docker compose --profile go-rewrite run --rm migrate-go -target=sqlite`
  - verified `docker compose --profile go-rewrite run --rm migrate-go -target=clickhouse`

- Completed Go rewrite Phase 7 analytics/profile parity:
  - implemented public Go routes for `GET /analytics/overview`,
    `GET /analytics/message-volume`, `GET /analytics/top-senders`,
    `GET /analytics/top-channels`, and `GET /analytics/top-emotes`
  - implemented public Go routes for `GET /users/{slug}/analytics` and
    `GET /channels/{slug}/analytics`
  - added ClickHouse analytics repository queries for overview counts, bucketed volume, top
    senders, top channels, top emotes, and latest scoped messages
  - preserved analytics date filters, exact channel scope, sender username/slug scope with
    underscore/hyphen variants, `bucket=hour|day`, and top-list `limit` validation
  - user and channel profile responses now combine SQLite identity metadata with ClickHouse
    analytics and latest message rows
  - unknown profile slugs return the existing 404 detail strings
  - closed all checklist items in `docs/tasks/go_rewrite_07_analytics_profiles.md`
  - verified `go test ./...`
  - verified `go vet ./...`
  - verified live ClickHouse integration test with
    `KICK_LOGS_RUN_CLICKHOUSE_TESTS=1 go test ./internal/infra/clickhouse -run TestClickHouseMigrationsAndRepositories -count=1 -v`
  - verified `docker compose --profile go-rewrite up --build -d api-go`
  - verified live `GET /analytics/overview`, `GET /analytics/message-volume`,
    `GET /analytics/top-senders`, `GET /analytics/top-channels`, `GET /analytics/top-emotes`,
    analytics invalid-range 422, and unknown user-profile 404 smoke checks

- Completed Go rewrite Phase 6 listener ingestion parity:
  - implemented the Go listener runtime wiring in `cmd/listener`
  - added Kick sender profile resolver and Pusher websocket client
  - subscribed to `chatrooms.{chatroom_id}.v2` and channel-level streams for enabled followed
    channels
  - persisted raw Kick chat events into ClickHouse before normalization
  - added raw-event retry processing, processing attempts, max-attempt filtering, and idempotent
    message inserts by `kick_message_id`
  - normalized sender/channel snapshots, reply metadata, emotes, badges, message type, timestamps,
    raw payload JSON, and sender profile cache updates
  - added listener heartbeat writes and operations-summary raw-event health fixes
  - added `listener-go` to the `go-rewrite` Compose profile
  - closed all checklist items in `docs/tasks/go_rewrite_06_listener_ingestion.md`
  - verified `go test ./...`
  - verified `go vet ./...`
  - verified live ClickHouse integration test with
    `KICK_LOGS_RUN_CLICKHOUSE_TESTS=1 go test ./internal/infra/clickhouse -run TestClickHouseMigrationsAndRepositories -count=1 -v`
  - verified `docker compose --profile go-rewrite up --build -d api-go listener-go`
  - verified authenticated `GET /admin/operations/summary` reports fresh listener heartbeat and
    consistent raw-event counts

- Completed Go rewrite Phase 5 message search/export parity:
  - added ClickHouse-backed message search use case and public `GET /messages`
  - added public `GET /messages/export` with JSON and CSV output
  - preserved query parsing for `sender`, `channel`, `q`, `start`, `end`, `cursor`, `limit`,
    `reply_only`, and `emote_only`
  - preserved case-insensitive exact sender matching and case-insensitive contains matching for
    channel/content
  - preserved newest-first ordering and `message_created_at|message_id` cursor pagination
  - expanded ClickHouse message snapshot columns for nested sender/channel IDs, channel banner,
    sender badges, and reply metadata JSON
  - mapped ClickHouse rows back to the current message JSON response shape
  - CSV export uses the contract column order and formula-safe cell prefixing
  - updated Go API startup to wire the message service when ClickHouse is reachable
  - ignored local `apps/api-go/.cache/` so Go test caches do not enter Docker build context
  - closed all checklist items in `docs/tasks/go_rewrite_05_messages_search_export.md`
  - verified `go test ./...`
  - verified `go vet ./...`
  - verified live ClickHouse integration test with
    `KICK_LOGS_RUN_CLICKHOUSE_TESTS=1 go test ./internal/infra/clickhouse -run TestClickHouseMigrationsAndRepositories -count=1 -v`
  - verified Docker Go API smoke for `GET /messages`, JSON export, and CSV export against
    `http://localhost:8001`
  - verified `pnpm format:check`
  - verified `git diff --check`

- Completed Go rewrite Phase 4 auth/admin API parity:
  - added JWT and bcrypt auth infrastructure for the Go API
  - added auth config fields for JWT secret, algorithm, expiry, cookie name, cookie secure flag,
    cookie SameSite value, super-admin seed behavior, and listener stale threshold
  - Go API startup applies SQLite migrations and seeds the default super admin when
    `SEED_SUPER_ADMIN_ON_STARTUP=true`
  - Go API startup applies ClickHouse migrations when ClickHouse is reachable; otherwise admin
    operations can still use SQLite-only data
  - implemented `POST /auth/login`, `POST /auth/logout`, and `GET /auth/me`
  - preserved response shapes and auth cookie behavior expected by the current frontend
  - implemented admin auth checks and super-admin-only admin user creation
  - implemented `GET /admin/users` and `POST /admin/users`
  - added Go Kick web channel resolver for `https://kick.com/api/v2/channels/{slug}`
  - implemented `GET /admin/channels`, `POST /admin/channels`, and disable-style
    `DELETE /admin/channels/{channel_id}`
  - implemented basic `GET /admin/operations/summary` with SQLite channel/sender/listener data
    and ClickHouse message/raw-event/storage/timestamp data when available
  - updated Compose `api-go` env passthrough for JWT and listener freshness settings
  - closed all checklist items in `docs/tasks/go_rewrite_04_auth_admin_api.md`
  - verified `go test ./...`
  - verified `go vet ./...`
  - verified Docker Go API smoke:
    `POST /auth/login`, `GET /auth/me`, `GET /admin/users`, and
    `GET /admin/operations/summary` against `http://localhost:8001`
  - verified `docker compose --profile go-rewrite up --build -d api-go`

- Completed Go rewrite Phase 3 storage/schema:
  - added Go config fields for SQLite path, ClickHouse connection, ClickHouse debug mode, and
    default super-admin credentials
  - added versioned SQLite and ClickHouse migration runners with idempotent migration tracking
  - added SQLite control-plane schema for `admin_users`, `followed_channels`, `sender_profiles`,
    `retention_settings`, `worker_heartbeats`, `schema_migrations`, and `data_migrations`
  - added ClickHouse data-plane schema for `chat_messages`, `raw_kick_events`, and
    `raw_event_attempts`
  - denormalized `chat_messages` with sender/channel snapshots, reply fields, thread parent id,
    emote arrays, normalized sender/channel/content helpers, message type, raw payload JSON, and
    ingestion timestamps
  - added repository interfaces for admin users, followed channels, messages, raw events, and
    storage stats
  - added concrete SQLite repositories for admin users, followed channels, super-admin seeding, and
    control-plane stats
  - added concrete ClickHouse repositories for messages, raw events/attempts, and table-size stats
  - added `clickhouse` and `migrate-go` Compose services behind profile `go-rewrite`
  - corrected ClickHouse healthcheck to use `clickhouse-client` with the configured user/password
  - updated README, architecture, implementation plan, and context docs for the storage foundation
  - closed all checklist items in `docs/tasks/go_rewrite_03_storage_schema.md`
  - verified `go test ./...`
  - verified live ClickHouse integration test with
    `KICK_LOGS_RUN_CLICKHOUSE_TESTS=1 go test ./internal/infra/clickhouse -run TestClickHouseMigrationsAndRepositories -count=1 -v`
  - verified `docker compose --profile go-rewrite run --rm migrate-go`
  - verified `docker compose --profile go-rewrite build api-go`

- Completed Go rewrite Phase 2 workspace/tooling:
  - added `apps/api-go` Go module and command entrypoints for `api`, `listener`, and `migrate`
  - added environment config loading with local defaults and structured `log/slog` JSON logging
  - added stdlib HTTP server bootstrap with CORS, request logging, panic recovery, and
    contract-compatible `GET /health`
  - added internal package skeletons for domain, ports, use cases, infrastructure, HTTP routes, and
    schemas
  - added Go tests for config loading, CORS preflight, and health response shape
  - added Go Dockerfile and optional Compose `api-go` service behind the `go-rewrite` profile
  - documented Go rewrite development commands in README and added current architecture notes
  - ignored local Go build outputs and build cache under `apps/api-go`
  - closed all checklist items in `docs/tasks/go_rewrite_02_workspace_tooling.md`
  - verified `go test ./...`
  - verified `go vet ./...`
  - verified local binary health smoke: `GET /health` returned `{"status":"ok"}`
  - verified `docker compose --profile go-rewrite build api-go`
  - verified `pnpm format:check`
  - verified `git diff --check`

## 2026-05-15

- Completed Go rewrite Phase 1 contract inventory:
  - added `docs/contracts/api_contract.md` as the current Python backend contract snapshot for the
    Go rewrite
  - added representative successful and error JSON fixtures under `docs/contracts/fixtures/`
  - documented endpoint paths, methods, access boundaries, request body fields, query parameters,
    response shapes, auth cookie behavior, status-code expectations, cursor parsing, CSV export
    column order, sender exact matching, channel/content matching, reply metadata, and emote fields
  - verified the endpoint list against backend route/schema files and frontend API wrappers/types
  - closed all checklist items in `docs/tasks/go_rewrite_01_contract_inventory.md`
  - verified `python -m uv run pytest`: 72 passed, 52 skipped
  - verified `pnpm format:check`
  - verified `git diff --check`
- Started the Go + ClickHouse rewrite planning track:
  - reorganized historical docs into `docs/archive/mvp/` and `docs/archive/post_mvp/`
  - archived completed post-MVP task files so `docs/tasks/` can hold only active rewrite tasks
  - replaced `docs/implementation_plan.md` with the Go + ClickHouse rewrite implementation plan
  - documented the storage decision: ClickHouse for messages/raw events/analytics and SQLite for
    auth/admin/control-plane state
  - added active phase task files from contract inventory through cutover and smoke testing
- Fixed Docker Compose backend env passthrough for release readiness:
  - API service now receives `DATABASE_ECHO`, `JWT_ALGORITHM`, `JWT_EXPIRES_MINUTES`,
    `JWT_COOKIE_SECURE`, `JWT_COOKIE_SAMESITE`, and `SEED_SUPER_ADMIN_ON_STARTUP` from `.env`
  - listener service now receives `DATABASE_ECHO`
  - verified `docker compose config` renders the expected environment variables
- Completed Post-MVP Feature 8 final smoke and documentation:
  - hardened three backend assertions that were too brittle against a live local database with
    existing raw events/messages
  - verified backend checks: `python -m uv run pytest` reported 124 passed,
    `python -m uv run ruff check .` passed, and `python -m uv run ruff format --check .` passed
  - verified frontend checks: `pnpm --filter @kick-logs/web test` reported 16 files and 66 tests
    passed, plus typecheck, lint, build, and `pnpm format:check`
  - verified `docker compose up --build -d` starts `postgres`, `api`, `listener`, and `web`
  - smoke checked public landing/search/login/admin shell pages, public messages/search/export,
    analytics, user profile, channel profile, authenticated operations, authenticated data
    management summary, and data cleanup dry-run
  - verified unauthenticated admin APIs return 401 while public routes remain accessible
  - updated README project status and archived MVP docs so historical plans are clearly marked
  - `docs/tasks/post_mvp_08_final_smoke.md` has all checkboxes closed
- Completed Post-MVP Feature 7 data management:
  - README now documents admin data-management usage, retention behavior, guarded cleanup, and
    Docker Compose PostgreSQL backup/restore commands
  - `docs/tasks/post_mvp_07_data_management.md` has all checkboxes closed
  - destructive cleanup remains admin-only and requires dry-run preview plus exact confirmation
    text before deletion
  - verified `python -m uv run pytest`: 124 passed
  - verified `python -m uv run ruff check .`
  - verified `python -m uv run ruff format --check .`
  - verified `pnpm --filter @kick-logs/web test`: 16 files, 66 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
  - verified `pnpm --filter @kick-logs/web build`
  - verified `pnpm format:check`
- Implemented the frontend for Post-MVP Feature 7 data management:
  - added typed data-management API wrappers
  - added `/admin` `DataManagementPanel` below operations status
  - panel shows database/table sizes and current retention settings
  - retention controls support keep forever, 30 days, and 90 days for messages/raw events
  - cleanup flow requires dry-run preview before confirmation
  - delete action is disabled until exact backend confirmation text is typed
  - success state reports deleted message/raw-event counts
  - added frontend tests for settings display, dry-run preview, blocked deletion without
    confirmation, confirmed deletion, and API errors
  - verified
    `pnpm --filter @kick-logs/web test -- data-management-panel.test.tsx admin-dashboard.test.tsx`:
    2 files, 8 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
- Implemented the backend foundation for Post-MVP Feature 7 data management:
  - added `data_retention_settings` with singleton retention settings
  - default message/raw-event retention is `null`, meaning keep forever
  - added admin-only `GET /admin/data-management/summary`
  - added admin-only `PUT /admin/data-management/retention-settings`
  - added admin-only `POST /admin/data-management/cleanup/preview`
  - added admin-only `POST /admin/data-management/cleanup/confirm`
  - cleanup targets cover old messages, old raw events, a specific channel, or a specific sender
  - confirmed cleanup requires the exact confirmation text returned by preview
  - added backend tests for permissions, retention defaults/updates, dry-run counts, rejected
    confirmation, and confirmed deletion
  - verified
    `python -m uv run pytest tests/data_management/test_http_admin_data_management.py tests/database/test_alembic_migration.py tests/database/test_models_metadata.py`:
    13 passed
  - verified `python -m uv run ruff check .`

## 2026-05-14

- Completed Post-MVP Feature 6 channel/publisher profiles:
  - README documents `/channels/[slug]` and `GET /channels/{slug}/analytics`
  - `docs/tasks/post_mvp_06_channel_profiles.md` has all checkboxes closed
  - verified `python -m uv run pytest`: 119 passed
  - verified `python -m uv run ruff check .`
  - verified `python -m uv run ruff format --check .`
  - verified `pnpm --filter @kick-logs/web test`: 15 files, 61 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
  - verified `pnpm --filter @kick-logs/web build`
  - verified `pnpm format:check`
- Implemented the frontend for Post-MVP Feature 6 channel profiles:
  - added public `/channels/[slug]`
  - added typed channel profile API wrapper and response types
  - channel profile UI renders summary metadata, activity metrics, day-bucket message volume, top
    senders, top emotes, latest messages, loading, empty, error, and not-found states
  - channel profile pages link to `/search?channel={slug}`
  - `/search` channel labels now link to public channel profiles
  - `/admin` channel rows now link to public channel profiles when slug data is present
  - verified `pnpm --filter @kick-logs/web test`: 15 files, 61 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
  - verified `pnpm --filter @kick-logs/web build`
- Implemented the backend API for Post-MVP Feature 6 channel profiles:
  - added public `GET /channels/{slug}/analytics`
  - endpoint returns stored Kick channel metadata, overview totals, day-bucket message volume,
    top senders, top emotes, and latest messages
  - unknown channel slugs return 404
  - latest profile messages are queried by exact channel id
  - added backend coverage for existing channel profiles, unknown channels, volume, top senders,
    top emotes, and latest messages
  - verified
    `python -m uv run pytest tests/profiles/test_http_channel_profiles.py tests/analytics/test_http_analytics.py tests/messages/test_http_search_messages.py`:
    18 passed
  - verified `python -m uv run ruff check .`
  - verified `python -m uv run ruff format --check .`
- Fixed Kick profile slug handling for usernames with underscores:
  - frontend sender profile links now convert `_` to `-`, so `example_user` routes to
    `/users/example-user`
  - reply preview sender profile slugs use the same canonical Kick URL behavior
  - backend ingestion normalizes new sender slugs to Kick profile URL form
  - backend sender/profile/search/analytics lookups accept both underscore and hyphen forms so
    existing stored data remains reachable
  - added backend and frontend coverage for underscore-to-hyphen profile slug behavior
  - verified targeted backend tests:
    `python -m uv run pytest tests/domain/test_value_objects.py tests/messages/test_ingest_message.py tests/messages/test_http_search_messages.py tests/profiles/test_http_user_profiles.py`
    returned 28 passed
  - verified `python -m uv run ruff check .`
  - verified `python -m uv run ruff format --check .`
  - verified `pnpm --filter @kick-logs/web test`: 14 files, 56 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
  - verified `pnpm --filter @kick-logs/web build`
  - verified `pnpm format:check`
- Polished public profile navigation and profile panel styling:
  - `/search` reply previews now link the muted replied-to sender name to `/users/[slug]`
  - reply metadata extraction reads `original_sender.slug` when present and falls back to a
    lowercase username-derived slug
  - `/users/[slug]` top identity section now uses the same rounded bordered padded panel treatment
    as the rest of the profile UI
  - added frontend coverage for reply sender profile links and reply slug fallback
  - verified `pnpm --filter @kick-logs/web test`: 13 files, 54 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
  - verified `pnpm --filter @kick-logs/web build`
  - verified `pnpm format:check`
- Implemented Post-MVP Feature 4 landing page with analytics:
  - replaced root `/` search redirect with `LandingPage`
  - landing explains the self-hosted Kick Logs project with compact product-focused copy
  - landing fetches public analytics overview, recent day-bucket message volume, top channels,
    top emotes, and top senders
  - landing includes loading, API-error, and fresh-install empty states
  - landing links to `/search`, `/admin`, GitHub, and Buy Me a Coffee support
  - added frontend tests for analytics rendering, empty state, and navigation links
  - updated README, design guide, project plan, architecture notes, implementation plan, and
    context docs so `/` is documented as the landing page
  - closed all checkboxes in `docs/tasks/post_mvp_04_landing_analytics.md`
  - verified `pnpm --filter @kick-logs/web test`: 12 files, 50 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
  - verified `pnpm --filter @kick-logs/web build`
  - verified `pnpm format:check`
  - verified `docker compose up --build -d web`
  - verified `GET http://localhost:3000/`: HTTP 200
  - verified `GET http://localhost:3000/search`: HTTP 200
- Linked the `/search` and `/admin` header brand/logo areas back to `/`:
  - `/search` header now wraps the Kick Logs logo/title block in a `/` link
  - `/admin` header brand link now points to `/` instead of `/admin`
  - added frontend assertions for both brand links
  - verified `pnpm --filter @kick-logs/web test -- search-screen.test.tsx admin-dashboard.test.tsx`:
    2 files, 11 tests passed
  - verified `pnpm --filter @kick-logs/web lint`
  - verified `pnpm format:check`
- Implemented Post-MVP Feature 5 user profile analytics:
  - added public `GET /users/{slug}/analytics`
  - response includes sender identity/profile image, overview totals, day-bucket message volume,
    top channels, top emotes, and latest messages
  - unknown sender slugs return 404
  - added backend tests for existing profile analytics, unknown sender, volume, top channels, top
    emotes, and latest messages
  - added public `/users/[slug]` frontend route and profile UI
  - search result sender names and avatars link to `/users/[slug]`
  - profile UI links to `/search?sender={slug}`
  - added frontend tests for profile rendering, not-found behavior, and search-row sender links
  - updated README, project plan, architecture, design guide, task checklist, and context docs
  - verified `python -m uv run pytest`: 113 passed
  - verified `python -m uv run ruff check .`
  - verified `python -m uv run ruff format --check .`
  - verified `pnpm --filter @kick-logs/web test`: 13 files, 53 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
  - verified `pnpm --filter @kick-logs/web build`
  - verified `pnpm format:check`

## 2026-05-13

- Implemented Post-MVP Feature 3 analytics foundation:
  - added `AnalyticsFilters` with date range and exact channel/sender scope
  - added analytics DTOs, use cases, repository port, and SQLAlchemy aggregate repository
  - added public read-only `GET /analytics/overview`
  - added public read-only `GET /analytics/message-volume`
  - added public read-only `GET /analytics/top-senders`
  - added public read-only `GET /analytics/top-channels`
  - added public read-only `GET /analytics/top-emotes`
  - message volume supports `bucket=hour|day`
  - top-list endpoints support `limit` from 1 to 100
  - top emotes aggregate parsed `chat_messages.emotes` JSONB values
  - added backend tests for aggregate correctness, empty datasets, date range filtering, channel
    scope, sender scope, and limit handling
  - added typed frontend analytics API wrappers and parameter mapping tests
  - documented the analytics API shape in README, architecture, project plan, and context docs
  - verified `python -m uv run pytest`: 111 passed
  - verified `python -m uv run ruff check .`
  - verified `python -m uv run ruff format --check .`
  - verified `pnpm --filter @kick-logs/web test -- analytics/api.test.ts`: 1 file, 3 tests passed
  - verified `pnpm --filter @kick-logs/web test`: 11 files, 47 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
  - verified `pnpm --filter @kick-logs/web build`
  - verified `pnpm format:check`
- Polished the public `/search` filter form density:
  - moved quick date ranges from four visible buttons into one compact `Hızlı aralık` select
  - moved JSON/CSV export actions behind one square `Dışa aktar` icon button
  - added outside-click close behavior for the export menu
  - relabeled result-type filters to `Sadece yanıtlar` and `Sadece emote` so their scope is clearer
  - moved result-type filters below the date controls, to the left of the `İşlem` action group
  - updated design and context docs for the compact control behavior
  - verified `pnpm --filter @kick-logs/web test -- search-screen.test.tsx`: 1 file, 8 tests passed
  - verified `pnpm --filter @kick-logs/web test`: 10 files, 44 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
  - verified `pnpm --filter @kick-logs/web build`
  - verified `pnpm format:check`
- Implemented the frontend for Post-MVP Feature 2 search improvements:
  - added date preset buttons for last 1 hour, 24 hours, 7 days, and 30 days
  - added `Yanıtlar` and `Emote içerenler` controls mapped to `reply_only` and `emote_only`
  - kept the new filters shareable in `/search` URL query state
  - rendered URLs inside message content as safe new-tab anchors
  - highlighted matched `q` text in message content without moving inline emotes
  - added CSV and JSON export buttons that use the last submitted filters
  - updated `docs/design/design.md` and the Feature 2 task file
  - verified `pnpm --filter @kick-logs/web test`: 10 files, 42 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
  - verified `pnpm --filter @kick-logs/web build`
  - verified `pnpm format:check`
  - re-verified backend `python -m uv run ruff check .`
  - re-verified backend `python -m uv run pytest tests/domain/test_value_objects.py tests/test_config.py tests/messages/test_http_search_messages.py`: 18 passed
  - closed all acceptance checkboxes in `docs/tasks/post_mvp_02_search_improvements.md`
- Implemented the backend foundation for Post-MVP Feature 2 search improvements:
  - `MessageSearchFilters` now carries `reply_only` and `emote_only`
  - public `GET /messages` applies both filters with existing optional `AND` semantics
  - added public `GET /messages/export` for filtered JSON and CSV exports
  - export reuses the same search use case/filter contract and caps output with
    `MESSAGE_EXPORT_MAX_ROWS`
  - Compose and `.env.example` expose the export row cap
  - README, architecture, project plan, task file, and context docs describe the new API
  - verified `python -m uv run ruff check .`
  - verified `python -m uv run pytest tests/domain/test_value_objects.py tests/test_config.py tests/messages/test_http_search_messages.py`: 18 passed
- Added a Feature 2 planning task for clickable message links:
  - `/search` result rows should render URLs inside message content as safe clickable links
  - link rendering must preserve inline emote placement and matched-text highlighting
  - link rendering tests were added to the Feature 2 task checklist
- Completed Post-MVP Feature 1 admin operations acceptance:
  - README now documents `/admin` operations dashboard usage and
    `GET /admin/operations/summary`
  - `docs/tasks/post_mvp_01_admin_operations.md` has all checkboxes closed
  - verified backend, frontend, and formatting checks for the touched areas
- Implemented the frontend dashboard for Post-MVP Feature 1 admin operations:
  - added typed frontend operations API wrapper for `GET /admin/operations/summary`
  - mounted `OperationsDashboard` at the top of `/admin`
  - added compact cards for listener status, database size, message count, raw event count,
    failed raw events, pending raw events, and last ingest time
  - added manual refresh and warning/error states for stale listener heartbeat, failed raw
    events, and API failures
  - kept operations metrics visually separate from channel and user management
  - verified `pnpm --filter @kick-logs/web test`: 10 files, 36 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`
  - verified `pnpm --filter @kick-logs/web lint`
- Implemented the backend foundation for Post-MVP Feature 1 admin operations:
  - added `worker_heartbeats` domain entity, SQLAlchemy model, repository, and Alembic
    migration `20260513_0003`
  - listener now records a periodic `listener` heartbeat controlled by
    `LISTENER_HEARTBEAT_INTERVAL_SECONDS`
  - added operations repository/use case and admin-only
    `GET /admin/operations/summary`
  - summary response includes core counts, raw event status counts, database/table sizes,
    key ingest timestamps, and listener freshness based on
    `LISTENER_HEARTBEAT_STALE_AFTER_SECONDS`
  - updated Compose and `.env.example` with heartbeat settings
  - verified `python -m uv run alembic upgrade head`
  - verified `python -m uv run ruff check .`
  - verified `python -m uv run pytest`: 101 passed
- Updated validation workflow branch triggers:
  - `Code Style` now runs for pull requests targeting `main` or `dev`
  - `Code Style` now runs on pushes to `main` or `dev`
  - `Python CI` now runs for pull requests targeting `main` or `dev`
  - `Python CI` now runs on pushes to `main` or `dev`
- Changed public message search sender filtering:
  - `sender` now uses case-insensitive exact matching against sender username/slug snapshots
  - partial sender queries such as `yavuz` no longer match `notyavuz` or `yavuz123`
  - `channel` and message content filters keep case-insensitive contains behavior
  - added backend coverage for exact sender matches and rejected partial sender matches

## 2026-05-12

- Archived the completed MVP implementation plan:
  - moved the old active plan to `docs/archive/mvp_implementation_plan.md`
  - moved old phase task files to `docs/archive/tasks/`
  - replaced `docs/implementation_plan.md` with the active post-MVP feature roadmap
  - added post-MVP task files for admin operations, search improvements, analytics foundation, landing analytics, user profiles, channel profiles, data management, and final smoke/docs
  - updated agent/project/context docs so archived MVP task files are historical context only
- Added Buy Me a Coffee sponsorship metadata:
  - created `.github/FUNDING.yml` with `buy_me_a_coffee: yavuzselim` so GitHub can show the Sponsor button
  - added a README support badge linked to `https://buymeacoffee.com/yavuzselim`
  - added a short README `Support` section for contributors/users who want to support continued development
- Fixed `/search` date filter submission:
  - URL state keeps local `datetime-local` values for stable input rendering and sharing
  - API query params convert `start` and `end` to UTC ISO strings
  - `end` now includes the full selected minute so minute-precision inputs include messages through `:59.999`
  - ISO date values in shared URLs normalize back to local input values
  - the site favicon now uses the existing Kick Logs app logo
- Added repository formatting standards:
  - added root `.prettierrc.json` using the current frontend style: 2 spaces, semicolons, double quotes, no trailing commas, 100-column print width, LF line endings
  - added `.prettierignore` for generated files, lockfiles, `.pen` artifacts, and local agent skills
  - added root `pnpm format` and `pnpm format:check` scripts
  - added `prettier` as a root dev dependency
  - configured Ruff Format for Python with spaces, double quotes, LF line endings, and the existing 100-column line length
  - added `.github/workflows/code-style.yml` to run `pnpm format:check`
  - updated backend Python CI to run `ruff format --check .`
  - normalized existing frontend, docs, and Python files with the configured formatters
- Added backend GitHub Actions workflow:
  - `.github/workflows/python-tests.yml` runs on pull requests and pushes to `main`
  - starts PostgreSQL 16 as a workflow service
  - installs backend dependencies with `uv`
  - applies Alembic migrations before tests
  - runs `python -m uv run ruff check .`
  - runs `python -m uv run pytest`
  - added README Python CI badge and continuous integration section
- Rewrote root `README.md` as a professional public repository guide:
  - added the app logo at the top of the document
  - added repository, issues, and pull request links
  - documented quick start with Docker Compose
  - documented default local admin usage and required secret overrides
  - documented services, API surface, local development commands, data captured, configuration, contribution workflow, and operational notes
  - added fork/PR guidance for community contributors
- Added root MIT `LICENSE` file with copyright holder `YSelim0` and updated the README license section.
- Implemented GitHub issue #3 reply rendering:
  - added backend coverage for the observed Kick reply payload shape:
    - `type="reply"`
    - `thread_parent_id`
    - `metadata.original_sender.username`
    - `metadata.original_message.id`
    - `metadata.original_message.content`
  - added public `/messages` coverage to verify reply metadata and thread parent ids are returned unchanged
  - added frontend reply metadata extraction guard for `message_type === "reply"`
  - rendered replied-to sender/content above the current message in `/search` result rows
  - added a `title` attribute to reply previews for long original content
  - added frontend tests for reply metadata extraction and reply/non-reply row rendering
  - verified `pnpm --filter @kick-logs/web test`: 9 files, 28 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`: passed
  - verified `pnpm --filter @kick-logs/web lint`: passed
  - verified `pnpm --filter @kick-logs/web build`: passed
  - verified `python -m uv run ruff check .`: passed
  - verified `python -m uv run pytest`: 96 passed
- Updated public `/search` initial-load behavior:
  - bare `/search` no longer fetches latest messages automatically
  - result area shows an icon prompt: `Arama yapmak için yukarıdaki formu kullanın.`
  - URL query params still trigger search on load
  - explicitly submitting empty filters still fetches latest messages
  - added `SearchScreen` tests for no initial fetch, URL query fetch, and explicit empty search
  - verified `pnpm --filter @kick-logs/web test`: 7 files, 23 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`: passed
  - verified `pnpm --filter @kick-logs/web lint`: passed

## 2026-05-11

- Started GitHub issue #1 durable Kick ingestion work on branch `feature/issue-1-durable-inbox`:
  - added `RawKickEvent` and `RawEventStatus`
  - added `raw_kick_events` SQLAlchemy model and Alembic revision `20260511_0002`
  - added raw event repository port and SQLAlchemy implementation
  - added raw event storage and processing use cases
  - refactored the listener websocket read path to persist raw chat events before normalization/message insert work
  - added raw event worker loop with batch processing, retry state, stale processing reclaim, and pending-count logging
  - added periodic listener reconnect for enabled-channel resync
  - exposed listener worker/batch/retry/resync settings through config, Compose, and `.env.example`
  - added listener, domain, metadata, migration, and repository tests for durable inbox behavior
  - verified `python -m uv run ruff check .`: passed
  - verified `python -m uv run alembic upgrade head`: applied `20260511_0002`
  - verified `python -m uv run alembic current`: `20260511_0002 (head)`
  - verified `python -m uv run pytest`: 94 passed
  - verified `python -m uv run pytest tests/listener tests/domain tests/database/test_models_metadata.py tests/database/test_alembic_migration.py`: 43 passed
  - verified `python -m uv run pytest tests/database/test_repositories.py tests/messages/test_ingest_message.py tests/listener/test_listener_service.py`: 19 passed against local PostgreSQL
  - verified `docker compose up --build -d postgres api listener`: passed
  - verified `GET http://localhost:8000/health`: `{"status":"ok"}`
  - verified listener logs show raw event storage and raw event worker processing with `pending=0`
- Fixed `/search` hydration mismatch caused by server/client timezone differences in default date range rendering:
  - changed the search screen's first render to use static empty state
  - kept the required default 7-day date range by applying it after client hydration
  - restarted the `web` service and verified server HTML no longer includes dynamic default datetime values
  - verified `pnpm --filter @kick-logs/web test`: 6 files, 20 tests passed
  - verified `pnpm --filter @kick-logs/web typecheck`: passed
  - verified `pnpm --filter @kick-logs/web lint`: passed
  - verified `pnpm --filter @kick-logs/web build`: passed
- Fixed browser CORS for the frontend login flow:
  - added FastAPI `CORSMiddleware`
  - wired allowed origins from comma-separated `BACKEND_CORS_ORIGINS`
  - covered `/auth/login` preflight with a backend test
  - hardened the message repository pagination test with a unique query term so existing local chat history cannot affect it
  - verified `python -m uv run pytest`: 85 passed
  - verified `python -m uv run ruff check .`: passed
  - verified live Docker `OPTIONS /auth/login` from `http://localhost:3000` returns CORS headers
  - verified live Docker `POST /auth/login` returns 200 and sets the auth cookie

## 2026-05-09

- Created project context structure request.
- Created commit convention skill and committed it:
  - `679d936 feat(repo): add commit convention skill`
- Planned Docker Compose dev stack:
  - `postgres`
  - `api`
  - `listener`
  - `web`
- Expanded MVP plan with auth, search semantics, date filters, full payload storage, sender profile enrichment, emote rendering fallback, and one-worker listener model.
- Added `docs/context/recent_changes.md` as the short latest-change handoff file and linked it from `AGENTS.md`.
- Added architecture plan covering clean architecture backend structure, SQLAlchemy/Alembic ORM choice, listener entrypoint, frontend structure, and Docker runtime shape.
- Added UI design guide under `docs/design/design.md` and documented the backend-first development rule.
- Documented that multi-agent development is allowed for non-overlapping work scopes.
- Added search screen design to `docs/design/design.pen` and updated UI palette/rules.
- Refined search design guidance so the provided reference image is used for form structure only, while the app keeps its dark `#26001B` / `#FFF600` palette and avoids blur, glow, and oversized typography.
- Refined `/search` result design to use one outer list container with stacked message rows, circular avatars, inline emotes, and adjusted spacing below the search button.
- Clarified route access: `/search` is public, while `/admin` is the authenticated backend management dashboard for operational tasks like followed-channel management.
- Added `docs/implementation_plan.md` and phase-scoped task files from `docs/tasks/phase1_tasks.md` through `docs/tasks/phase10_tasks.md`.
- Updated agent instructions so implementation agents read the plan and only the matching phase task file before working.

## 2026-05-10

- Locked Phase 1 Docker Compose scope to `postgres` and `api` only; `web` and `listener` services must be added later in their owning phases, with no placeholder services.
- Started Phase 1 by adding root local development defaults:
  - `.gitignore`
  - `.env.example`
  - `README.md`
- Added the initial `apps/api` FastAPI project skeleton with:
  - `uv` project metadata in `apps/api/pyproject.toml`
  - clean architecture package folders
  - settings and logging core modules
  - FastAPI app factory and `GET /health`
  - minimal tests for settings, app factory, and health route
- Added Phase 1 Docker runtime files:
  - root `compose.yaml` with `postgres` and `api` services only
  - `apps/api/Dockerfile`
  - `apps/api/.dockerignore`
  - API hot reload volume setup through Docker Compose
- Updated backend lint instructions in `README.md`.
- Verified backend package import through `uv`.
- Verified `uv run pytest` from `apps/api`: 3 tests passed.
- Verified `uv run ruff check .` from `apps/api`: all checks passed.
- Verified `docker compose config --services`: only `postgres` and `api` are present.
- Attempted `docker compose up --build -d postgres api`, but live Docker startup is pending because the local Docker daemon was not running.
- Retried Docker Compose after daemon access was available. The API image build exposed two container-build issues:
  - `apps/api/pyproject.toml` referenced root `README.md`, which is outside the Docker build context.
  - `uv sync` attempted editable project installation before `src/` was copied into the image.
- Updated API packaging/build flow so dependency sync happens before source copy with `--no-install-project`, then installs the project after `src/` is present.
- Re-ran `docker compose up --build -d postgres api` successfully.
- Verified `GET http://localhost:8000/health` returned `{"status":"ok"}`.
- Phase 1 acceptance is complete.
- Started Phase 2 by adding framework-independent domain entities/value objects and application repository/unit-of-work ports.
- Added Phase 2 SQLAlchemy/Alembic foundation:
  - SQLAlchemy async engine/session factory
  - ORM models for `users`, `channels`, `senders`, and `chat_messages`
  - Alembic async environment
  - initial migration with `pg_trgm`, JSONB columns, dedupe constraints, and search indexes
- Verified the initial migration applies cleanly to local Docker PostgreSQL with `alembic upgrade head`.
- Added SQLAlchemy repository implementations and async unit of work wiring for users, channels, senders, and chat messages.
- Added repository tests for create/read/update flows and message search/pagination repository behavior using isolated transactions.
- Verified full backend test suite: 27 tests passed.
- Verified `ruff check .`: all checks passed.
- Verified `alembic current`: `20260510_0001 (head)`.
- Phase 2 acceptance is complete.
- Started Phase 3 by adding auth security ports and infrastructure services:
  - Passlib password hasher
  - PyJWT token service
  - JWT cookie/session settings
- Added Phase 3 application use cases and seed support:
  - login
  - get current user
  - list admin users
  - create admin user
  - idempotent default super admin seed
- Added Phase 3 HTTP auth/admin user surface:
  - `POST /auth/login`
  - `POST /auth/logout`
  - `GET /auth/me`
  - `GET /admin/users`
  - `POST /admin/users`
  - cookie-based current-user and role dependencies
  - startup super admin seed wiring
- Verified full backend test suite: 45 tests passed.
- Verified `ruff check .`: all checks passed.
- Verified Docker rebuild/start with auth dependencies.
- Verified real API login/me smoke with default super admin credentials.
- Updated Docker API startup to run `alembic upgrade head` before Uvicorn so the startup super admin seed runs after migrations.
- Phase 3 acceptance is complete.
- Started Phase 4 by adding Kick channel resolver, admin channel use cases, and admin channel route scaffolding.
- Added Phase 4 channel management implementation:
  - Kick web channel resolver using `curl_cffi`
  - channel DTOs and resolver port
  - list/add/remove channel use cases
  - `GET /admin/channels`
  - `POST /admin/channels`
  - `DELETE /admin/channels/{id}`
  - tests for resolver parsing/failure and authenticated admin channel management
- Verified Phase 4 channel management scope:
  - `uv run pytest tests/channels`: 7 tests passed
  - `uv run ruff check .`: all checks passed
- Added Phase 4 message ingestion foundation:
  - emote parser for `[emote:id:name]` tokens
  - chat message DTO mapping
  - channel lookup by Kick chatroom id
  - idempotent `IngestMessageUseCase`
  - sender upsert from Kick sender payload
  - raw payload, badges, reply metadata, thread parent id, timestamp, and parsed emote persistence
- Verified Phase 4 ingestion scope:
  - `uv run pytest tests/messages`: 8 tests passed
  - `uv run ruff check .`: all checks passed
- Added Phase 4 public message search API:
  - `SearchMessagesUseCase`
  - public `GET /messages`
  - response schemas with sender/channel metadata and parsed emotes
  - cursor encoding as `{message_created_at.isoformat()}|{message_id}`
  - batch sender/channel lookup for search response enrichment
  - public HTTP tests for empty filters, optional filter combinations, date range, cursor pagination, and invalid cursor handling
- Verified Phase 4 search scope:
  - `uv run pytest tests/messages`: 13 tests passed
  - `uv run ruff check .`: all checks passed
- Completed Phase 4 acceptance:
  - admins can manage followed channels through API
  - public message search works without login
  - ingestion use case persists normalized messages without listener runtime
  - documented search filter combinations and cursor pagination are covered by tests
- Final Phase 4 verification:
  - `uv run pytest`: 65 tests passed
  - `uv run ruff check .`: all checks passed
  - `docker compose up --build -d postgres api`: backend stack starts
  - `GET http://localhost:8000/health`: returned `{"status":"ok"}`
  - `GET http://localhost:8000/messages?limit=1`: returned a public search response
- Started Phase 5 listener worker implementation.
- Added listener foundation:
  - listener channel DTOs
  - `LoadEnabledChannelsUseCase`
  - Kick Pusher chat event parser
  - reconnect backoff policy
  - unit tests for enabled-channel loading, event parsing, and reconnect delays
- Verified Phase 5 listener foundation:
  - `uv run pytest tests/listener`: 10 tests passed
  - `uv run ruff check .`: all checks passed
- Added Phase 5 listener runtime:
  - direct `websockets` runtime dependency
  - `KickPusherClient`
  - sender profile resolver port and Kick web implementation
  - listener settings for Pusher URL and reconnect backoff
  - `ListenerService`
  - worker entrypoint at `kick_logs.presentation.worker.main`
  - tests for fake Pusher ingestion, malformed event handling, Pusher subscriptions, sender profile resolver, and enrichment fallback
- Verified Phase 5 listener runtime:
  - `uv run pytest tests/listener`: 17 tests passed
  - `uv run ruff check .`: all checks passed
- Added listener Docker Compose service:
  - same backend source/image pattern as API
  - separate `listener_venv` volume
  - depends on healthy PostgreSQL
  - starts with `uv run alembic upgrade head && uv run python -m kick_logs.presentation.worker.main`
- Added listener environment defaults to `.env.example`.
- Completed Phase 5 acceptance:
  - listener ingests mocked Kick chat events through the existing ingestion use case
  - listener Docker service starts without breaking API
  - malformed events and transient websocket failures do not crash permanently
  - no frontend work was introduced
- Final Phase 5 verification:
  - `uv run pytest`: 83 tests passed
  - `uv run ruff check .`: all checks passed
  - `docker compose config --services`: returned `postgres`, `api`, `listener`
  - `docker compose up --build -d postgres api listener`: backend stack starts
  - `GET http://localhost:8000/health`: returned `{"status":"ok"}`
  - listener logs show idle no-channel checks without crashing
- Aligned listener runtime with the verified Kick web chat flow:
  - Pusher subscription payload now includes empty `auth`
  - websocket connection uses 30 second ping interval and 10 second ping timeout
  - Kick web HTTP resolvers use `chrome124` impersonation
- Completed Phase 6 backend verification and acceptance:
  - `python -m uv run pytest`: 83 tests passed
  - `python -m uv run ruff check .`: passed
  - `python -m uv run alembic current`: `20260510_0001 (head)`
  - `docker compose up --build -d postgres api listener`: passed
  - `GET /health`: passed
  - default super admin login and `GET /auth/me`: passed
  - unauthenticated `GET /admin/channels`: returned 401
  - admin channel add/disable smoke with slug `hype`: passed
  - public `GET /messages?limit=1`: passed without login
  - listener Docker logs show useful idle status without crashing
- Cleaned Phase 6 runtime warnings:
  - increased default local/Compose `JWT_SECRET_KEY` length for HS256
  - pinned `bcrypt` to `>=4.0.1,<4.1` for Passlib compatibility
- Updated `README.md` with backend verification steps, access model, env/local secret expectations, and Kick integration fragility notes.
- Marked Phase 6 task file acceptance as complete.
- Started and completed Phase 7 frontend foundation:
  - added pnpm workspace files
  - scaffolded `apps/web` with Next.js App Router and TypeScript
  - configured Tailwind and shadcn/ui base files
  - added lucide-react dependency
  - added dark-only palette tokens from the UI design guide
  - added placeholder routes for `/`, `/search`, `/login`, and `/admin`
  - added typed frontend API client and feature endpoint wrappers for health, auth, messages, channels, and users
  - added `web` Docker Compose service and web Dockerfile
  - added frontend env defaults to `.env.example`
- Verified Phase 7 frontend foundation:
  - `pnpm install`: completed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `docker compose up --build -d web`: passed
  - `GET http://localhost:3000`: returned HTTP 200
  - `GET http://localhost:8000/health`: returned `{"status":"ok"}`
- Documented frontend install/scripts, full dev stack startup, and Phase 7 verification in `README.md`.
- Marked Phase 7 task file acceptance as complete.
- Started and completed Phase 8 public search UI:
  - read `docs/design/design.pen` JSON directly because Pencil MCP app connection was unavailable
  - used `Search Screen / Desktop (User Friendly ReTouch Current)` as the implementation reference
  - replaced the `/search` placeholder with the public search screen
  - added search form fields mapped to `sender`, `channel`, `q`, `start`, and `end`
  - preserves submitted filters in the URL
  - omits empty filter values from backend query params
  - fetches public `GET /messages` without auth
  - implements cursor-based infinite scroll
  - renders dense message rows inside one shared list container
  - renders circular sender avatars and fallback initials
  - renders `[emote:id:name]` tokens inline with image fallback text
  - added compact loading, empty, and error states
  - added the app logo to `apps/web/public/app-logo.png`
- Added frontend test tooling and Phase 8 tests:
  - Vitest
  - React Testing Library
  - query mapping tests
  - empty filter tests
  - infinite-scroll append helper test
  - emote fallback rendering test
- Verified Phase 8:
  - `pnpm --filter @kick-logs/web test`: 2 files, 7 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `docker compose up --build -d web`: passed
  - `GET http://localhost:3000/search`: HTTP 200
  - `GET http://localhost:3000/search?sender=yavuz&q=selam`: HTTP 200 and no admin placeholder content
  - `GET http://localhost:8000/health`: returned `{"status":"ok"}`
- Updated `README.md` and marked Phase 8 task file acceptance as complete.
- Updated `/search` date range defaults:
  - `Başlangıç` defaults to current local date/time minus 7 days.
  - `Bitiş` defaults to current local date/time.
  - clearing either date field still omits that filter from the API query.
- Added frontend tests for the default date range behavior.
- Started Phase 9 admin dashboard UI.
- Added Phase 9 auth foundation:
  - `/login` email/password UI
  - `POST /auth/login` integration through shared API client
  - compact login error state
  - safe redirect to `/admin` or local `next` path after login
  - `useCurrentUser` hook backed by `GET /auth/me`
  - `/admin` route guard redirecting unauthenticated users to `/login?next=/admin`
  - admin logout action using `POST /auth/logout`
- Added frontend tests for login success/failure, admin route guard, and logout.
- Verified Phase 9 auth foundation:
  - `pnpm --filter @kick-logs/web test`: 4 files, 14 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
- Added Phase 9 followed-channel admin UI:
  - authenticated `/admin` now mounts channel management
  - channel list calls `GET /admin/channels`
  - add form calls `POST /admin/channels` with slug/nickname and shows resolver/loading/error state
  - disable action calls `DELETE /admin/channels/{id}`
  - admin session panel shows current email, role, and active state
- Added mocked API tests for channel list/add/disable flows.
- Verified channel admin unit:
  - `pnpm --filter @kick-logs/web test`: 5 files, 17 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
- Added Phase 9 super-admin user management UI:
  - `UserAdmin` mounts only for current user role `super_admin`
  - `GET /admin/users` list shows email, role, and active state only
  - `POST /admin/users` creates new admin users
  - password hashes/secrets are not rendered
  - channel management, user management, and session summary are visually separate admin sections
- Added frontend tests for user list/create and super-admin-only visibility.
- Verified user admin unit:
  - `pnpm --filter @kick-logs/web test`: 6 files, 20 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
- Completed Phase 9 admin dashboard acceptance:
  - login and auth guard implemented
  - followed-channel management implemented
  - super-admin user management implemented
  - `/search` remains public
  - final frontend test/typecheck/lint/build passed
  - Docker `web` rebuild/start passed
  - route smoke checks for `/search`, `/login`, `/admin`, and API `/health` passed
- Completed Phase 10 final MVP smoke and cleanup:
  - backend tests and ruff passed
  - frontend tests, typecheck, lint, and build passed
  - `docker compose up --build -d` starts all services
  - API health and web `/search`, `/login`, `/admin` routes return from host
  - historical MVP root returned HTTP 307 to `/search`
  - listener logs idle status and then channel subscription status after `hype` is enabled
  - default super admin login succeeds
  - authenticated channel add stores Kick metadata for `hype`
  - sample message ingestion stores marker `phase10-smoke-20260510235338`
  - public search finds the sample message without authentication
  - PostgreSQL restart preserves the sample message in the named volume
  - README and context files now reflect final MVP startup and smoke behavior
  - no tracked generated cache, dependency folder, `.env`, secret, log, or build output was found
- Removed the unused frontend `RouteShell` scaffold and kept the MVP root behavior search-first
  until post-MVP landing work.
