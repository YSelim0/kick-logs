# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest (issue #23 — storage hot path hardening plan)

- Active implementation plan now targets issue #23 storage hot-path hardening.
- Locked the intended storage split:
  - ClickHouse owns durable data-plane history (`chat_messages`, `raw_kick_events`,
    `raw_event_attempts`, `channel_subscription_periods`).
  - SQLite owns control-plane state plus temporary queue/inbox rows.
- Planned safe production changes:
  - sender profile cache upserts become best-effort and TTL-gated
  - processed `raw_event_queue` rows are removed after successful ClickHouse processed attempts
  - permanent invalid raw events stop retrying forever
  - processed/ignored webhook inbox rows get short retention
  - admin Operations copy/metrics clarify active queue state versus historical ClickHouse data

## Previously Latest (admin webhook status UI)

- Operations Webhooks panel now summarizes each channel's subscription state in one row instead of
  rendering three event rows inline. The summary is clickable:
  - `aktif` when all configured Kick subscription events are active.
  - `Aktif değil` when one or more event subscriptions are missing/inactive without a sync error.
  - `N Hata` when one or more event subscriptions has a sync error.
- Clicking the summary opens a design-system modal with channel metadata and per-event
  `aktif` / `aktif değil` / `hata` status rows.
- Verification: `pnpm --filter @kick-logs/web test -- webhook-health-panel.test.tsx` and
  `pnpm --filter @kick-logs/web typecheck` green.

## Previously Latest (webhook sync contract fix)

- Follow-up fix: `GET /channels/{slug}/subscription-summary` was returning zero even when
  ClickHouse had active subscription periods. Root cause was scanning ClickHouse
  `countDistinctIf(...)` (`UInt64`) into Go `int64`; the route swallowed the repository error and
  returned the zero-value response. The repository now scans unsigned counts and safely converts
  them. Local smoke: `/channels/levo/subscription-summary` returns `active_count: 1`.
- Fixed Kick webhook subscription sync against the current Kick API contract:
  - create requests now send `events: [{name, version: 1}]` plus `method: webhook`, not the old
    single `type` field.
  - missing events are created per channel in one batch request and then reconciled with a
    list-after-create fallback if Kick returns an ambiguous response.
  - delete now uses `DELETE /public/v1/events/subscriptions?id=<subscription_id>`.
  - public key auto-fetch now uses `GET /public/v1/public-key` and reads `data.public_key`.
- Webhook signature verification corrected to RSA-SHA256 over
  `message_id + "." + timestamp + "." + raw_body`; old Ed25519 assumptions are removed.
- Webhook processor now ignores events for disabled channels, preventing stale remote subscriptions
  from polluting active subscriber metrics.
- Local smoke after Docker API rebuild:
  - API fetched the Kick webhook public key successfully.
  - Enabled channels `gugucan`, `levo`, and `prensesperver` all have active
    `channel.subscription.new`, `channel.subscription.renewal`, and `channel.subscription.gifts`
    subscriptions.
- Verification: `go test ./...` and `go vet ./...` green in `apps/api-go`.

## Latest (Phase 7 — docs and smoke)

- Context docs updated for completed webhook backend (Phases 2–7):
  - `docs/architecture.md`: new endpoints added to API surface and public/admin route lists.
  - `docs/context/living_brain.md`: phase status → Phase 7 complete; API contract section updated;
    rate-limit note added for `POST /webhooks/kick` (exempt, signature-secured).
  - `docs/context/decisions.md`: 10 webhook pipeline decision entries added (broadcaster_user_id
    sentinel, inbox idempotency, rate-limit exemption, RSA-SHA256 signed message format, ClickHouse
    engine choice, expires_at NULL strategy, kick_subscription_id preservation, sync non-blocking,
    processor placement, ResolveBroadcasterUserID via web API).
  - `docs/context/change_log.md`: Phase 1–7 summary entry added.
  - `docs/operations/webhooks.md`: new runbook — Kick Developer panel setup, cloudflared tunnel,
    production Cloudflare bypass, rate-limit exemption rationale, sync/health endpoints, partial
    data window note.
- Verification: `go test ./...` green, `go vet ./...` green, `pnpm format:check` green.

## Previously Latest (Phase 6)

