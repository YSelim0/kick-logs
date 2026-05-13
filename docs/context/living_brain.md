# Living Brain

This file is the active project memory. Keep it updated whenever project behavior, architecture, implementation details, or working assumptions change.

## Current State

- Repository `kick-logs` has been initialized locally.
- Commit convention skill exists under `.agents/skills/commit-message-conventions`.
- Phase 1 backend/Docker foundation is complete.
- Phase 2 persistence foundation is complete.
- Phase 3 auth/admin user foundation is complete.
- Phase 4 channel management unit is implemented and committed.
- Phase 4 message ingestion foundation is implemented and committed.
- Phase 4 public message search API is implemented and committed.
- Phase 4 acceptance is complete.
- Phase 5 listener foundation is complete.
- Phase 5 listener runtime and Docker service are complete.
- Phase 6 backend verification and acceptance is complete.
- Phase 7 frontend foundation is complete.
- Phase 8 public search UI is complete.
- Phase 9 admin UI is complete.
- Phase 10 final MVP smoke and documentation cleanup are complete.
- The original MVP implementation plan and phase task files are archived under `docs/archive/`.
- `docs/implementation_plan.md` now tracks the active post-MVP feature roadmap.
- Active post-MVP task files are under `docs/tasks/post_mvp_*.md`.
- Selected post-MVP roadmap:
  - admin operations dashboard
  - search improvements
  - analytics foundation
  - landing page with analytics blocks
  - user profile analytics
  - channel/publisher profile analytics
  - admin data management
  - final smoke and docs
- Post-MVP Feature 1 backend operations foundation is implemented:
  - `worker_heartbeats` persists listener freshness
  - listener records a `listener` heartbeat every `LISTENER_HEARTBEAT_INTERVAL_SECONDS`
  - admin-only `GET /admin/operations/summary` returns listener freshness, core row counts,
    raw event status counts, PostgreSQL database/table sizes, and key ingest timestamps
  - backend verification passed with `python -m uv run pytest` reporting 101 tests
  - `python -m uv run ruff check .` passed
- Post-MVP Feature 1 admin operations UI is implemented:
  - `/admin` mounts `OperationsDashboard` above channel/user management
  - operations cards show listener status, database size, messages, raw events, failed raw,
    pending raw, and last ingest time
  - manual refresh and calm stale-listener/failed-raw/API-error states are covered by tests
  - frontend verification passed with `pnpm --filter @kick-logs/web test`, `typecheck`, and
    `lint`
- Post-MVP Feature 1 admin operations dashboard is complete:
  - README/admin usage notes are updated
  - `docs/tasks/post_mvp_01_admin_operations.md` has all checkboxes closed
  - final touched-area verification passed for backend tests/lint, frontend tests/typecheck/lint,
    and `pnpm format:check`
- Post-MVP Feature 3 analytics backend foundation is implemented:
  - public read-only `GET /analytics/overview`
  - public read-only `GET /analytics/message-volume`
  - public read-only `GET /analytics/top-senders`
  - public read-only `GET /analytics/top-channels`
  - public read-only `GET /analytics/top-emotes`
  - analytics repository aggregate queries run over `chat_messages`
  - supported filters include date range plus exact channel/sender scope
  - frontend typed analytics API wrappers exist under `apps/web/src/features/analytics/api.ts`
- Post-MVP Feature 4 landing page is implemented:
  - root `/` now renders a public compact landing page instead of redirecting
  - landing fetches analytics overview, recent day-bucket message volume, top channels, top
    emotes, and top senders
  - landing includes loading, error, and fresh-install empty states
  - landing links to `/search`, `/admin`, GitHub, and Buy Me a Coffee support
  - `/search` and `/admin` header brand/logo areas navigate back to `/`
  - frontend tests cover analytics rendering, empty data, and navigation links
  - verification passed with `pnpm --filter @kick-logs/web test`, `typecheck`, `lint`, `build`,
    `pnpm format:check`, Docker web rebuild/start, and route smoke for `/` plus `/search`
- Post-MVP Feature 5 user profile analytics is implemented:
  - public `GET /users/{slug}/analytics` returns sender identity/profile image, overview totals,
    day-bucket message volume, top channels, top emotes, and latest messages
  - unknown sender slugs return 404
  - public `/users/[slug]` renders the user profile, loading/error/not-found states, top
    channels, top emotes, volume bars, and latest messages
  - `/search` sender names and avatars link to `/users/[slug]`
  - profile pages link back to `/search?sender={slug}`
  - verification passed with `python -m uv run pytest`, backend ruff checks, web
    test/typecheck/lint/build, and `pnpm format:check`
- Post-MVP Feature 2 planning now includes clickable message links:
  - URLs inside `/search` message content should render as safe clickable anchors
  - link rendering must preserve inline emote placement and matched-text highlighting
  - `docs/tasks/post_mvp_02_search_improvements.md` includes explicit link rendering tests
