# Change Log

This is a living implementation log. Add new entries for each meaningful project change.

## 2026-05-16

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