- Webhook subscription storage foundation (issue #22, Phase 2, branch `feat/issue-22-kick-subscription-webhooks`):
  - `domain/models.go`: added `BroadcasterUserID int64` to `FollowedChannel`; new types
    `KickWebhookEvent`, `KickEventSubscription`, `KickAPIEventSub`, `ChannelSubscriptionPeriod`,
    `ChannelSubscriptionSummary`; webhook/event-subscription status constants.
  - `ports/storage.go`: new `KickWebhookEventRepository`, `KickEventSubscriptionRepository`,
    `SubscriptionPeriodRepository` interfaces; `GetByBroadcasterUserID` added to
    `FollowedChannelRepository`.
  - `ports/kick.go`: new `KickEventSubscriptionClient` interface (impl deferred to Phase 3).
  - SQLite migrations v5 (`broadcaster_user_id` column on `followed_channels`), v6
    (`kick_webhook_events` inbox with `INSERT OR IGNORE` idempotency), v7
    (`kick_event_subscriptions` registry with UNIQUE upsert guard).
  - `infra/sqlite/followed_channels.go`: all queries updated for `broadcaster_user_id`;
    `GetByBroadcasterUserID` added.
  - `infra/sqlite/kick_webhook_events.go`: inbox repo — idempotent insert, list pending,
    mark processed/failed/ignored, count by status, latest received_at.
  - `infra/sqlite/kick_event_subscriptions.go`: registry repo — ON CONFLICT upsert,
    list/delete by channel, update sync error (soft-delete pattern).
  - ClickHouse migration v5 (`channel_subscription_periods`,
    `ReplacingMergeTree(ingested_at)` ORDER BY `id`, partitioned by month).
  - `infra/clickhouse/subscription_periods.go`: batch insert + active summary
    (`countDistinctIf` + `FINAL`).
  - Tests: new SQLite repo tests for webhook inbox transitions, subscription upsert/delete;
    new ClickHouse integration test for insert batch + active summary (gated on
    `KICK_LOGS_RUN_CLICKHOUSE_TESTS=1`).
  - Verification: `go build ./...`, `go vet ./...`, `go test ./...` all green.

## Previously Latest (Phase 3)

- Kick webhook subscription sync (issue #22, Phase 3, branch `feat/issue-22-kick-subscription-webhooks`):
  - `config.go`: 9 new fields (`KICK_CLIENT_ID`, `KICK_CLIENT_SECRET`, `KICK_API_BASE_URL`,
    `KICK_OAUTH_TOKEN_URL`, `KICK_WEBHOOK_PUBLIC_KEY`, `KICK_WEBHOOK_SYNC_ENABLED`,
    `KICK_WEBHOOK_EVENTS`, `KICK_WEBHOOK_PROCESS_BATCH_SIZE`, `KICK_WEBHOOK_PROCESS_MAX_ATTEMPTS`).
    If credentials missing, API starts and logs a warning; sync disabled.
    If public key missing, API starts and logs a warning.
  - `.env.example` and `compose.yaml` wired with all new vars.
  - `infra/kick/channel_resolver.go`: `channelPayload` now includes `user_id` and `user.id` fields;
    `ResolveChannel` populates `FollowedChannel.BroadcasterUserID`.
  - `infra/kick/event_subscription_client.go`: new `EventSubscriptionClient` implements
    `ports.KickEventSubscriptionClient`. OAuth2 client credentials with 60s expiry buffer token cache.
    `ResolveBroadcasterUserID` uses `kick.com/api/v2/channels/{slug}` (same as web resolver).
    `ListEventSubscriptions`, `CreateEventSubscription`, `DeleteEventSubscription` use official
    `KICK_API_BASE_URL/public/v1/events/subscriptions` with Bearer token.
    404 on delete is silently ignored (already gone).
  - `infra/sqlite/kick_event_subscriptions.go`: upsert SQL updated — `kick_subscription_id` is
    preserved when the new value is empty (`CASE WHEN excluded != '' THEN excluded ELSE existing`).
  - `usecase/kicksync/service.go`: new `Service` — `SyncAll` (startup, no error return),
    `EnsureChannelSubscriptions` (on channel add), `RemoveChannelSubscriptions` (on channel disable).
    Per-channel errors logged and stored in registry; sync never takes down the API.
  - `usecase/kicksync/service_test.go`: 5 tests with fake client covering resolve, create,
    skip-existing-active, delete, and error-storage scenarios.
  - `http/routes/dependencies.go`: `KickSync *kicksync.Service` added.
  - `http/routes/admin_channels.go`: channel add triggers `EnsureChannelSubscriptions` in goroutine;
    channel disable triggers `RemoveChannelSubscriptions` in goroutine.
  - `cmd/api/main.go`: wires `EventSubscriptionClient` and `kicksync.Service` when credentials
    are present; runs `SyncAll` in background goroutine on startup.
  - Verification: `go build ./...`, `go vet ./...`, `go test ./...` all green.

## Previously Latest

- README refresh:
  - Replaced the long technical README with a shorter product-focused version.
  - Added a compact centered hero with logo + title, concise slogan, demo GIF, repository/status/
    license/support badges, and lightweight repo links.
  - Removed the API endpoint catalogue and long operational sections from README. Kept the content
    focused on product purpose, user-facing capabilities, contributing, and a short self-hosting
    command flow.

- Rate limiting hardening / real-client-IP follow-up (issue #20, branch
  `feat/issue-20-rate-limiting`):
  - **Security — bypass hole closed.** `compose.yaml` now binds published api/web/ClickHouse host
    ports to `127.0.0.1` by default (`API_BIND_HOST` / `WEB_BIND_HOST` / `CH_BIND_HOST`). They were
    on `0.0.0.0`, so the app was reachable directly on the host IP, bypassing the reverse proxy and
    letting a client forge the client-IP header to defeat IP limits + hammer ClickHouse.
  - **Configurable client-IP header.** New `RATE_LIMIT_CLIENT_IP_HEADER` (default `CF-Connecting-IP`)
    in `config.go`; `ClientIP(r, trustProxy, header)` reads it, validates with `net.ParseIP` (first
    entry for `X-Forwarded-For`), falls back to `RemoteAddr`. `DefaultPolicies` + `auth.go` login key
    updated for the new signature. Lets plain-nginx self-hosters use `X-Real-IP`/`X-Forwarded-For`.
  - **Bucket isolation.** Middleware prefixes each store key with the policy `Name`, so `analytics`
    and `profile-analytics` (both 60/min burst 15 keyed by IP) no longer share one GCRA bucket.
  - **Env wiring.** All four `RATE_LIMIT_*` vars now passed through `compose.yaml` `api` env and
    listed in `.env.example` (plus the three `*_BIND_HOST` vars).
  - **GCRA** uses `context.Background()` instead of `nil` in `RateLimitCtx`.
  - **Tests.** `TestLoginRateLimit_IPBlocked` rewritten to use distinct emails per request so it
    actually exercises the IP-only policy (capacity 6) instead of tripping the IP+email limit; removed
    the dead `if i < 6 { continue }`. `config_test.go` now asserts the four rate-limit env defaults +
    overrides. Middleware tests updated for the new `ClientIP`/`DefaultPolicies` signatures.
  - **Docs.** New generic `docs/operations/reverse_proxy_and_origin.md` (loopback bind, real-IP
    header per topology, origin firewall to the CDN with SSH kept open). Skipped: login-handler
    limiter-error logging (handler already fails open).
  - Verification: `apps/api-go` `go build`/`go vet`/`go test ./internal/...` green;
    `docker compose config --quiet` green.

- Rate limiting added (issue #20, branch `feat/issue-20-rate-limiting`):
  - `internal/ports/ratelimit.go`: `RateLimiter` interface + `RateLimitResult`.
  - `internal/infra/ratelimit/gcra.go`: GCRA implementation using `throttled/throttled/v2` v2.15.0
    with `memstore.NewCtx` backend. Lazy per-config GCRA limiter creation in `sync.Map`. Store key
    prefixed with rate params to ensure cross-policy isolation.
  - `internal/http/middleware/ratelimit.go`: `ClientIP` (CF-Connecting-IP / RemoteAddr),
    `DefaultPolicies` (ordered slice, 8 policies), `RateLimit` middleware. Fail-open on limiter
    errors. 429 JSON `{"detail":"Too many requests."}` + `Retry-After` header.
  - `internal/http/routes/auth.go`: login handler applies IP+email rate limit (8/10min burst 3)
    after body parse, independent of the middleware's IP-only check.
  - `internal/http/server.go`: `RateLimit` middleware inserted between Recover and mux.
    Only applied when `cfg.RateLimitEnabled && deps.RateLimiter != nil`.
  - `cmd/api/main.go`: creates `GCRARateLimiter`, captures `tokenService` separately to pass both
    to `routes.Dependencies`.
  - `internal/config/config.go`: `RATE_LIMIT_ENABLED`, `RATE_LIMIT_STORE_MAX_KEYS`,
    `RATE_LIMIT_TRUST_PROXY` env vars.
  - Tests: 4 GCRA unit tests, 11 middleware unit tests, 3 integration tests.
  - Verification: `go vet ./...` green, `go test ./...` green (all packages).

## Previously Latest

- Prediction moved client-side (issue #19, branch `feat/issue-19-client-side-prediction`):
  - `apps/web/src/features/prediction/kick-prediction-client.ts` now fetches Kick's public endpoints
    directly from the browser. Channel validation (`/api/v2/channels/{slug}`) is cached per
    slug/browser fetch context so 5-second poll refreshes call only `.../predictions/latest`. The
    client validates the latest-prediction shape, normalizes the snake_case payload into the existing
    `Prediction` shape, derives totals/point-share/winner, and throws `ApiClientError` (channel 404 /
    null prediction → 404; blocked/non-2xx/network/malformed JSON or malformed shape → 502).
    `features/prediction/api.ts` `getPrediction(slug)` calls it; the analysis page and all its tests
    are unchanged.
  - Removed the Go prediction proxy: `domain/prediction.go`, `ports/prediction.go`,
    `infra/kick/prediction_resolver.go`, `usecase/predictions/`, `http/routes/predictions.go`,
    `http/prediction_routes_test.go`, and wiring in `cmd/api/main.go`, `http/server.go`,
    `http/routes/dependencies.go`. `GET /channels/{slug}/prediction` no longer exists.
  - Decision supersedes the 2026-05-26 backend-proxy decision; prediction is excluded from the #20
    rate-limit policy.
  - Verification: web typecheck/lint/test (21 files, 110 tests)/build green,
    `pnpm format:check` green; `apps/api-go` `go vet ./...` and `go test ./...` green. Local
    `gofmt -l cmd internal` lists Go files because of Windows CRLF working-tree line endings; no Go
    source was changed in this unit.

## Previously Latest

- Prediction analysis auto-refresh (superseded data source: current branch polls Kick directly from
  the browser):
  - `/prediction/{slug}` now polls prediction data every 5 seconds after the prediction first loads
    and keeps polling while the page is open, including after terminal states.
  - Background refreshes keep the existing summary/cards/charts mounted instead of returning to the
    loading state, so the page does not flash or redraw from scratch during live updates.
  - Recharts containers now provide positive initial dimensions and `min-w-0` wrappers to avoid
    `width(-1) and height(-1)` console warnings during first measurement.
  - Outcome cards now include point-share progress bars using each option's own color; losing
    outcomes are slightly muted once a winner exists, and top-user rows have a subtle hover
    background.
  - Channel/user identity panels hide internal channel/chatroom/count fragments that were not useful
    to public users.

- Mobile responsive site header + active-route fixes + text visibility fix (branch `feat/kick-prediction-analysis`):
  - `site-header.tsx` converted to client component; hamburger (Menu/X toggle) added for mobile. Desktop layout unchanged. Mobile shows logo + wordmark + GitHub + hamburger; clicking hamburger opens a z-40 fixed dropdown panel with all nav links (Search, Channels, Users, Prediction) and Admin. Backdrop closes menu on outside click.
  - `channel-profile-page.tsx` `activeRoute` corrected from `"search"` to `"channels"` — Channels nav item now stays highlighted on `/channels/[slug]` pages.
  - `user-profile-page.tsx` `activeRoute` corrected from `"search"` to `"users"` — Users nav item highlighted on `/users/[slug]`.
  - All `text-secondary` class usages replaced with `text-muted-foreground` in `channel-profile-page.tsx`, `user-profile-page.tsx`, and `prediction-analysis-page.tsx`. Root cause: Tailwind `text-secondary` resolves to `#24272c` (bg-elevated), making text nearly invisible on dark panels; `text-muted-foreground` is `#9ca3af` as intended.
  - `design.md` Global Header section updated.
  - Verification: lint, typecheck, test (20 files / 96 tests), build all green.

## Previously Latest (prediction feature)

- Prediction feature (branch `feat/kick-prediction-analysis`). Ported the `kick-prediction`
  prototype into kick-logs as a first-class feature, rebuilt to the project's architecture and
  design system. Three commits:
  1. `feat(api): add kick prediction fetch endpoint` — public `GET /channels/{slug}/prediction`.
     `infra/kick` prediction adapter calls Kick's undocumented latest-prediction endpoint with
     browser-like headers; `usecase/predictions` derives totals/share/winner; route maps
     not-found/channel-not-found/blocked. Live + stateless (no storage, no migration). Decided
     backend proxy over client fetch because a direct browser call hits CORS + Kick's "Request
     blocked by security policy".
  2. `feat(web): add prediction search and analysis pages` — `/prediction` (search-first,
     submit→`/prediction/{slug}`) and `/prediction/{slug}` (summary card, outcome cards,
     recharts donut + grouped-bar + horizontal top-users charts, `KAZANAN` badge,
     loading/not-found/error states, refresh). Added `recharts`, `Prediction` header nav, response
     types. Outcome cards are shown before charts and split the row into two equal columns on larger
     screens. Charts use `#22C55E` + `#C084FC` as the primary two-outcome colors, separate
     vote/return Y axes, and legends outside fixed-height chart boxes.
  3. Docs commit (design/architecture/context).
- Verification: `apps/api-go` `go build`/`go vet`/`go test ./...` green; web `typecheck`, `lint`,
  `test` (20 files, 96 tests), `build` green; `pnpm format:check` green.

## Previously Latest

- Issue #15 memory stability (branch `fix/15-memory-stability`) — fix the ~24h VPS lockup caused
  by host RAM exhaustion / swap thrash on the 4 GB box. Five commits, each a separate feature:
  1. `feat(clickhouse): cap server memory on 4GB host` — `clickhouse/config.d/memory.xml`
     (`max_server_memory_usage` ~1.2 GiB, `mark_cache_size` 256 MiB, `uncompressed_cache_size` 0)
     mounted read-only + `mem_limit: 1536m`.
  2. `feat(compose): add memory limits and restart policy to all services` — `restart:
unless-stopped` + `mem_limit` on all four (clickhouse 1536m, web 768m, listener 512m,
     api 384m).
  3. `feat(web): build and run Next.js in production mode` — multi-stage Dockerfile, `next start`,
     `NEXT_PUBLIC_API_BASE_URL` as build arg, dev bind mounts + `web_*` volumes removed.
     (`output: "standalone"` rejected: EPERM symlink failure on the Windows dev host.)
  4. `feat(api): set GOMEMLIMIT on go services` — api 307MiB, listener 410MiB (~80% of mem_limit).
  5. `feat(listener): tune ingestion batching to reduce clickhouse part pressure` — raw write
     batch 500→1000 / flush 500→1500ms, normalize batch 100→500.
  - Plus `docs/operations/vps_memory.md` runbook (budget table, swap safety net, verify steps).
- Follow-up fix on the same branch: `/messages` search now runs as a two-step ClickHouse read.
  First it ranks/filters only narrow columns to find the page of message IDs, then it fetches the
  wide response columns (`raw_payload_json`, emote arrays, reply JSON) only for those IDs. This
  prevents channel/date searches from reading wide JSON columns across the whole filtered week and
  tripping the new 1.2 GiB ClickHouse memory cap. Verified the previously failing local URL
  `GET /messages?limit=50&channel=eray&start=2026-05-17T20:28:00.000Z&end=2026-05-24T20:28:59.999Z`
  now returns 200.
- Deferred (not in this branch, to discuss): P1 SQLite `raw_event_queue` pruning
  (`MarkProcessed` still does `UPDATE`, never `DELETE`) and the host swap setup (operator does it
  manually per the runbook).
- Verification: `go test ./...` in `apps/api-go` green; previously failing `/messages` URL green;
  `pnpm --filter @kick-logs/web build` green; `docker compose build web` green + web container
  smoke (`next start`, HTTP 200, ready ~650ms); `docker compose config --quiet` green after every
  commit; `pnpm format:check` green.

## Previously Latest

- Switched channels/users index pages from debounced auto-search to explicit submit:
  - `ChannelsIndexPage` and `UsersIndexPage` now wrap the input in a `<form onSubmit>` with an
    `Ara` button. Typing alone never triggers a request; submit fires on click or Enter.
  - Submit button is disabled until the trimmed query is at least 2 characters, and while a
    request is in flight (label switches to `Aranıyor…`).
  - Empty-state message quotes the last submitted query (`submittedQuery` state) instead of the
    current input value.
  - Idle prompt updated: `Kanal/Kullanıcı adı veya slug girin ve Ara butonuna basın.`
  - Decision: ClickHouse `LIKE '%…%'` over denormalized columns is too expensive under live
    ingestion load to fire on every debounced keystroke.
  - Tests rewritten: 11 channels-index tests + 12 users-index tests (89 total frontend tests
    passing).
  - Docs updated.
- Verification:
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web test`: 18 files, 89 tests passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

## Previously Latest

- Channels and users search index pages:
  - Backend: `AnalyticsFilter.Query` field added; `topSendersWhere` and `topChannelsWhere` apply
    a LIKE `%…%` filter on denormalized username/slug/display-name columns when `q=` is set.
    `parseAnalyticsFilter` reads `q` from the URL and assigns it to `filter.Query`.
  - Frontend API: `AnalyticsQueryParams` type gets `q?: string`; `buildAnalyticsQuery` passes it
    through.
  - Frontend pages: `app/channels/page.tsx` and `app/users/page.tsx` route handlers with SEO
    metadata; `ChannelsIndexPage` and `UsersIndexPage` feature components — search-first, 300ms
    debounced, loading/empty/error states, v2 design tokens.
  - SiteHeader updated: `Channels` and `Users` nav links added; `ActiveRoute` extended.
  - Tests: `channels-index-page.test.tsx` and `users-index-page.test.tsx` (18 test files, 83
    tests total). Go HTTP route tests added for `q=` param in `analytics_q_param_test.go`.
  - `@testing-library/user-event` added as a dev dependency.
- Verification:
  - `go test ./...`: passed
  - `go vet ./...`: passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web test`: 18 files, 83 tests passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

## Previously Latest

- Responsive mobile polish follow-up:
  - `ProfileAvatar` now keeps both the real profile image and fallback initials at
    `h-[72px] w-[72px] min-w-[72px]`, so user profile photos do not collapse inside narrow
    mobile flex rows.
  - Channel/user admin tests now account for the responsive desktop + mobile DOM variants that
    intentionally render the same row data twice behind breakpoint classes.
  - Context docs were refreshed after the recent responsive pass covering profile message rows,
    the admin hamburger drawer, channel/user admin mobile rows, and operations/data-management
    panels.
- Verification:
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web test`: 16 files, 70 tests passed

## Previously Latest

- Landing page minimal footer added (`apps/web/src/features/landing/landing-page.tsx`): single-row
  footer with `Copyright` lucide icon, year, and "Tüm hakları saklıdır." in mono uppercase.
  Max height ~44px. Landing-page only.
- Frontend v2 re-skin plan archived to `docs/archive/redesign/frontend_v2_reskin_plan.md`.
  `docs/implementation_plan.md` now reflects no active plan.
- verification: pnpm typecheck/lint/test (70/70)/build green.

## Previously Latest

- Admin panel refactor (4 commits):
  - **Backend:** `/admin/channels` now returns `message_count` (ClickHouse TopChannels) and
    `last_message_at` (SQLite field). If ClickHouse is unavailable, counts default to 0.
  - **Layout:** New `apps/web/src/app/admin/layout.tsx` — sticky header (app-logo.png, breadcrumb,
    email, SUPER ADMIN badge, Çıkış) + fixed 220px sidebar with real Next.js links. Sub-pages:
    `/admin/operations`, `/admin/channels`, `/admin/users`, `/admin/data`. `/admin` redirects to
    `/admin/operations`. `admin-dashboard.tsx` deleted; auth logic now lives in layout.
  - **Channel table:** shows `profile_image_url` via `<Image>` (files.kick.com already allowed in
    next.config.mjs), formatted message count and last activity date; fallback to `—` when zero/null.
  - **UserAdmin v2:** v2 tokens, E-POSTA / GEÇİCİ PAROLA mono labels, inline input row, status pill.
  - **DataManagementPanel v2:** v2 tokens, `<h2>` heading, table rows with mono values, styled
    retention selects, warning-bordered preview card, danger-styled confirm button.
  - Tests: admin layout test rewritten; user-admin label + count assertions updated;
    data-management heading + label assertions preserved with proper `<label htmlFor>`.
  - verification: go build/test/vet green; pnpm typecheck/lint/test (70/70)/build green.

## Previously Latest

- Frontend v2 re-skin: `/admin` and `/login` now match the v2 designs (`Admin / v2`, `L9oE7`;
  `Login / v2`, `vlFSq`). All six routes are now on v2.
  - admin: new sidebar layout (Operations, Channels, Users, Data, Settings nav); top chrome
    brand + `/ admin` mono breadcrumb + user email + `SUPER ADMIN` pill + `Çıkış` button; all
    existing sections (OperationsDashboard, ChannelAdmin, UserAdmin, DataManagementPanel)
    rendered stacked in the main column; `bg-kick-background` and legacy tokens removed
  - operations dashboard: 4-card metric row (MESAJ, RAW EVENT, BAŞARISIZ RAW, DB BOYUTU) with
    mono labels and large values; status banner with Canlı/Bayat indicator; Ingestion panel
    with 6-cell strip (Queue depth, Write queue, Drop count, Flush count, Son flush, CH
    failures) and Kapalı/Açık breaker pill; all warning notices preserved
  - channel admin: new v2 table (KANAL, DURUM, MESAJ, SON AKTİVİTE columns); inline add form
    with sr-only label; direct Devre dışı bırak button in action column
  - login: centered 380px card on bg-page; brand square + `kick logs` + subtitle; E-POSTA /
    ŞİFRE uppercase mono labels; full-width accent `Giriş yap` button; muted footer link
  - tests updated: login label `Parola` → `ŞİFRE`, admin email count 2 → 1, channel admin
    kick ID assertions removed, button label `kanal ekle` → `ekle`
  - verification: pnpm typecheck/lint/test (70/70)/build all green; prettier clean

## Previously Latest

- Frontend v2 re-skin: `/users/[slug]` and `/channels/[slug]` now match the v2 profile designs
  (`User Profile / v2`, `ksyyS`; `Channel Profile / v2`, `WGYFT`).
  - both routes use the shared `SiteHeader` with the `Search` pill, v2 breadcrumbs, `bg-panel`
    identity panels, single 4-cell stats bars, three equal analytics panels, and dense latest
    message lists
  - user profiles render the circular sender avatar, `Mesajlarda ara` CTA, top channels, top
    emotes, inline emotes, reply context chips, and channel links in accent green
  - channel profiles render the rounded-square channel image, `LOGGING` pill, `Kanalda ara` CTA,
    top users, top emotes, sender-color latest rows, and Kick channel/chatroom metadata
  - user profile reply previews now reuse the `/search` reply chip styling, including the `↳`
    marker and higher-contrast replied-to sender link color
  - `docs/design/screens/` is ignored so exported design/reference PNGs do not get committed
  - verification: `pnpm --filter @kick-logs/web typecheck` and `test` (16/70) passed; lint was
    fixed by removing unused chart summary plumbing and should be rerun before commit

- Frontend v2 re-skin: `/search` is now on the v2 design (`Search Screen / v2`, `zKUtf`).
  - `apps/web/src/features/search/search-screen.tsx` switched to the shared `SiteHeader`,
    a `Search` title + mono `Tüm kanallar · Yeni → Eski` strip, single `bg-panel` form card,
    and a new results header row (`Sonuçlar` + count + `son eşleşme` timer caption)
  - `search-form.tsx` rebuilt: row 1 `Kullanıcı Adı / Kanal Adı / İçerik`, row 2
    `Başlangıç / Bitiş / Hızlı aralık`, row 3 `Sadece yanıtlar` + `Sadece emote` toggle
    pills on the left and a square export icon + `Sıfırla` outline + `Ara` accent
    primary on the right; uppercase mono field labels, `bg-elevated` inputs
  - `message-list.tsx` rebuilt without avatars: `meta` column (sender username in
    Kick sender color + `#channel` mono accent below) | message content | mono muted
    right-aligned timestamp; reply chip rendered as a muted `bg-elevated` mini-chip
  - inline `accent dot + daha eski mesajlar yükleniyor…` mono loader below the
    results card; previous `SearchSummary` aside removed
  - `apps/web/src/components/ui/input.tsx` migrated off the dropped
    `bg-kick-background` token onto v2 `bg-elevated` + `border-strong` focus
  - tests updated: `message-list.test.tsx` no longer asserts avatar links and
    expects a single `#channel` link per row
  - verification: `pnpm --filter @kick-logs/web typecheck`, `lint`, `test` (16/70),
    and `build` all green; `pnpm exec prettier --check` clean for touched files
    (pre-existing warnings in unrelated docs/admin/profile files remain)

- Frontend v2 re-skin: landing page (`/`) is now on the v2 design.
  - v2 tokens replace legacy magenta repo-wide in `apps/web/src/app/globals.css` and
    `apps/web/tailwind.config.ts`; `kick-*` color names removed
  - Geist Sans + Geist Mono wired through the `geist` npm package and `RootLayout`
  - `apps/web/src/features/landing/landing-page.tsx` rebuilt to match `Landing / v2`
    (`mRzu8`): compact hero, 4-cell stats bar, 2×2 analytics grid (volume bars, top
    channels/users/emotes), accent-green primary CTAs
  - new shared `apps/web/src/components/site-header.tsx` powers the v2 chrome (brand + active
    `Search` pill + GitHub icon + Admin outline); landing consumes it
  - landing test rewritten for the new strings, links, and empty hints; `Support` link removed
  - after later v2 passes, `/search` and profile routes are also on v2; `/admin` and `/login`
    remain to be re-skinned
  - verification: pnpm typecheck/lint/test/build green; `go build/test/vet ./...` green;
    Prettier check passed on touched files

## Previously Latest

- Issue #9 phase 7: live load test completed and all acceptance criteria met.
  - baseline run: 163,473 events at 2000 events/s over 60s (burst-factor 2, 5 channels)
  - peak writer queue depth 24,356; all 302 flushes at batch size 500; peak flush latency 392ms
  - 0 writer drops, 0 ClickHouse failures, 0 circuit breaker events
  - 1 isolated sqlite_enqueue_failure (non-critical, claim release context timeout)
  - all HTTP endpoints (`/health`, `/messages`, `/analytics/overview`) returned 200 during burst
  - two WebSocket close 1006 events on real Kick Pusher (Docker DNS blip); listener reconnected
    automatically; no ClickHouse DNS errors
  - SQLite queue backlog drained to 0 within ~90s after burst ended
  - external durable queue (RabbitMQ/NATS/Kafka) confirmed deferred: in-process pipeline
    handles 2000 events/s sustained burst without drops or 500s
  - PR `feat/issue-9-ingestion-batching` → `main` opened with load-test summary

## Previously Latest

- Issue #9 phase 7: synthetic ingestion load harness and operator runbook.
  - new `apps/api-go/cmd/loadgen` command wires the real listener service with a synthetic
    Pusher emitter that produces configurable events-per-second, supports a burst factor for
    the second half of the run, seeds `loadgen-*` channels in SQLite, and reports buffered
    writer stats periodically
  - flags: `-events-per-second`, `-duration`, `-channels`, `-burst-factor`, `-report-every`
  - new `docs/operations/load_test.md` documents prerequisites, run commands, observable
    metrics, pass/fail thresholds, and cleanup queries
  - README configuration block lists every new `LISTENER_RAW_EVENT_WRITE_*` and
    `LISTENER_CLICKHOUSE_*` knob and links to the load-test runbook
  - external durable queues (RabbitMQ/NATS/Kafka) remain deferred pending the live load run
  - verification: `go build ./...`, `go test ./...`, `go vet ./...`,
    `pnpm --filter @kick-logs/web test`, `pnpm --filter @kick-logs/web typecheck`,
    `pnpm --filter @kick-logs/web lint`, `pnpm --filter @kick-logs/web build`,
    `pnpm exec prettier --check docs/operations/ docs/tasks/`

## Previously Latest

- Issue #9 phase 6: ingestion metrics on admin operations summary.
  - listener heartbeat metadata JSON now embeds buffered writer stats (queue depth,
    high-water mark, drop count, flush count, last flush size + ms, ClickHouse failure count,
    SQLite enqueue failure count) and circuit breaker state (state, current delay ms,
    failures)
  - operations repository reads SQLite `raw_event_queue` directly for live queue depth and
    oldest pending age, then parses the heartbeat metadata to fill the new
    `domain.IngestionHealth` block
  - new HTTP `IngestionHealthResponse` is emitted under
    `operations_summary.ingestion`
  - frontend types include `IngestionHealth`; operations dashboard renders four new cards
    (Queue Backlog, Writer Buffer, ClickHouse Breaker, Son Flush) and warning banners for an
    open breaker or non-zero writer drop count
  - tests cover the ingestion cards and the breaker-open notice
  - verification: `go build ./...`, `go test ./...`, `go vet ./...`,
    `pnpm --filter @kick-logs/web test`, `pnpm --filter @kick-logs/web typecheck`,
    `pnpm --filter @kick-logs/web lint`

## Earlier Phase 5 Notes

- Issue #9 phase 5: bounded backoff and shared ClickHouse circuit breaker.
  - new `Backoff` helper produces non-decreasing delays with full jitter, capped at max
  - new `CircuitBreaker` opens after a configurable consecutive failure threshold and holds
    the open window for the current backoff delay; the next caller probes ClickHouse and the
    breaker closes on success or re-opens with a longer delay on failure
  - one breaker instance is created in `NewService` and shared by the buffered writer flush
    and the worker loop so opening it in one path pauses the other
  - workers call `breaker.Wait(ctx)` before each tick and record success/failure based on the
    tick outcome; the buffered writer wraps each ClickHouse batch insert attempt
  - env knobs: `LISTENER_CLICKHOUSE_BACKOFF_INITIAL_MS` (1000),
    `LISTENER_CLICKHOUSE_BACKOFF_MAX_MS` (30000),
    `LISTENER_CLICKHOUSE_BACKOFF_MULTIPLIER` (2), and
    `LISTENER_CLICKHOUSE_BREAKER_FAILURE_THRESHOLD` (5)
  - added unit tests for backoff growth, reset, breaker open-after-threshold,
    probe-success-closes, probe-failure-re-opens-with-longer-delay,
    context-cancellation, and shared-breaker waiting
  - verification: `go build ./...`, `go test ./...`, `go vet ./...`

## Earlier Phase 4 Notes

- Issue #9 phase 4: worker batch normalization output.
  - worker tick claims all pending queue rows, loads each raw payload from ClickHouse by id,
    normalizes the message in memory, then writes the entire tick as one
    `InsertMessagesBatch` and one `InsertAttemptsBatch`
  - tick-internal dedupe via `seenKickMessageIDs` prevents two queue rows with the same
    `kick_message_id` from inserting a duplicate visible message in the same batch
  - ClickHouse `InsertMessagesBatch` failure releases every claimed queue row back to pending
    so the next tick retries the full set
  - attempts batch is best-effort: a failure logs and the worker continues marking the queue
    rows so audit gaps recover on the next attempt
  - removed the per-row `markRawEventProcessed`/`markRawEventFailed`/`recordAttempt` helpers;
    `prepareMessage` is now a pure normalization function used by the worker
  - listener service tests cover one-batch-per-tick, mixed processed/failed in a single tick,
    and full claim-release on ClickHouse batch failure
  - verification: `go build ./...`, `go test ./...`, `go vet ./...`

## Earlier Phase Notes

- Issue #9 phase 3: buffered websocket raw-event writer.
  - new `bufferedRawWriter` accepts raw events from the websocket callback through an
    in-memory channel and flushes batches to ClickHouse via `InsertEventsBatch`, then enqueues
    the same batch into SQLite `raw_event_queue`
  - flush triggers: batch size threshold, flush interval, or shutdown drain
  - shutdown drains the buffer before returning so context cancellation does not lose events
  - in-memory queue full drops the oldest event with a warning and a counter; the writer
    still drops the incoming event when both the channel and the eviction attempt are full
  - ClickHouse batch failures retry with bounded exponential delay up to
    `LISTENER_RAW_EVENT_WRITE_MAX_RETRIES`; the batch is dropped after all retries fail and a
    drop counter is incremented
  - SQLite enqueue failures after a successful ClickHouse archive retry indefinitely so an
    archived event always lands in the queue
  - added env vars `LISTENER_RAW_EVENT_WRITE_BATCH_SIZE`,
    `LISTENER_RAW_EVENT_WRITE_FLUSH_INTERVAL_MS`, `LISTENER_RAW_EVENT_WRITE_QUEUE_SIZE`, and
    `LISTENER_RAW_EVENT_WRITE_MAX_RETRIES` with defaults 500, 500ms, 50000, 10
  - `.env.example` exposes the new knobs
  - `Service.WriterStats()` returns queue depth, high-water mark, drop count, flush count,
    last flush size, last flush nanos, ClickHouse failure count, and SQLite enqueue failure
    count for future operations dashboard fields
  - listener service `RunForever` now starts the writer goroutine before bootstrap; tests
    short-circuit the writer to keep the synchronous insert/enqueue path under coverage
  - verification: `go build ./...`, `go test ./...`, `go vet ./...`, `pnpm exec prettier
--check docs/tasks/`

## Older Phase Notes

- Issue #9 phase 1: moved the raw-event work queue out of ClickHouse into SQLite.
  - added migration v4 creating `raw_event_queue` with status, attempts, claim ownership,
    enqueued_at, last_error, and indexes for pending scan and stale-claim recovery
  - added `domain.RawEventQueueItem` plus status constants and `ports.RawEventQueueRepository`
  - added SQLite `RawEventQueueRepository` covering enqueue, batch enqueue, list pending in
    FIFO order, claim/release, mark processed/failed with max-attempts exhaustion, count
    pending, oldest pending age, stale-claim recovery, and lookup by id
  - added `RawEventRepository.GetByID` on ClickHouse so workers can load the raw payload by id
  - listener websocket callback now enqueues the queue row alongside the ClickHouse archive
    insert; failure to enqueue is returned as a callback error
  - worker tick now lists, claims, and counts pending work from SQLite, removing the heavy
    `raw_event_attempts` GROUP BY + LEFT JOIN ClickHouse query from the hot path
  - `RawEventClaimRepository` is no longer wired into the listener because the queue
    repository fully covers the claim contract
  - listener bootstrap backfills any unprocessed ClickHouse raw events into the queue and runs
    a stale-claim recovery sweep; a periodic background loop repeats the sweep
  - listener tests now exercise the queue path with a faithful in-memory fake
  - verification: `go build ./...`, `go test ./...`, `go vet ./...`, `pnpm exec prettier
--check docs/tasks/`

## Earlier Latest

- Finalized Go + ClickHouse cutover cleanup:
  - archived the completed Go rewrite implementation plan, task files, and API contract inventory
    under `docs/archive/go_rewrite/`
  - removed the Python/FastAPI application from `apps/api`
  - removed PostgreSQL and Python runtime services/volumes from Docker Compose
  - kept `migrate-go` as the legacy PostgreSQL import tool under the `tools` profile
  - replaced Python CI with Go CI that runs on every push and pull request
  - kept code-style CI running on every push and pull request
  - updated README, architecture, project plan, context, and active implementation plan to make Go
    - ClickHouse + SQLite the only current runtime
  - local migration into fresh ClickHouse/SQLite succeeded with 2 admin users, 7 followed
    channels, 8570 sender profiles, 1 retention setting, 1 heartbeat, 123790 chat messages,
    121664 raw events, and 121664 raw-event attempts
  - legacy PostgreSQL volume was intentionally left untouched
  - verification: `go test ./...`, `go vet ./...`, live ClickHouse repository test,
    `pnpm format:check`, `git diff --check`, `docker compose config --services`,
    `docker compose --profile tools config --services`, `docker compose up --build -d
--remove-orphans`, API health, latest-message search, default admin login, and admin channel
    list smoke checks

- Completed Go rewrite Phase 9 cutover:
  - default Compose services now use Go `api`, Go `listener`, `clickhouse`, and `web`
  - Python/FastAPI/PostgreSQL remains available through the `python-reference` profile as a
    reference/rollback runtime
  - `migrate-go` is now a `tools` profile service for PostgreSQL to SQLite/ClickHouse migration
  - `.env.example`, README, architecture, project plan, implementation plan, decisions, living
    context, and Phase 9 task docs now reflect the Go + ClickHouse default runtime
  - before cutover, Go admin data-management parity was added for summary, retention settings,
    cleanup preview, and cleanup confirmation
  - fixed a Go listener SQLite sender-profile upsert race that could fail on concurrent
    `sender_profiles.slug` conflicts during live ingestion
  - verification: `go test ./...`, `go vet ./...`, live ClickHouse repository test,
    `docker compose up --build -d --remove-orphans`, Go API/listener/web smoke, live/fixture
    searchable message checks, reply/emote metadata checks, JSON/CSV export, analytics/profile
    API checks, frontend route smoke, admin operations/data-management smoke, cleanup preview, and
    unauthenticated admin rejection

- Completed Go rewrite Phase 8 PostgreSQL data migration:
  - added `cmd/migrate -target=data` with dry-run, execute, validation-only, batch size, sample
    size, and source PostgreSQL URL flags
  - added read-only PostgreSQL source adapter that accepts Python `postgresql+asyncpg://` URLs
  - added idempotent SQLite and ClickHouse migration writers that preserve source IDs where API
    rows expose them
  - migrated users, followed channels, senders, retention settings, heartbeats, chat messages, raw
    events, and deterministic raw-event attempt rows
  - validate bcrypt admin hashes, JSONB serialization, UTC timestamps, counts, and representative
    samples before accepting a run
  - record execute/validation run metadata in SQLite `data_migration_runs`
  - closed `docs/tasks/go_rewrite_08_data_migration.md`
  - verification: `go test ./...`, `go vet ./...`, live ClickHouse repository test,
    `docker compose --profile go-rewrite build migrate-go`, and `migrate-go` SQLite/ClickHouse
    command smoke checks

- Completed Go rewrite Phase 7 analytics/profile parity:
  - added public Go analytics endpoints for overview, message volume, top senders, top channels,
    and top emotes
  - added ClickHouse aggregate repository queries over denormalized `chat_messages`
  - preserved date filters, exact channel scope, sender `_`/`-` lookup variants, hour/day buckets,
    top-list limit validation, and public access
  - added public Go user and channel profile analytics routes with SQLite identity metadata,
    overview, day-bucket volume, top lists, and latest messages in the existing message shape
  - closed `docs/tasks/go_rewrite_07_analytics_profiles.md`
  - verification: `go test ./...`, `go vet ./...`, live ClickHouse repository test, Docker Go API
    build, analytics route smoke, invalid-range smoke, and unknown-profile 404 smoke

- Completed Go rewrite Phase 6 listener ingestion parity:
  - wired `cmd/listener` to SQLite, ClickHouse, Kick resolvers, Pusher websocket, raw workers, and
    heartbeat recording
  - added Go Pusher subscriptions for `chatrooms.{chatroom_id}.v2` plus channel-level streams
  - stored `App\Events\ChatMessageEvent` raw payloads in ClickHouse before normalization
  - added raw-event retry processing, `kick_message_id` dedupe, sender-profile upsert, and
    normalized ClickHouse message inserts
  - preserved reply metadata, emote arrays/image URLs, badges, sender color, timestamps, and raw
    payload JSON for frontend-compatible search results
  - added `listener-go` to the `go-rewrite` Compose profile and closed
    `docs/tasks/go_rewrite_06_listener_ingestion.md`
  - verification: `go test ./...`, `go vet ./...`, live ClickHouse repository test, Docker Go API
    and listener build/smoke, authenticated operations summary smoke, and listener log smoke

- Completed Go rewrite Phase 5 message search/export parity:
  - added ClickHouse-backed public `GET /messages`
  - added public `GET /messages/export` with JSON and CSV output
  - preserved sender exact matching, channel/content contains matching, date range filters,
    reply-only, emote-only, newest-first ordering, and existing cursor shape
  - expanded ClickHouse message snapshots for nested sender/channel/badge/reply response fields
  - wired Go API startup to create the message repository/use case when ClickHouse is reachable
  - closed `docs/tasks/go_rewrite_05_messages_search_export.md`
  - verification: `go test ./...`, `go vet ./...`, live ClickHouse repository test,
    Docker Go API smoke for `/messages` and `/messages/export`, `pnpm format:check`, and
    `git diff --check`

- Completed Go rewrite Phase 4 auth/admin API parity:
  - added Go bcrypt password hasher and JWT token service
  - preserved auth cookie settings, expiry, HttpOnly behavior, SameSite, Secure, and session user
    response shapes
  - implemented `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`
  - implemented admin middleware, super-admin checks, `GET /admin/users`, and `POST /admin/users`
  - added Go Kick web channel resolver and admin followed-channel list/add/disable routes
  - added basic `GET /admin/operations/summary` using SQLite control-plane counts and ClickHouse
    data-plane counts when available
  - Go API startup now applies SQLite migrations, seeds the default super admin, and applies
    ClickHouse migrations when ClickHouse is reachable
  - closed `docs/tasks/go_rewrite_04_auth_admin_api.md`
  - verification: `go test ./...`, `go vet ./...`, Docker Go API smoke for login/me/users/ops,
    Docker Go API rebuild, `pnpm format:check`, and `git diff --check`

## Previous

- Completed Go rewrite Phase 3 storage/schema:
  - added SQLite and ClickHouse configuration defaults for the Go runtime
  - added versioned migration runners for both stores
  - added SQLite control-plane schema for admin users, followed channels, sender profiles,
    retention settings, worker heartbeats, and migration bookkeeping
  - added ClickHouse schema for denormalized chat messages, raw Kick events, and raw-event
    attempts
  - added repository interfaces plus concrete SQLite/ClickHouse repositories and storage stats
  - added SQLite default super-admin seeding with bcrypt hashes
  - added Compose `clickhouse` and `migrate-go` services behind profile `go-rewrite`
  - closed `docs/tasks/go_rewrite_03_storage_schema.md`
  - verification: `go test ./...`, targeted live ClickHouse repository test,
    `docker compose --profile go-rewrite run --rm migrate-go`, and Go Docker image builds

- Completed Go rewrite Phase 2 workspace/tooling:
  - added `apps/api-go` with `cmd/api`, `cmd/listener`, `cmd/migrate`, config, app bootstrap,
    stdlib HTTP server, middleware, health route, and package skeletons
  - added an optional Docker Compose `api-go` service behind profile `go-rewrite`
  - documented Go rewrite local commands in README and current architecture notes
  - closed `docs/tasks/go_rewrite_02_workspace_tooling.md`
  - verification: `go test ./...`, `go vet ./...`, local binary `GET /health`, Docker image
    build, `pnpm format:check`, and `git diff --check`

- Completed Go rewrite Phase 1 contract inventory:
  - added `docs/contracts/api_contract.md`
  - added representative JSON fixtures under `docs/contracts/fixtures/`
  - documented endpoint access, request bodies, query params, response shapes, auth cookie behavior,
    cursor format, CSV export columns, search matching rules, reply metadata, and emote fields
  - closed `docs/tasks/go_rewrite_01_contract_inventory.md`
  - verification: `python -m uv run pytest` reported 72 passed and 52 skipped, `pnpm format:check`
    passed, and `git diff --check` passed

- Started the Go + ClickHouse rewrite planning track:
  - archived completed MVP docs under `docs/archive/mvp/`
  - archived completed post-MVP docs under `docs/archive/post_mvp/`
  - replaced the active implementation plan with the Go API/listener rewrite plan
  - added phase task files for contract inventory, Go workspace, storage, auth/admin, search,
    listener, analytics, migration, and cutover

- Fixed Docker Compose backend env passthrough for release readiness:
  - API now receives `.env` overrides for database echo, JWT algorithm/expiry/cookie settings,
    and super-admin seed behavior
  - listener now receives `DATABASE_ECHO`
  - verified with `docker compose config`

- Completed Post-MVP Feature 8 final smoke and documentation:
  - backend tests/Ruff checks passed after hardening live-data-sensitive assertions
  - frontend tests/typecheck/lint/build and `pnpm format:check` passed
  - `docker compose up --build -d` starts `postgres`, `api`, `listener`, and `web`
  - live smoke passed for public landing/search/profile/analytics/export routes, authenticated
    operations/data-management APIs, and unauthenticated admin API rejection
  - README project status and archived MVP docs were updated for the completed post-MVP state
  - `docs/tasks/post_mvp_08_final_smoke.md` is fully checked off
- Verification:
  - `python -m uv run pytest`: 124 passed
  - `python -m uv run ruff check .`: passed
  - `python -m uv run ruff format --check .`: passed
  - `pnpm --filter @kick-logs/web test`: 16 files, 66 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

- Completed Post-MVP Feature 7 data management:
  - README documents data-management usage, retention behavior, guarded cleanup, and Docker
    Compose PostgreSQL backup/restore
  - `docs/tasks/post_mvp_07_data_management.md` is fully checked off
  - destructive cleanup requires dry-run preview plus exact confirmation text
- Verification:
  - `python -m uv run pytest`: 124 passed
  - `python -m uv run ruff check .`: passed
  - `python -m uv run ruff format --check .`: passed
  - `pnpm --filter @kick-logs/web test`: 16 files, 66 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

- Implemented the frontend for Post-MVP Feature 7 data management:
  - `/admin` now includes `DataManagementPanel` below operations status
  - panel shows database/table sizes and retention settings
  - retention controls support keep forever, 30 days, and 90 days
  - cleanup requires dry-run preview and exact confirmation text before delete
  - success/error states show deleted rows or API failures
- Verification:
  - targeted frontend data-management/admin tests: 8 passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed

- Implemented the backend foundation for Post-MVP Feature 7 data management:
  - `data_retention_settings` persists message/raw-event retention windows
  - retention defaults to keep forever with `null` values
  - admin-only summary endpoint returns counts, table sizes, DB size, and retention settings
  - admin-only retention update endpoint accepts `null`, `30`, or `90`
  - cleanup preview/confirm endpoints cover old messages, old raw events, channel, and sender
  - destructive cleanup requires exact preview confirmation text
- Verification:
  - targeted backend data-management/migration/metadata tests: 13 passed
  - `python -m uv run ruff check .`: passed

- Completed Post-MVP Feature 6 channel/publisher profiles:
  - README now documents `/channels/[slug]` and `GET /channels/{slug}/analytics`
  - `docs/tasks/post_mvp_06_channel_profiles.md` is fully checked off
  - visitors can inspect a logged channel's metadata/activity and jump to
    `/search?channel={slug}`
- Verification:
  - `python -m uv run pytest`: 119 passed
  - `python -m uv run ruff check .`: passed
  - `python -m uv run ruff format --check .`: passed
  - `pnpm --filter @kick-logs/web test`: 15 files, 61 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

- Implemented the frontend for Post-MVP Feature 6 channel profiles:
  - public `/channels/[slug]`
  - typed channel profile API wrapper and response types
  - profile UI shows channel summary, activity metrics, volume bars, top senders, top emotes,
    latest messages, loading, empty, error, and not-found states
  - profile links to `/search?channel={slug}`
  - `/search` channel labels and `/admin` channel rows link to `/channels/[slug]`
- Verification:
  - `pnpm --filter @kick-logs/web test`: 15 files, 61 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed

- Implemented the backend API for Post-MVP Feature 6 channel profiles:
  - public `GET /channels/{slug}/analytics`
  - response includes stored channel metadata, overview totals, day-bucket message volume, top
    senders, top emotes, and latest messages
  - unknown channel slugs return 404
  - latest profile messages use exact channel-id lookup
- Verification:
  - targeted backend channel profile/analytics/search tests: 18 passed
  - `python -m uv run ruff check .`: passed
  - `python -m uv run ruff format --check .`: passed

- Fixed Kick profile slug handling for underscore usernames:
  - visible chat usernames stay unchanged, such as `example_user`
  - sender/profile links now route to Kick-style profile slugs, such as `/users/example-user`
  - reply preview sender links use the same profile slug behavior
  - backend sender/profile/search/analytics lookups accept both underscore and hyphen forms for
    compatibility with existing stored data
- Verification:
  - targeted backend slug/search/profile tests: 28 passed
  - `python -m uv run ruff check .`: passed
  - `python -m uv run ruff format --check .`: passed
  - `pnpm --filter @kick-logs/web test`: 14 files, 56 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

- Polished reply-profile navigation and user profile panel styling:
  - muted replied-to sender names in `/search` reply previews link to `/users/[slug]`
  - reply metadata now uses `original_sender.slug` when available and falls back to a lowercase
    username-derived profile slug
  - the `/users/[slug]` top identity panel now matches other bordered/padded profile sections
- Verification:
  - `pnpm --filter @kick-logs/web test`: 13 files, 54 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

- Implemented Post-MVP Feature 5 user profile analytics:
  - added public `GET /users/{slug}/analytics`
  - endpoint returns sender identity/profile image, overview totals, day-bucket message volume,
    top channels, top emotes, and latest messages
  - unknown sender slugs return 404
  - added public `/users/[slug]` profile pages
  - search result sender names and avatars now link to user profiles
  - profile pages link to `/search?sender={slug}`
  - `docs/tasks/post_mvp_05_user_profiles.md` is fully checked off
- Verification:
  - `python -m uv run pytest`: 113 passed
  - `python -m uv run ruff check .`: passed
  - `python -m uv run ruff format --check .`: passed
  - `pnpm --filter @kick-logs/web test`: 13 files, 53 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

- Implemented Post-MVP Feature 4 landing page with analytics:
  - root `/` now renders a compact public landing page instead of redirecting to `/search`
  - landing uses Feature 3 analytics endpoints for overview, recent day-bucket volume, top
    channels, top emotes, and top senders
  - navigation links point to `/search`, `/admin`, GitHub, and Buy Me a Coffee support
  - `/search` and `/admin` header brand/logo areas now navigate back to `/`
  - loading, API-error, and fresh-install empty states are covered
  - `docs/tasks/post_mvp_04_landing_analytics.md` is fully checked off
- Verification:
  - `pnpm --filter @kick-logs/web test`: 12 files, 50 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed
  - `docker compose up --build -d web`: passed
  - `GET http://localhost:3000/`: HTTP 200
  - `GET http://localhost:3000/search`: HTTP 200

- Implemented Post-MVP Feature 3 analytics foundation:
  - added public read-only analytics endpoints for overview, message volume, top senders, top
    channels, and top emotes
  - added reusable analytics DTOs, use cases, repository port, and SQLAlchemy aggregate repository
  - analytics filters support date range plus exact sender/channel scope
  - added typed frontend analytics API wrappers and parameter mapping tests
  - documented the analytics API shape in README, architecture, project plan, and context docs
- Verification:
  - `python -m uv run pytest`: 111 passed
  - `python -m uv run ruff check .`: passed
  - `python -m uv run ruff format --check .`: passed
  - `pnpm --filter @kick-logs/web test -- analytics/api.test.ts`: 1 file, 3 tests passed
  - `pnpm --filter @kick-logs/web test`: 11 files, 47 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

- Polished the `/search` filter form density:
  - date presets moved from four separate buttons to one compact `Hızlı aralık` select
  - export moved behind one square `Dışa aktar` icon button with `JSON indir` and `CSV indir`
  - export menu closes on outside click
  - result-type filters now read `Sadece yanıtlar` and `Sadece emote`
  - result-type filters moved below date controls, to the left of the `İşlem` action group
  - design/context docs describe the compact control behavior
- Verification:
  - `pnpm --filter @kick-logs/web test -- search-screen.test.tsx`: 1 file, 8 tests passed
  - `pnpm --filter @kick-logs/web test`: 10 files, 44 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed

- Implemented Post-MVP Feature 2 public search UI improvements:
  - search form now has date presets, reply-only, and emote-only controls
  - `/search` URL state preserves the new filters
  - message content renders clickable links and highlights matched `q` text without moving inline emotes
  - CSV and JSON export buttons open filtered exports for the last submitted search
  - `docs/design/design.md` and the Feature 2 task file document the UI behavior
- Verification:
  - `pnpm --filter @kick-logs/web test`: 10 files, 42 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `pnpm format:check`: passed
  - `python -m uv run ruff check .`: passed
  - `python -m uv run pytest tests/domain/test_value_objects.py tests/test_config.py tests/messages/test_http_search_messages.py`: 18 passed
- Completed Post-MVP Feature 2 acceptance in `docs/tasks/post_mvp_02_search_improvements.md`.

- Implemented Post-MVP Feature 2 backend search/export foundation:
  - public `GET /messages` now supports `reply_only` and `emote_only`
  - public `GET /messages/export` returns filtered JSON or CSV without auth
  - export reuses the same `MessageSearchFilters` semantics and clamps rows with
    `MESSAGE_EXPORT_MAX_ROWS`
  - README, project plan, architecture, and task docs describe the new backend contract
- Verification:
  - `python -m uv run ruff check .`: passed
  - `python -m uv run pytest tests/domain/test_value_objects.py tests/test_config.py tests/messages/test_http_search_messages.py`: 18 passed

- Added a Post-MVP Feature 2 task for clickable message links:
  - URLs inside `/search` message content should render as safe clickable anchors
  - link rendering must not break inline emotes or future matched-text highlighting
  - Feature 2 tests now explicitly include clickable link rendering

- Completed Post-MVP Feature 1 admin operations dashboard:
  - README documents the operations dashboard and `GET /admin/operations/summary`
  - all checkboxes in `docs/tasks/post_mvp_01_admin_operations.md` are closed
  - `/admin` lets an authenticated admin understand storage growth, raw backlog/status, and
    listener freshness without reading Docker logs
- Final verification for the feature:
  - `python -m uv run pytest`: 101 passed
  - `python -m uv run ruff check .`: passed
  - `pnpm --filter @kick-logs/web test`: 10 files, 36 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm format:check`: passed

- Implemented Post-MVP Feature 1 admin operations UI:
  - added typed `getOperationsSummary` frontend API wrapper
  - `/admin` now shows `OperationsDashboard` above channel/user management
  - compact cards show listener status, DB size, message count, raw event count, failed raw,
    pending raw, and last ingest time
  - manual refresh, stale listener warning, failed raw warning, and API error states are tested
- Verification:
  - `pnpm --filter @kick-logs/web test`: 10 files, 36 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed

- Implemented Post-MVP Feature 1 backend operations foundation:
  - added `worker_heartbeats` persistence and migration `20260513_0003`
  - listener writes a periodic `listener` heartbeat
  - added admin-only `GET /admin/operations/summary`
  - summary includes listener freshness, row counts, raw event status counts, DB/table sizes,
    latest ingest timestamps, and oldest pending raw event timestamp
  - `.env.example` and Compose expose listener heartbeat interval/staleness settings
- Verification:
  - `python -m uv run alembic upgrade head`: applied `20260513_0003`
  - `python -m uv run ruff check .`: passed
  - `python -m uv run pytest`: 101 passed

- Updated GitHub validation workflow triggers:
  - `Code Style` and `Python CI` now run for pull requests targeting `main` or `dev`
  - both workflows now run on pushes to `main` or `dev`
  - README CI wording now reflects `main` and `dev`

- Changed public message search sender filtering:
  - `sender` now matches sender username/slug exactly, case-insensitively
  - partial sender searches such as `yavuz` no longer return `notyavuz` or `yavuz123`
  - channel and content filters still use contains matching
  - backend tests cover exact sender matches and rejected partial matches

- Archived the completed MVP plan and added the active post-MVP roadmap:
  - old `docs/implementation_plan.md` moved to `docs/archive/mvp_implementation_plan.md`
  - old `docs/tasks/phase*_tasks.md` files moved to `docs/archive/tasks/`
  - new `docs/implementation_plan.md` covers post-MVP feature work
  - new active task files live under `docs/tasks/post_mvp_*.md`
  - selected roadmap: admin operations, search improvements, analytics foundation, landing analytics, user profiles, channel profiles, data management, final smoke/docs

- Added Buy Me a Coffee sponsorship metadata:
  - `.github/FUNDING.yml` now enables the GitHub Sponsor button for `yavuzselim`
  - README now shows a Buy Me a Coffee badge and a short `Support` section
  - support URL: `https://buymeacoffee.com/yavuzselim`

- Fixed `/search` date filtering and favicon:
  - the site favicon now uses `/app-logo.png`
  - search URL params keep local `datetime-local` values for the date inputs
  - backend `/messages` requests receive UTC ISO `start`/`end` values
  - `Bitiş` includes the whole selected minute, so selected ranges include messages up to `:59.999`
  - ISO date URL values normalize back to local input values

- Added repository formatting standards:
  - root Prettier config matches the existing frontend style: 2 spaces, semicolons, double quotes, no trailing commas, 100-column print width
  - root Prettier scripts: `pnpm format` and `pnpm format:check`
  - `.prettierignore` excludes generated/runtime files, locks, `.pen`, and local agent skills
  - Python formatting uses Ruff Format with 100-column line width, double quotes, spaces, and LF line endings
  - added Code Style GitHub Actions workflow for `pnpm format:check`
  - backend Python CI now also checks `ruff format --check .`
  - normalized existing frontend/docs/Python files with Prettier and Ruff Format

- Added backend GitHub Actions workflow:
  - `.github/workflows/python-tests.yml` runs on pull requests and pushes to `main`
  - starts PostgreSQL 16 service
  - installs backend dependencies with `uv`
  - applies Alembic migrations
  - runs `ruff check .` and `pytest`
  - README now includes the Python CI badge and CI section

- Rewrote root `README.md` as a public-facing project page:
  - added centered app logo and repository links
  - documented product purpose, features, stack, quick start, usage, services, API surface, development commands, configuration, contribution flow, and operational notes
  - added clear fork/contribution guidance for community development
- Added MIT `LICENSE` file and linked it from the README.

- Implemented GitHub issue #3 reply rendering on branch `feat/issue-3-kick-reply-rendering`:
  - backend tests now lock the observed Kick reply payload shape (`type="reply"`, `metadata.original_sender`, `metadata.original_message`, `thread_parent_id`)
  - public `/messages` test verifies reply fields are returned unchanged
  - `/search` result rows render replied-to sender/content above the current message in muted gray text
  - long reply previews expose full original content through a `title` attribute
- Verification:
  - `pnpm --filter @kick-logs/web test`: 9 files, 28 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `python -m uv run ruff check .`: passed
  - `python -m uv run pytest`: 96 passed

- Updated public `/search` initial-load behavior:
  - bare `/search` does not call `/messages` automatically anymore
  - results area shows `Arama yapmak için yukarıdaki formu kullanın.`
  - URL query params still trigger a search on load
  - explicit empty search still fetches latest messages
- Verification:
  - `pnpm --filter @kick-logs/web test`: 7 files, 23 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed

- Implemented GitHub issue #1 durable Kick ingestion on branch `feature/issue-1-durable-inbox`:
  - added `raw_kick_events` domain entity/status, SQLAlchemy model, Alembic migration, repository port, and repository implementation
  - listener websocket path now stores raw chat events first instead of normalizing/inserting messages inline
  - raw event workers claim pending/stale rows in batches with `FOR UPDATE SKIP LOCKED`
  - failed raw events retain payload, attempts, and last error
  - duplicate raw processing remains safe because `IngestMessageUseCase` deduplicates by Kick message id
  - listener reconnects periodically to refresh followed-channel subscriptions
- Verification:
  - `python -m uv run ruff check .`: passed
  - `python -m uv run alembic upgrade head`: applied `20260511_0002`
  - `python -m uv run alembic current`: `20260511_0002 (head)`
  - `python -m uv run pytest`: 94 passed
  - `python -m uv run pytest tests/listener tests/domain tests/database/test_models_metadata.py tests/database/test_alembic_migration.py`: 43 passed
  - `python -m uv run pytest tests/database/test_repositories.py tests/messages/test_ingest_message.py tests/listener/test_listener_service.py`: 19 passed
  - `docker compose up --build -d postgres api listener`: passed
  - `GET http://localhost:8000/health`: `{"status":"ok"}`
  - listener logs show raw event storage and worker processing with `pending=0`

## Earlier

- Fixed `/search` hydration mismatch caused by timezone-dependent default date values:
  - first render now uses static empty search state
  - default 7-day local date range is applied after client hydration
  - restarted `web` and confirmed server HTML no longer includes default `datetime-local` values
- Verification:
  - `pnpm --filter @kick-logs/web test`: 6 files, 20 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed

## Older

- Fixed browser CORS for frontend-to-backend API calls:
  - FastAPI now installs `CORSMiddleware` from comma-separated `BACKEND_CORS_ORIGINS`
  - `OPTIONS /auth/login` from `http://localhost:3000` returns `200`
  - `Access-Control-Allow-Origin: http://localhost:3000`
  - `Access-Control-Allow-Credentials: true`
  - actual `POST /auth/login` returns `200` and sets `kick_logs_session`
- Hardened the message repository pagination test so existing local chat history cannot pollute its `q` filter.
- Verification:
  - `python -m uv run pytest`: 85 passed
  - `python -m uv run ruff check .`: passed
  - live Docker preflight and login smoke passed against `http://localhost:8000`

## Oldest

- Phase 10 final MVP smoke and cleanup are complete.
- Latest smoke:
  - full Docker stack starts with `docker compose up --build -d`
  - default super admin login succeeds
  - authenticated channel add stores Kick metadata for `hype`
  - sample message marker `phase10-smoke-20260510235338` was ingested through the backend use case
  - public `/messages` search finds the sample without authentication
  - PostgreSQL restart preserves the sample message
  - listener logs channel subscription status after `hype` is enabled
- Verification:
  - `python -m uv run pytest`: 83 tests passed
  - `python -m uv run ruff check .`: passed
  - `pnpm --filter @kick-logs/web test`: 6 files, 20 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `docker compose up --build -d`: passed
  - historical MVP `GET http://localhost:3000/`: HTTP 307 to `/search`
  - `GET http://localhost:3000/search`: HTTP 200 without login
  - `GET http://localhost:3000/login`: HTTP 200
  - `GET http://localhost:3000/admin`: HTTP 200
  - `GET http://localhost:8000/health`: `{"status":"ok"}`
- Cleanup:
  - no tracked generated cache, dependency folder, `.env`, secret, log, or build output found
  - unused `RouteShell` scaffold removed
  - MVP root behavior was search-first before the post-MVP landing page
  - README and context files updated for final MVP state

## Commit Context

- Previous committed units:
  - `3c9178b feat(backend): complete phase six acceptance`
  - `c8c5eb9 feat(web): scaffold frontend foundation`
  - `f2250a9 feat(docs): complete phase seven foundation`
  - `619f4f9 feat(search): add public message search ui`
  - `2ab7c91 feat(search): default date range filters`
  - `813d713 feat(auth): add admin login guard`
  - `823a8ee feat(admin): add channel management ui`
  - `43b03db feat(admin): add user management ui`
- Latest completed unit:
  - Phase 10 final MVP smoke and cleanup
- Commit message for this unit:
  - `feat(docs): complete phase ten smoke`