- Issue #1 durable ingestion implementation is complete locally on branch `feature/issue-1-durable-inbox`.
- Issue #3 Kick reply rendering is implemented locally on branch `feat/issue-3-kick-reply-rendering`.
- Kick listener now uses a durable raw event inbox design:
  - websocket reader persists supported chat events into PostgreSQL first
  - raw event workers normalize and insert chat messages out of the websocket read path
  - stale `processing` events can be reclaimed after a timeout
  - listener periodically reconnects to refresh enabled channel subscriptions
- Root `/` is the public landing page; `/search` remains the primary search workflow.
- Phase 9 auth foundation is implemented:
  - `/login` has an email/password form wired to `POST /auth/login`
  - `/admin` uses `GET /auth/me` for client-side route guarding
  - unauthenticated `/admin` users are redirected to `/login?next=/admin`
  - admin header includes logout through `POST /auth/logout`
- Phase 9 followed-channel management UI is implemented:
  - `/admin` lists followed channels with enabled state, Kick channel id, and Kick chatroom id
  - admins can add a channel by slug/nickname through `POST /admin/channels`
  - add flow shows resolver/loading/error state while backend resolves Kick metadata
  - admins can disable channels through `DELETE /admin/channels/{id}`
- Phase 9 super-admin user management UI is implemented:
  - only `super_admin` users see the admin user management section
  - user list calls `GET /admin/users`
  - create form calls `POST /admin/users`
  - password hashes or secrets are never rendered
- Post-MVP admin operations UI is implemented:
  - `OperationsDashboard` calls `GET /admin/operations/summary`
  - stale listener heartbeat and failed raw event counts render compact warning states
  - manual refresh re-fetches the operations summary without mixing with channel/user controls
- `apps/api` contains the FastAPI skeleton with `GET /health`, settings/logging modules, clean architecture folders, tests, and `uv.lock`.
- Root `compose.yaml` currently has Phase 7 services:
  - `postgres`
  - `api`
  - `listener`
  - `web`
- Active implementation plan exists at `docs/implementation_plan.md`.
- Active task files exist under `docs/tasks/post_mvp_01_admin_operations.md` through `docs/tasks/post_mvp_08_final_smoke.md`.
- Archived MVP task files exist under `docs/archive/tasks/phase1_tasks.md` through `docs/archive/tasks/phase10_tasks.md`.
- Frontend `web` service exists and runs the Next.js development server.
- Current backend verification:
  - `python -m uv run pytest` passes from `apps/api` with 94 tests.
  - `python -m uv run ruff check .` passes from `apps/api`.
  - `OPTIONS http://localhost:8000/auth/login` from origin `http://localhost:3000` returns the expected CORS headers.
  - `POST http://localhost:8000/auth/login` from origin `http://localhost:3000` returns 200 and sets `kick_logs_session`.
  - `docker compose config --services` returns `postgres`, `api`, and `listener`.
  - `docker compose up --build -d postgres api listener` starts the backend stack successfully.
  - `GET http://localhost:8000/health` returns `{"status":"ok"}`.
  - `GET http://localhost:8000/messages?limit=1` works without login.
  - `GET http://localhost:8000/admin/channels` returns 401 without login.
  - Default super admin login and `GET /auth/me` pass.
  - Admin channel add/disable smoke passes with slug `hype`.
  - Listener logs useful idle status when no enabled channels are ready.
  - `alembic current` reports `20260511_0002 (head)` after the durable inbox migration is applied.
  - `python -m uv run pytest` passes from `apps/api` with 113 tests.
  - `pnpm --filter @kick-logs/web test` passes with 13 files and 53 tests.
  - `pnpm --filter @kick-logs/web typecheck`, `lint`, and `build` pass.
  - `docker compose up --build -d web` builds and starts `web`.
  - `GET http://localhost:3000` returns HTTP 200.
  - `GET http://localhost:3000/search` returns HTTP 200 without login.
- Phase 2 verification:
  - `alembic upgrade head` applies revision `20260510_0001`
  - `alembic current` reports `20260510_0001 (head)`
  - `uv run pytest` passes with 27 tests
  - `uv run ruff check .` passes
- Phase 3 verification:
  - `uv run pytest` passes with 45 tests
  - `uv run ruff check .` passes
  - `docker compose up --build -d postgres api` succeeds with auth dependencies
  - default super admin startup seed is available at `admin@kicklogs.local` / `admin123`
  - real API smoke for `POST /auth/login` and `GET /auth/me` passes
- Docker API startup runs `alembic upgrade head` before Uvicorn so startup seed has the required database schema.

## Kick Chat Ingestion Method

The MVP listener should implement this self-contained Kick web chat ingestion flow:

- Use `curl_cffi` with browser impersonation to resolve `https://kick.com/api/v2/channels/{slug}`.
- Read Kick `channel_id` from response `id`.
- Read Kick `chatroom_id` from response `chatroom.id`.
- Connect to:
  - `wss://ws-us2.pusher.com/app/32cbd69e4b950bf97679?protocol=7&client=js&version=8.4.0-rc2&flash=false`
- Subscribe to:
  - `chatrooms.{chatroom_id}.v2`
  - `channel.{channel_id}`
