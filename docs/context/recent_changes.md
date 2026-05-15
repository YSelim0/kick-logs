# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest

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
