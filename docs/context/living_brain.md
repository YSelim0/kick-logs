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
- `apps/api` contains the FastAPI skeleton with `GET /health`, settings/logging modules, clean architecture folders, tests, and `uv.lock`.
- Root `compose.yaml` currently has backend Phase 5 services:
  - `postgres`
  - `api`
  - `listener`
- Sequential implementation plan exists at `docs/implementation_plan.md`.
- Phase task files exist under `docs/tasks/phase1_tasks.md` through `docs/tasks/phase10_tasks.md`.
- Frontend `web` service is still deferred until its owning frontend phase.
- Local verification:
  - `uv run pytest` passes from `apps/api`.
  - `uv run ruff check .` passes from `apps/api`.
  - `docker compose config --services` returns only `postgres` and `api`.
  - `docker compose up --build -d postgres api` starts the Phase 1 stack successfully.
  - `GET http://localhost:8000/health` returns `{"status":"ok"}`.
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
- PostgreSQL extension:
  - `pg_trgm`
- JSONB fields:
  - `channels.raw_payload`
  - `senders.raw_profile_payload`
  - `chat_messages.sender_badges`
  - `chat_messages.emotes`
  - `chat_messages.reply_metadata`
  - `chat_messages.raw_payload`
- Dedupe/identity constraints:
  - `users.email`
  - `channels.kick_channel_id`
  - `channels.kick_chatroom_id`
  - `channels.slug`
  - `senders.kick_user_id`
  - `senders.slug`
  - `chat_messages.kick_message_id`
- Search/index strategy:
  - btree indexes for message timestamp, cursor tuple support, channel id, sender id, chatroom id, sender username/slug, and channel slug
  - trigram GIN indexes for lowercased channel slug/display name, sender username/slug, and message content
- Repository implementations:
  - `SqlAlchemyUserRepository`
  - `SqlAlchemyChannelRepository`
  - `SqlAlchemySenderRepository`
  - `SqlAlchemyMessageRepository`
  - `SqlAlchemyUnitOfWork`

## Auth Details

- Auth uses Passlib bcrypt password hashing through `PasslibPasswordHasher`.
- Auth uses signed JWTs through `JwtTokenService`.
- Session tokens are stored in an HttpOnly cookie named by `JWT_COOKIE_NAME`.
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
- Query parameters:
  - `sender`
  - `channel`
  - `q`
  - `start`
  - `end`
  - `cursor`
  - `limit`
- The route requires no authentication.
- Empty filters return latest messages across all channels.
- Non-empty filters combine with `AND`.
- Sender, channel, and content filters use case-insensitive contains matching.
- Date filters apply to `message_created_at`.
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
- `ListenerService` composes enabled-channel loading, Pusher event streaming, event parsing, sender profile enrichment, and `IngestMessageUseCase`.
- `ListenerService.run_forever()` reconnects with backoff and reloads enabled channels on each reconnect.
- Sender profile enrichment uses Kick web channel metadata by sender slug and continues ingestion when enrichment fails.
- Worker entrypoint is `kick_logs.presentation.worker.main`.

## Phase 5 Verification

- `uv run pytest`: 83 tests passed.
- `uv run ruff check .`: all checks passed.
- `docker compose config --services`: returns `postgres`, `api`, `listener`.
- `docker compose up --build -d postgres api listener`: backend stack builds and starts.
- `docker compose ps`: `postgres`, `api`, and `listener` are up.
- `GET http://localhost:8000/health`: returns `{"status":"ok"}`.
- Listener logs show idle no-channel checks without crashing when no channels are enabled.

## Design Direction

- UI implementation is deferred until the backend API and listener are working end-to-end.
- `docs/design/design.md` is the source of truth for future UI and UX decisions.
- Any future UI/frontend agent must read `docs/design/design.md` before changing frontend code.
- The UI is dark-only with palette `#26001B`, `#810034`, `#FF005C`, `#FFF600`, black, and white.
- Primary buttons should use `#FFF600`.
- Do not use blur, glow, colored lighting, or atmospheric background effects.
- The provided search screenshot is a layout/workflow reference only; do not copy its green visual style exactly.
- The user-provided logo should be used where a product mark is needed.
- The search screen is the first design target in `docs/design/design.pen`; admin panel screens should not be designed until the search screen is approved.
- Search results should render as dense rows inside one shared outer container, not as separate modal/card components per message.
- Sender avatars should be circular, and emotes should render inline at their message positions.

## Locked Product Decisions

- Store messages indefinitely.
- Use a full login system with `super_admin` and `admin` roles.
- Seed default super admin:
  - email: `admin@kicklogs.local`
  - password: `admin123`
- Allow env override for default super admin credentials.
- Use `/search` for the public app search screen, `/admin` for authenticated backend management, and reserve `/` for a future landing page.
- `/search` does not require login.
- `/admin` manages backend operational state such as followed channels and admin users.
- Search filters are optional and combined with `AND`:
  - sender nickname
  - channel nickname/slug
  - message content
  - start datetime
  - end datetime
- Use case-insensitive contains matching for sender, channel, and message content.
- Use one listener worker/container to subscribe to all enabled channels.
- Store all useful available data, including normalized fields, parsed emotes, sender badges, profile image when enriched, reply metadata, and raw payload JSONB.
- Render emotes with `https://files.kick.com/emotes/{id}/fullsize` and fall back to the emote name/token if the image fails.

## Operational Rules

- Every agent must read `AGENTS.md` and context files before making changes.
- Every implementation agent must read `docs/implementation_plan.md` and the matching phase task file before changing files.
- Phase task files are scoped handoff contracts; do not implement work from a later phase unless the user explicitly changes the plan.
- Keep documentation and context current with implementation changes.
- Update `docs/context/recent_changes.md` with a short latest-change handoff after each meaningful change.
- Multi-agent work is allowed for non-overlapping scopes; assign clear file/subsystem ownership and integrate outputs before committing.
- Commit after each completed unit of work when requested.
- User will manually push commits.