- Handle event:
  - `App\Events\ChatMessageEvent`
- Extract sender username from `payload.sender.username`.
- Extract message content from `payload.content`.

## Product Direction

Build an MVP monorepo with:

- Python backend
- Next.js frontend
- PostgreSQL persistence
- Docker Compose local runtime
- Admin channel management
- Searchable historical Kick chat logs

## Architecture Direction

- `docs/architecture.md` is the source of truth for backend/frontend structure.
- Backend uses pragmatic clean architecture with domain, application, infrastructure, and presentation layers.
- HTTP API and listener are separate Docker services but share one Python backend package.
- Backend uses OOP for use cases, services, repositories, integration clients, and unit-of-work boundaries.
- ORM decision is SQLAlchemy 2.x async ORM with asyncpg and Alembic.
- Domain entities stay independent from SQLAlchemy, FastAPI, Pydantic, and external clients.
- Frontend uses Next.js App Router with feature-oriented folders and shadcn/ui primitives in `components/ui`.

## Persistence Details

- Domain entities/value objects live under `apps/api/src/kick_logs/domain/` and stay framework-independent.
- Application repository/unit-of-work ports live under `apps/api/src/kick_logs/application/ports/`.
- SQLAlchemy infrastructure lives under `apps/api/src/kick_logs/infrastructure/database/`.
- Alembic migration revision `20260510_0001` creates:
  - `users`
  - `channels`
  - `senders`
  - `chat_messages`
- Alembic migration revision `20260511_0002` creates:
  - `raw_kick_events`
- Alembic migration revision `20260513_0003` creates:
  - `worker_heartbeats`
- PostgreSQL extension:
  - `pg_trgm`
- JSONB fields:
  - `channels.raw_payload`
  - `senders.raw_profile_payload`
  - `chat_messages.sender_badges`
  - `chat_messages.emotes`
  - `chat_messages.reply_metadata`
  - `chat_messages.raw_payload`
  - `raw_kick_events.payload`
  - `raw_kick_events.metadata`
- Dedupe/identity constraints:
  - `users.email`
  - `channels.kick_channel_id`
  - `channels.kick_chatroom_id`
  - `channels.slug`
  - `senders.kick_user_id`
  - `senders.slug`
  - `chat_messages.kick_message_id`
  - partial unique index on `raw_kick_events.kick_message_id` when present
- Search/index strategy:
  - btree indexes for message timestamp, cursor tuple support, channel id, sender id, chatroom id, sender username/slug, and channel slug
  - trigram GIN indexes for lowercased channel slug/display name, sender username/slug, and message content
- Repository implementations:
  - `SqlAlchemyUserRepository`
  - `SqlAlchemyChannelRepository`
  - `SqlAlchemySenderRepository`
  - `SqlAlchemyMessageRepository`
  - `SqlAlchemyRawEventRepository`
  - `SqlAlchemyOperationsRepository`
  - `SqlAlchemyWorkerHeartbeatRepository`
  - `SqlAlchemyUnitOfWork`

## Auth Details

- Auth uses Passlib bcrypt password hashing through `PasslibPasswordHasher`.
- Auth uses signed JWTs through `JwtTokenService`.
- Session tokens are stored in an HttpOnly cookie named by `JWT_COOKIE_NAME`.
- FastAPI CORS middleware is enabled from comma-separated `BACKEND_CORS_ORIGINS` so browser clients such as `http://localhost:3000` can call cookie-backed auth routes.
- Cookie settings:
  - `JWT_COOKIE_NAME`
  - `JWT_COOKIE_SECURE`
  - `JWT_COOKIE_SAMESITE`
  - `JWT_EXPIRES_MINUTES`
- Default super admin seed runs at API startup when `SEED_SUPER_ADMIN_ON_STARTUP=true`.
- Default super admin env:
  - `DEFAULT_SUPER_ADMIN_EMAIL`
  - `DEFAULT_SUPER_ADMIN_PASSWORD`
- Seed is idempotent:
  - creates the user when missing
  - promotes/reactivates an existing default user if needed
  - does not reset an existing password
- Auth/admin routes implemented:
  - `POST /auth/login`
  - `POST /auth/logout`
  - `GET /auth/me`
  - `GET /admin/users`
  - `POST /admin/users`
- `GET /admin/users` requires an authenticated admin or super admin.
- `POST /admin/users` requires `super_admin`.
- Public routes such as `GET /health` remain unauthenticated.

## Channel Management Details

- Kick channel resolution uses `curl_cffi` browser impersonation against `https://kick.com/api/v2/channels/{slug}`.
- Resolved channel metadata includes:
  - Kick channel id
  - Kick chatroom id
  - normalized slug
  - display name
  - profile image URL when available
  - banner image URL when available
  - raw Kick payload
- Admin channel routes implemented:
  - `GET /admin/channels`
  - `POST /admin/channels`
  - `DELETE /admin/channels/{id}`
