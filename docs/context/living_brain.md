# Living Brain

This file is the active project memory. Keep it updated whenever project behavior, architecture, implementation details, or working assumptions change.

## Current State

- Repository `kick-logs` has been initialized locally.
- Commit convention skill exists under `.agents/skills/commit-message-conventions`.
- Phase 1 backend/Docker foundation is complete.
- Phase 2 persistence foundation is complete.
- `apps/api` contains the FastAPI skeleton with `GET /health`, settings/logging modules, clean architecture folders, tests, and `uv.lock`.
- Root `compose.yaml` has only the Phase 1 services:
  - `postgres`
  - `api`
- Sequential implementation plan exists at `docs/implementation_plan.md`.
- Phase task files exist under `docs/tasks/phase1_tasks.md` through `docs/tasks/phase10_tasks.md`.
- Phase 1 Compose scope is only `postgres` and `api`; do not add placeholder `web` or `listener` services early.
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
