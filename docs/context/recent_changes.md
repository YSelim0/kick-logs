# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest

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

## Previous

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
  - `GET http://localhost:3000/`: HTTP 307 to `/search`
  - `GET http://localhost:3000/search`: HTTP 200 without login
  - `GET http://localhost:3000/login`: HTTP 200
  - `GET http://localhost:3000/admin`: HTTP 200
  - `GET http://localhost:8000/health`: `{"status":"ok"}`
- Cleanup:
  - no tracked generated cache, dependency folder, `.env`, secret, log, or build output found
  - unused `RouteShell` scaffold removed
  - `/` now redirects to `/search`
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