- `POST /admin/channels` resolves Kick metadata before persistence.
- Re-adding an existing disabled channel re-enables it and refreshes stored metadata.
- `DELETE /admin/channels/{id}` disables the channel for the MVP instead of hard-deleting it.
- Channel resolver and admin channel route tests are covered under `apps/api/tests/channels/`.

## Message Ingestion Details

- Issue #1 durable inbox flow:
  - `StoreRawKickEventUseCase` writes supported raw Kick chat events to `raw_kick_events`.
  - `ProcessRawKickEventsUseCase` claims pending/stale raw events in batches.
  - SQLAlchemy claims use row locking with `FOR UPDATE SKIP LOCKED`.
  - raw events move through `pending`, `processing`, `processed`, and `failed` statuses.
  - failed raw events retain payload, attempts, and last error.
  - `processing` rows older than the configured timeout are claimable again.
  - `IngestMessageUseCase` remains idempotent by Kick message id, so duplicate raw processing does not duplicate `chat_messages`.
- Emote parsing is implemented by `EmoteParser`.
- Supported emote token format:
  - `[emote:id:name]`
- Parsed emote data stores:
  - `id`
  - `name`
  - original `token`
  - inferred `image_url`
- Message content remains unchanged after emote parsing.
- `IngestMessageUseCase` normalizes Kick chat event payloads into:
  - sender record
  - chat message record
  - sender snapshots
  - sender badges
  - reply metadata
  - thread parent id
  - raw message payload
- Ingestion deduplicates by Kick message id and returns the existing stored message on duplicate input.
- Ingestion resolves the followed channel by Kick chatroom id; unknown chatrooms fail with `ChannelNotFoundError`.

## Public Search API Details

- Public route implemented:
  - `GET /messages`
  - `GET /messages/export`
- Query parameters:
  - `sender`
  - `channel`
  - `q`
  - `start`
  - `end`
  - `reply_only`
  - `emote_only`
  - `cursor`
  - `limit`
- The route requires no authentication.
- Empty filters return latest messages across all channels.
- Non-empty filters combine with `AND`.
- Sender filters use case-insensitive exact matching against sender username/slug snapshots.
- Channel and content filters use case-insensitive contains matching.
- Date filters apply to `message_created_at`.
- `reply_only=true` filters to messages where `message_type` is `reply`.
- `emote_only=true` filters to messages with one or more parsed emotes.
- Results are newest-first.
- Cursor format is:
  - `{message_created_at.isoformat()}|{message_id}`
- Cursor pagination uses `(message_created_at, id)`.
- Search response includes:
  - message content and timestamps
  - sender snapshot fields
  - sender badges
  - reply metadata
  - thread parent id
  - parsed emotes with image URLs
  - sender profile fields including avatar URL
  - channel metadata including profile/banner URLs
- Search use case batches sender/channel lookup by id after message search to avoid one metadata query per row.
- Export response supports:
  - `format=json`
  - `format=csv`
  - the same filter semantics as `GET /messages`
  - no auth requirement
  - per-request `limit` clamped by `MESSAGE_EXPORT_MAX_ROWS`

## Public Analytics API Details

- Public routes implemented:
  - `GET /analytics/overview`
  - `GET /analytics/message-volume`
  - `GET /analytics/top-senders`
  - `GET /analytics/top-channels`
  - `GET /analytics/top-emotes`
- The routes require no authentication.
- Common optional query parameters:
  - `start`
  - `end`
  - `channel`
  - `sender`
- Date filters apply to `chat_messages.message_created_at`.
- Analytics sender scope uses case-insensitive exact matching against sender username/slug and
  stored message sender snapshots.
- Analytics channel scope uses case-insensitive exact matching against channel slug/display name.
- `message-volume` accepts `bucket=hour|day`.
- Top-list endpoints accept `limit` from 1 to 100.
- `overview` returns message, distinct sender, distinct channel, total emote usage, first message,
  and latest message metrics.
- `top-emotes` aggregates parsed `chat_messages.emotes` JSONB values by emote id/name/token/image.
- Frontend typed wrappers live under `apps/web/src/features/analytics/api.ts`.

## Post-MVP Feature 3 Verification

- `python -m uv run pytest`: 111 passed.
- `python -m uv run ruff check .`: passed.
- `python -m uv run ruff format --check .`: passed.
- `pnpm --filter @kick-logs/web test`: 11 files, 47 tests passed.
- `pnpm --filter @kick-logs/web typecheck`: passed.
- `pnpm --filter @kick-logs/web lint`: passed.
- `pnpm --filter @kick-logs/web build`: passed.
- `pnpm format:check`: passed.

## Phase 4 Verification

- `uv run pytest`: 65 tests passed.
- `uv run ruff check .`: all checks passed.
- `docker compose up --build -d postgres api`: backend stack builds and starts.
- `GET http://localhost:8000/health`: returns `{"status":"ok"}`.
- `GET http://localhost:8000/messages?limit=1`: returns public search response successfully.

## Listener Foundation Details

- `LoadEnabledChannelsUseCase` loads enabled channels for the worker.
- Channels missing Kick channel/chatroom metadata are resolved through the existing Kick channel resolver before subscription.
- Disabled channels are excluded by repository query.
- Channels still missing subscription metadata after resolution failure are skipped and returned in a skipped list for structured logging.
- `KickEventParser` parses Pusher envelopes and extracts only `App\Events\ChatMessageEvent` payloads.
- Malformed JSON, non-chat events, and incomplete chat payloads are ignored without raising.
- `ReconnectPolicy` computes exponential backoff delays with a maximum cap.
- `KickPusherClient` connects to the configured Kick Pusher websocket URL and subscribes to:
  - `chatrooms.{kick_chatroom_id}.v2`
  - `channel.{kick_channel_id}`
- Pusher subscription payload includes an empty `auth` field and channel name:
  - `{"event":"pusher:subscribe","data":{"auth":"","channel":"chatrooms.{id}.v2"}}`
- Websocket runtime uses 30 second ping interval and 10 second ping timeout.
- Kick web HTTP resolvers use `curl_cffi` browser impersonation `chrome124`.
- `ListenerService` composes enabled-channel loading, Pusher event streaming, event parsing, raw event persistence, and raw event worker processing.
- `ListenerService.run_forever()` reconnects with backoff and reloads enabled channels on each reconnect.
- `ListenerService.run_forever()` starts raw event worker tasks before connecting to Pusher.
- `ListenerService.run_forever()` starts a heartbeat task that upserts `worker_heartbeats`
  for service `listener`.
- `ListenerService.run_once()` persists parsed chat events into `raw_kick_events` before message normalization or sender upsert work begins.
- Raw event workers call `ProcessRawKickEventsUseCase` in batches and log claimed/processed/failed/pending counts.
- Admin `GET /admin/operations/summary` exposes storage growth, raw backlog/status, latest
  ingest timestamps, and listener heartbeat freshness without requiring Docker logs.
- The websocket loop reconnects after `LISTENER_CHANNEL_RESYNC_INTERVAL_SECONDS` so followed-channel add/remove changes take effect without manual restart.
- Sender profile enrichment is no longer on the websocket read path; profile images are stored when present in the raw message payload.
- Worker entrypoint is `kick_logs.presentation.worker.main`.

## Phase 5 Verification

- `uv run pytest`: 83 tests passed.
- `uv run ruff check .`: all checks passed.
- `docker compose config --services`: returns `postgres`, `api`, `listener`.
- `docker compose up --build -d postgres api listener`: backend stack builds and starts.
- `docker compose ps`: `postgres`, `api`, and `listener` are up.
- `GET http://localhost:8000/health`: returns `{"status":"ok"}`.
- Listener logs show idle no-channel checks without crashing when no channels are enabled.

## Phase 6 Verification

- `python -m uv run pytest`: 83 tests passed.
- `python -m uv run ruff check .`: all checks passed.
- `python -m uv run alembic current`: reports `20260510_0001 (head)`.
- `docker compose up --build -d postgres api listener`: backend stack builds and starts.
- `docker compose ps`: `postgres`, `api`, and `listener` are up.
- API logs show Alembic migration startup, default super admin seed, and clean request handling.
- Listener logs show Alembic migration startup and useful idle status when no enabled channels are ready.
- Public `GET /messages?limit=1` works without authentication.
- Admin `GET /admin/channels` returns 401 without authentication.
- Default super admin can login and call `GET /auth/me`.
- Admin channel add/disable smoke passes with slug `hype`.
- No `.env`, virtualenv, cache, log, or dependency directory is tracked.
- No repository files reference the local-only reference project.
- Runtime warning cleanup:
  - default local/Compose `JWT_SECRET_KEY` is now at least 32 bytes
  - `bcrypt` is pinned to `>=4.0.1,<4.1` for Passlib compatibility

## Frontend Foundation Details

- Root frontend workspace files:
  - `package.json`
  - `pnpm-workspace.yaml`
  - `pnpm-lock.yaml`
- `apps/web` uses:
  - Next.js App Router
  - TypeScript
  - Tailwind CSS
  - shadcn/ui base configuration
  - lucide-react dependency
- Root scripts proxy to the web workspace:
  - `pnpm dev`
  - `pnpm build`
  - `pnpm lint`
  - `pnpm typecheck`
- Web workspace scripts:
  - `pnpm --filter @kick-logs/web dev`
  - `pnpm --filter @kick-logs/web build`
  - `pnpm --filter @kick-logs/web lint`
  - `pnpm --filter @kick-logs/web typecheck`
- Placeholder routes exist:
  - `/`
  - `/search`
  - `/login`
  - `/admin`
- No final `/search` UI or `/admin` workflow has been implemented yet.
- Dark-only palette tokens are defined in `apps/web/src/app/globals.css` and exposed through Tailwind as `kick.*` tokens.
- shadcn/ui base files:
  - `apps/web/components.json`
  - `apps/web/src/lib/utils.ts`
  - `apps/web/src/components/ui/button.tsx`
- Shared frontend API layer:
  - `apps/web/src/lib/api-client.ts`
  - `apps/web/src/types/api.ts`
  - feature endpoint wrappers under `apps/web/src/features/*/api.ts`
- API client defaults:
  - base URL from `NEXT_PUBLIC_API_BASE_URL`
  - local fallback `http://localhost:8000`
  - `credentials: "include"` for cookie sessions
  - injectable fetcher/client for tests and mocks
- Web Docker service:
  - service name `web`
  - exposed at `http://localhost:3000`
  - reads `NEXT_PUBLIC_API_BASE_URL` from environment
  - does not require backend code changes

## Phase 7 Verification

- `pnpm install`: completed and generated `pnpm-lock.yaml`.
- `pnpm --filter @kick-logs/web typecheck`: passed.
- `pnpm --filter @kick-logs/web lint`: passed.
- `pnpm --filter @kick-logs/web build`: passed.
- `docker compose config --services`: returns `postgres`, `api`, `listener`, and `web`.
- `docker compose up --build -d web`: builds and starts `web` with API dependency.
- `docker compose ps`: `postgres`, `api`, `listener`, and `web` are up.
- `GET http://localhost:3000`: returns HTTP 200.
- `GET http://localhost:8000/health`: returns `{"status":"ok"}`.

## Public Search UI Details

- `/search` is implemented as a public client route with no auth gate.
- The implementation used `docs/design/design.pen` directly by reading the JSON frame:
  - `Search Screen / Desktop (User Friendly ReTouch Current)`
- The Pencil MCP app was unavailable during implementation, but the `.pen` file was readable as JSON and used for structure, labels, colors, spacing, and result-row behavior.
- `/search` does not show admin-only controls or admin placeholder content.
- Search form fields:
  - `Kullanıcı Adı` -> `sender`
  - `Kanal Adı` -> `channel`
  - `Aramak istediğiniz Kelime` -> `q`
  - `Başlangıç` -> `start`
  - `Bitiş` -> `end`
  - `Sadece yanıtlar` -> `reply_only`
  - `Sadece emote` -> `emote_only`
- Empty form fields are omitted from URL/backend query params.
- Opening bare `/search` does not automatically call the backend.
- Before the user submits a search, the result area shows `Arama yapmak için yukarıdaki formu kullanın.`
- Explicitly submitting empty filters still fetches latest messages.
- Missing date filters default in the `/search` UI to the last 7 days:
  - `Başlangıç` is current local date/time minus 7 days.
  - `Bitiş` is current local date/time.
  - users can clear date fields to omit date filters.
- A compact `Hızlı aralık` select sets the range to last 1 hour, 24 hours, 7 days, or 30
  days without clearing other filters.
- Date fields and `Hızlı aralık` occupy their own row; `Sadece yanıtlar` and `Sadece emote`
  sit below them on the left side of the action row, before the `İşlem` buttons.
- The `/search` initial SSR render uses an empty static search state; the default local date range is filled after hydration to avoid server/client timezone mismatches.
- Submitted filter state is preserved in the URL query string.
- A square `Dışa aktar` icon button opens compact `JSON indir` and `CSV indir` actions for
  the last submitted filter state.
- The export menu closes when the user chooses a format or clicks outside the menu.
- Result fetching uses public `GET /messages` through the shared frontend API client.
- Cursor pagination is wired to an IntersectionObserver sentinel for infinite scroll.
- Result rows render inside one shared list container with:
  - circular sender avatar or circular fallback initial
  - sender
  - channel
  - message content
  - timestamp
- Rows are dense and table-like on desktop, then collapse to a readable mobile layout.
- Inline emote rendering:
  - replaces `[emote:id:name]` tokens at their original message positions
  - prefers backend parsed emote image URL
  - falls back to `https://files.kick.com/emotes/{id}/fullsize`
  - falls back to emote text if the image fails
- Message content rendering:
  - URLs render as safe new-tab links with `rel="noopener noreferrer"`
  - matched `q` text renders as a compact inline highlight
  - link rendering and highlighting are applied only to text parts, so inline emote placement is
    preserved
- Reply rendering:
  - messages with `message_type === "reply"` render replied-to context above current content
  - reply preview reads `reply_metadata.original_sender.username`
  - reply preview reads `reply_metadata.original_message.content`
  - long reply previews use a `title` attribute for full-content hover inspection
- Search summary panel shows loading/error status, loaded count, scope, last match, and active filters.
- The app logo has been copied into `apps/web/public/app-logo.png` for the search header.
- Frontend tests added with Vitest and React Testing Library:
  - query mapping
  - empty filter behavior
  - active filter labels
  - infinite-scroll append dedupe helper
  - inline emote split/fallback rendering
  - date preset helpers
  - reply-only and emote-only URL/query mapping
  - clickable link rendering
  - matched-text highlighting with emote compatibility
  - export button URL behavior

## Post-MVP Feature 2 Verification

- Latest search form density polish:
  - `pnpm --filter @kick-logs/web test`: 10 files, 44 tests passed.
  - `pnpm --filter @kick-logs/web typecheck`: passed.
  - `pnpm --filter @kick-logs/web lint`: passed.
  - `pnpm --filter @kick-logs/web build`: passed.
  - `pnpm format:check`: passed.
- `pnpm --filter @kick-logs/web test`: 10 files, 42 tests passed.
- `pnpm --filter @kick-logs/web typecheck`: passed.
- `pnpm --filter @kick-logs/web lint`: passed.
- `pnpm --filter @kick-logs/web build`: passed.
- `pnpm format:check`: passed.
- `python -m uv run ruff check .`: passed.
- `python -m uv run pytest tests/domain/test_value_objects.py tests/test_config.py tests/messages/test_http_search_messages.py`: 18 passed.

## Phase 8 Verification

- `pnpm --filter @kick-logs/web test`: 2 files, 7 tests passed.
- `pnpm --filter @kick-logs/web typecheck`: passed.
- `pnpm --filter @kick-logs/web lint`: passed.
- `pnpm --filter @kick-logs/web build`: passed.
- `docker compose up --build -d web`: builds and starts `web` with current lockfile.
- `GET http://localhost:3000/search`: returns HTTP 200 without login.
- `GET http://localhost:3000/search?sender=yavuz&q=selam`: returns the search page and does not include admin placeholder content.
- `GET http://localhost:8000/health`: returns `{"status":"ok"}` after Docker rebuild startup.

## Phase 9 Auth UI Details

- Login route:
  - `/login`
  - pre-fills the local MVP super admin email `admin@kicklogs.local`
  - submits credentials to `POST /auth/login`
  - relies on the backend HttpOnly cookie for session persistence
  - redirects to `/admin` or a safe local `next` path after successful login
  - shows compact credential/API errors
- Admin guard:
  - `/admin` is a client route guarded by `GET /auth/me`
  - unauthenticated users are redirected to `/login?next=/admin`
  - authenticated users see a compact admin shell and logout action
- Verification:
  - `pnpm --filter @kick-logs/web test`: 4 files, 14 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed

## Phase 9 Channel Admin Details

- `ChannelAdmin` is mounted inside `/admin` for authenticated users.
- Channel list behavior:
  - calls `GET /admin/channels`
  - shows display name, slug, Kick channel id, Kick chatroom id, and enabled state
  - provides a manual refresh action
- Channel add behavior:
  - accepts a Kick slug/nickname
  - calls `POST /admin/channels`
  - relies on the backend resolver for Kick metadata
  - merges the created/refreshed channel into the list
- Channel disable behavior:
  - calls `DELETE /admin/channels/{id}`
  - merges the returned disabled channel into the list
- Verification:
  - `pnpm --filter @kick-logs/web test`: 5 files, 17 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed

## Phase 9 User Admin Details

- `UserAdmin` is mounted inside `/admin` only when the current user role is `super_admin`.
- User list behavior:
  - calls `GET /admin/users`
  - shows email, role, and active state only
  - does not render password hashes or secrets
- User creation behavior:
  - accepts email and temporary password
  - requires at least 8 password characters before enabling submit
  - calls `POST /admin/users`
  - merges the created user into the list
- Verification:
  - `pnpm --filter @kick-logs/web test`: 6 files, 20 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed

## Phase 9 Verification

- `pnpm --filter @kick-logs/web test`: 6 files, 20 tests passed.
- `pnpm --filter @kick-logs/web typecheck`: passed after build/type generation completed.
- `pnpm --filter @kick-logs/web lint`: passed.
- `pnpm --filter @kick-logs/web build`: passed.
- `docker compose up --build -d web`: passed and rebuilt the web/API images.
- `docker compose ps`: `postgres`, `api`, `listener`, and `web` are up.
- `GET http://localhost:3000/search`: HTTP 200 without login.
- `GET http://localhost:3000/login`: HTTP 200.
- `GET http://localhost:3000/admin`: HTTP 200; client guard handles unauthenticated redirect.
- `GET http://localhost:8000/health`: returns `{"status":"ok"}`.

## Phase 10 Final Smoke

- Backend checks:
  - `python -m uv run pytest`: 83 tests passed.
  - `python -m uv run ruff check .`: passed.
- Frontend checks:
  - `pnpm --filter @kick-logs/web test`: 6 files, 20 tests passed.
  - `pnpm --filter @kick-logs/web typecheck`: passed.
  - `pnpm --filter @kick-logs/web lint`: passed.
  - `pnpm --filter @kick-logs/web build`: passed.
- Full Docker stack:
  - `docker compose up --build -d`: starts `postgres`, `api`, `listener`, and `web`.
  - `docker compose ps`: all four services are up.
  - `GET http://localhost:8000/health`: returns `{"status":"ok"}`.
  - `GET http://localhost:3000/`: HTTP 307 to `/search`.
  - `GET http://localhost:3000/search`: HTTP 200 without login.
  - `GET http://localhost:3000/login`: HTTP 200.
  - `GET http://localhost:3000/admin`: HTTP 200; client guard owns unauthenticated redirect.
- Listener:
  - initially logs idle status when no channels are ready.
  - after enabling `hype`, logs `Subscribing to 1 enabled Kick channels.`
- End-to-end smoke:
  - default super admin login succeeds through `POST /auth/login`.
  - authenticated channel add for slug `hype` succeeds and stores Kick metadata:
    - `channel_slug`: `hype`
    - `kick_chatroom_id`: `24495088`
  - sample message ingested through `IngestMessageUseCase`:
    - marker: `phase10-smoke-20260510235338`
    - sender: `PhaseTenSmoke`
    - content includes `[emote:37226:KEKW]`
  - public `GET /messages?q=phase10-smoke-20260510235338&limit=5` finds the sample without authentication.
  - emote metadata in the search response includes `KEKW`.
  - restarting the `postgres` service preserves the sample message in the named volume.
- Cleanup:
  - no tracked `.env`, generated cache, dependency folder, log, or build output was found.
  - removed the unused `RouteShell` scaffold and kept the MVP root behavior search-first at that
    time.

## Issue #1 Durable Ingestion Verification

- Added `raw_kick_events` persistence with JSONB payload storage, status tracking, attempts, last error, received/processing/processed timestamps, and metadata.
- Added partial Kick message id dedupe index and processing/search indexes for raw events.
- Added raw event repository methods for add, lookup, `FOR UPDATE SKIP LOCKED` batch claim, processed mark, failed mark, pending count, and stale processing reclaim.
- Refactored listener websocket read path to store raw events before heavy normalization or sender/message writes.
- Added raw event worker loop inside the listener service so websocket reads and DB message processing proceed independently.
- Added periodic channel resync reconnect behavior through `LISTENER_CHANNEL_RESYNC_INTERVAL_SECONDS`.
- Verification so far:
  - `python -m uv run ruff check .`: passed.
  - `python -m uv run alembic upgrade head`: applied `20260511_0002`.
  - `python -m uv run alembic current`: reports `20260511_0002 (head)`.
  - `python -m uv run pytest`: 94 passed.
  - `python -m uv run pytest tests/listener tests/domain tests/database/test_models_metadata.py tests/database/test_alembic_migration.py`: 43 passed.
  - `python -m uv run pytest tests/database/test_repositories.py tests/messages/test_ingest_message.py tests/listener/test_listener_service.py`: 19 passed against local PostgreSQL.
  - `docker compose up --build -d postgres api listener`: passed.
  - `GET http://localhost:8000/health`: `{"status":"ok"}`.
  - listener logs show raw event storage and raw event worker processing with `pending=0`.

## Design Direction

- UI implementation is active and must stay aligned with the working backend API contracts.
- `docs/design/design.md` is the source of truth for future UI and UX decisions.
- Any future UI/frontend agent must read `docs/design/design.md` before changing frontend code.
- The UI is dark-only with palette `#26001B`, `#810034`, `#FF005C`, `#FFF600`, black, and white.
- Primary buttons should use `#FFF600`.
- Do not use blur, glow, colored lighting, or atmospheric background effects.
- The provided search screenshot is a layout/workflow reference only; do not copy its green visual style exactly.
- The user-provided logo should be used where a product mark is needed.
- The approved `/search` screen remains represented in `docs/design/design.pen`; later UI work
  should follow the same dark, compact product style.
- Search results should render as dense rows inside one shared outer container, not as separate modal/card components per message.
- Sender avatars should be circular, emotes should render inline at their message positions, and reply rows should show replied-to sender/content without adding per-message cards or modals.

## Locked Product Decisions

- Store messages indefinitely.
- Use a full login system with `super_admin` and `admin` roles.
- Seed default super admin:
  - email: `admin@kicklogs.local`
  - password: `admin123`
- Allow env override for default super admin credentials.
- Use `/` for the public landing page, `/search` for the public app search screen, and
  `/admin` for authenticated backend management.
- `/search` does not require login.
- `/admin` manages backend operational state such as followed channels and admin users.
- Search filters are optional and combined with `AND`:
  - sender nickname
  - channel nickname/slug
  - message content
  - start datetime
  - end datetime
- Use case-insensitive exact matching for sender username/slug search.
- Use case-insensitive contains matching for channel and message content.
- Use one listener worker/container to subscribe to all enabled channels.
- Store all useful available data, including normalized fields, parsed emotes, sender badges, profile image when enriched, reply metadata, and raw payload JSONB.
- Render emotes with `https://files.kick.com/emotes/{id}/fullsize` and fall back to the emote name/token if the image fails.

## Operational Rules

- Every agent must read `AGENTS.md` and context files before making changes.
- Every implementation agent must read `docs/implementation_plan.md` and the matching active task file before changing files.
- Active post-MVP task files are scoped handoff contracts; do not implement work from another feature unless the user explicitly changes the plan.
- Archived MVP task files are historical context only.
- Keep documentation and context current with implementation changes.
- Update `docs/context/recent_changes.md` with a short latest-change handoff after each meaningful change.
- Multi-agent work is allowed for non-overlapping scopes; assign clear file/subsystem ownership and integrate outputs before committing.
- Commit after each completed unit of work when requested.
- User will manually push commits.
