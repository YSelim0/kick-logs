# Kick Logs Architecture

## Overview

Kick Logs uses a pragmatic clean architecture backend and a feature-oriented Next.js frontend.

The backend is one Python codebase with two runtime entrypoints:

- HTTP API service
- Kick listener worker service

Both entrypoints share the same domain, application use cases, persistence adapters, and Kick integration clients. Docker Compose runs them as separate services with different commands.

## Monorepo Layout

```text
kick-logs/
  apps/
    api/
      pyproject.toml
      uv.lock
      alembic.ini
      alembic/
      src/
        kick_logs/
          main.py
          core/
          domain/
          application/
          infrastructure/
          presentation/
      tests/
    web/
      package.json
      src/
        app/
        components/
        features/
        lib/
        types/
  docs/
  compose.yaml
  README.md
```

There is no separate Python package for the listener in MVP. The `listener` Docker service runs the worker entrypoint from `apps/api`.

## Backend Principles

- Keep domain logic independent from FastAPI, SQLAlchemy, Pydantic, and external APIs.
- Use OOP for use cases, services, repositories, integration clients, and unit-of-work boundaries.
- Use dependency inversion: application code depends on interfaces, infrastructure implements them.
- Keep transaction control at application/use-case boundaries through a unit-of-work abstraction.
- Keep SQLAlchemy ORM models in infrastructure only.
- Keep Pydantic models in presentation/API schemas only.
- Prefer small classes with explicit responsibilities over generic service bags.

## Backend Folder Structure

```text
apps/api/src/kick_logs/
  main.py
  core/
    config.py
    errors.py
    logging.py
    security.py
  domain/
    entities/
      user.py
      channel.py
      chat_message.py
      raw_kick_event.py
      sender.py
      emote.py
      worker_heartbeat.py
    value_objects/
      roles.py
      search_filters.py
      pagination.py
      raw_event_status.py
    exceptions.py
  application/
    ports/
      unit_of_work.py
      user_repository.py
      channel_repository.py
      message_repository.py
      raw_event_repository.py
      sender_repository.py
      operations_repository.py
      worker_heartbeat_repository.py
      kick_channel_resolver.py
      sender_profile_resolver.py
      password_hasher.py
      token_service.py
    use_cases/
      auth/
        login.py
        get_current_user.py
      users/
        create_admin_user.py
        list_admin_users.py
      channels/
        add_channel.py
        remove_channel.py
        list_channels.py
      messages/
        search_messages.py
        ingest_message.py
      listener/
        load_enabled_channels.py
        store_raw_event.py
        process_raw_events.py
        record_worker_heartbeat.py
      operations/
        get_operations_summary.py
    dto/
      auth.py
      channels.py
      messages.py
      users.py
      operations.py
  infrastructure/
    database/
      session.py
      unit_of_work.py
      models.py
      repositories/
        sqlalchemy_user_repository.py
        sqlalchemy_channel_repository.py
        sqlalchemy_message_repository.py
        sqlalchemy_raw_event_repository.py
        sqlalchemy_sender_repository.py
        sqlalchemy_operations_repository.py
        sqlalchemy_worker_heartbeat_repository.py
    kick/
      channel_resolver.py
      sender_profile_resolver.py
      pusher_client.py
      event_parser.py
      reconnect_policy.py
    auth/
      jwt_token_service.py
      passlib_password_hasher.py
    seed/
      super_admin.py
  presentation/
    http/
      app.py
      dependencies.py
      routes/
        auth.py
        health.py
        messages.py
        admin_channels.py
        admin_users.py
        admin_operations.py
      schemas/
        auth.py
        channels.py
        messages.py
        users.py
        operations.py
    worker/
      main.py
      listener_service.py
```

## Backend Dependency Direction

```text
presentation -> application -> domain
infrastructure -> application/domain
```

Allowed dependencies:

- `domain` imports only Python standard library and local domain modules.
- `application` imports domain and application ports/DTOs.
- `infrastructure` imports application ports and domain entities.
- `presentation` imports application use cases and presentation schemas.

Disallowed dependencies:

- Domain importing SQLAlchemy, FastAPI, Pydantic, websocket clients, or HTTP clients.
- Application importing FastAPI route objects or SQLAlchemy ORM models.
- Presentation directly writing database queries.

## ORM And Database

Use:

- SQLAlchemy 2.x async ORM
- asyncpg PostgreSQL driver
- Alembic migrations

Rationale:

- Mature PostgreSQL support.
- First-class async support for FastAPI through `create_async_engine`, `AsyncSession`, and `async_sessionmaker`.
- Alembic is the standard migration tool for SQLAlchemy.
- Typed SQLAlchemy 2.x models work well while keeping domain entities independent.

Database policy:

- Use PostgreSQL `timestamptz` for message and audit timestamps.
- Store Kick raw payloads as `JSONB`.
- Add `pg_trgm` extension for case-insensitive contains search.
- Add indexes for message timestamp, Kick message id, channel slug, sender username, and content search.
- Deduplicate chat messages by Kick message id.
- Persist raw Kick chat events into `raw_kick_events` before normalization or enrichment so a received websocket event survives worker restarts.
- Process raw events with at-least-once delivery and idempotent message writes.
- Persist listener freshness in `worker_heartbeats` so admin screens can tell whether ingestion
  is alive even when followed channels are quiet.

## Core Tables

```text
users
  id
  email
  password_hash
  role
  is_active
  created_at
  updated_at

channels
  id
  kick_channel_id
  kick_chatroom_id
  slug
  display_name
  profile_image_url
  banner_image_url
  is_enabled
  raw_payload
  created_at
  updated_at

senders
  id
  kick_user_id
  username
  slug
  profile_image_url
  last_seen_color
  raw_profile_payload
  created_at
  updated_at

chat_messages
  id
  kick_message_id
  channel_id
  sender_id
  chatroom_id
  content
  message_type
  sender_username_snapshot
  sender_slug_snapshot
  sender_color_snapshot
  sender_badges
  emotes
  reply_metadata
  thread_parent_id
  raw_payload
  message_created_at
  ingested_at

raw_kick_events
  id
  event_name
  kick_message_id
  chatroom_id
  kick_channel_id
  channel_id
  payload
  status
  attempts
  received_at
  processing_started_at
  processed_at
  last_error
  metadata
  created_at
  updated_at

worker_heartbeats
  service_name
  last_seen_at
  metadata
  created_at
  updated_at
```

## Search Contract

Endpoint:

```text
GET /messages?sender=&channel=&q=&start=&end=&cursor=&limit=
GET /messages/export?format=json|csv&sender=&channel=&q=&start=&end=&reply_only=&emote_only=&limit=
```

Access:

- Public endpoint for the public `/search` screen.
- No login is required to search historical messages.

Rules:

- All filters are optional.
- Non-empty filters combine with `AND`.
- Empty `sender` searches all senders.
- Empty `channel` searches all channels.
- Empty `q` searches all message contents.
- Empty all filters returns latest messages across all channels.
- `sender` uses case-insensitive exact matching against sender username/slug snapshots.
- `channel` and `q` use case-insensitive contains matching.
- `start` and `end` filter by `message_created_at`.
- `reply_only=true` restricts results to rows where `message_type = reply`.
- `emote_only=true` restricts results to rows with at least one parsed emote in `emotes`.
- Results are ordered newest-first.
- Cursor pagination uses `(message_created_at, id)`.
- Export is public, reuses `MessageSearchFilters`, supports JSON and CSV, and clamps requested
  rows to `MESSAGE_EXPORT_MAX_ROWS`.
- Frontend `/search` initializes missing date inputs to the last 7 days by default, but the API keeps date filters optional.

## Auth Contract

Use signed JWT session tokens stored in an HttpOnly cookie.

Default super admin seed:

```text
email: admin@kicklogs.local
password: admin123
```

Environment overrides:

```text
DEFAULT_SUPER_ADMIN_EMAIL
DEFAULT_SUPER_ADMIN_PASSWORD
```

Rules:

- Passwords are hashed with Passlib.
- `super_admin` can create admin users.
- Admin routes require authenticated admin or super admin user.
- Public search routes do not require authentication.
- Login route returns user info and sets the session cookie.

## Listener Runtime

The listener service:

- Reads enabled channels from PostgreSQL.
- Resolves missing Kick metadata before subscribing.
- Connects to Kick Pusher websocket.
- Subscribes to `chatrooms.{chatroom_id}.v2` and channel-level events when needed.
- Parses `App\Events\ChatMessageEvent`.
- Stores supported chat events in `raw_kick_events` immediately after minimal parsing.
- Processes raw events from PostgreSQL in worker batches using `FOR UPDATE SKIP LOCKED`.
- Parses `[emote:id:name]` tokens into structured emote values during raw event processing.
- Saves messages through the `IngestMessage` use case outside the websocket read path.
- Marks raw events `processed`, `pending`, or `failed` with attempts and last error.
- Reclaims stale `processing` rows after the configured processing timeout.
- Periodically reconnects to refresh enabled-channel subscriptions so admin channel changes take effect without manually restarting the listener.
- Periodically writes a `listener` heartbeat row at `LISTENER_HEARTBEAT_INTERVAL_SECONDS`.
- Reconnects with backoff after websocket failures.

## Admin Operations Contract

Endpoint:

```text
GET /admin/operations/summary
```

Access:

- Requires an authenticated admin or super admin session.

Response includes:

- core row counts for channels, enabled channels, senders, messages, and raw events
- raw event counts grouped by status
- PostgreSQL database size and table sizes for `chat_messages` and `raw_kick_events`
- latest message, latest raw event receive, latest processed raw event, and oldest pending raw
  event timestamps
- listener heartbeat freshness based on `LISTENER_HEARTBEAT_STALE_AFTER_SECONDS`

## Analytics Contract

Endpoints:

```text
GET /analytics/overview
GET /analytics/message-volume
GET /analytics/top-senders
GET /analytics/top-channels
GET /analytics/top-emotes
```

Access:

- Public read-only endpoints for landing, user profile, and channel profile features.
- No login is required.

Common query parameters:

- `start`
- `end`
- `channel`
- `sender`

Rules:

- Date filters apply to `chat_messages.message_created_at`.
- `sender` scope uses case-insensitive exact matching against sender username/slug and stored
  sender snapshots.
- `channel` scope uses case-insensitive exact matching against channel slug/display name.
- `message-volume` accepts `bucket=hour|day`.
- Top-list endpoints accept `limit` from 1 to 100.

Responses:

- `overview`: total messages, distinct senders, distinct channels, emote usage count, first
  message timestamp, and latest message timestamp.
- `message-volume`: compact bucket rows with bucket start and message count.
- `top-senders`: sender identity/profile fields, message count, first seen in result scope, and
  latest seen in result scope.
- `top-channels`: channel identity/profile fields, message count, first activity in result scope,
  and latest activity in result scope.
- `top-emotes`: emote id/name/token/image URL, usage count, and distinct message count.

## Frontend Architecture

Use Next.js App Router with feature-oriented folders.

```text
apps/web/src/
  app/
    layout.tsx
    page.tsx
    login/
      page.tsx
    search/
      page.tsx
    admin/
      page.tsx
  components/
    ui/
    layout/
    messages/
    search/
    admin/
  features/
    auth/
      api.ts
      types.ts
      use-auth.ts
    search/
      api.ts
      types.ts
      search-form.tsx
      message-list.tsx
    channels/
      api.ts
      types.ts
      channel-admin.tsx
    users/
      api.ts
      types.ts
      user-admin.tsx
    operations/
      api.ts
      operations-dashboard.tsx
    analytics/
      api.ts
    landing/
      landing-page.tsx
  lib/
    api-client.ts
    utils.ts
    constants.ts
  types/
    api.ts
```

Frontend rules:

- `components/ui` contains shadcn/ui primitives only.
- Feature folders own API calls, feature components, and local types.
- Shared layout and generic components live in `components`.
- `lib/api-client.ts` owns base URL, credentials, and response handling.
- Use lucide-react icons for UI controls.
- Use Tailwind for layout and visual styling.
- `/` is the public landing page backed by read-only analytics endpoints.
- `/search` is the primary public app screen.
- `/admin` requires login and manages backend operational state such as followed channels.

## Frontend UI Direction

Search UI should follow the dark professional palette documented in `docs/design/design.md`:

- dark background
- yellow primary actions
- restrained pink accent icons
- compact form-first layout
- fields for user nickname, channel nickname, keyword, start datetime, end datetime
- infinite scroll results below the search form

Result rows render inside one shared outer list container and show:

- sender avatar
- sender nickname
- channel nickname/slug
- timestamp
- content with emote image rendering and fallback

Avoid per-message modal/card components in the high-volume message list. Keep rows dense, with a flexible message column and fixed metadata columns.

## Docker Runtime

Services:

- `postgres`
- `api`
- `listener`
- `web`

The `api` and `listener` services use the same backend source tree and dependency lock. They differ only by command.

Default command:

```powershell
docker compose up --build
```

## Testing Strategy

Backend:

- Use `pytest` and `pytest-asyncio`.
- Unit test use cases with fake repositories.
- Integration test SQLAlchemy repositories against PostgreSQL when practical.
- Test auth, role checks, search filters, pagination, channel resolution, message ingestion, emote parsing, and sender enrichment fallback.

Frontend:

- Typecheck with TypeScript.
- Lint with the Next.js lint setup.
- Test core UI behavior where practical: search form state, infinite scroll, auth gating, admin channel/user forms.

Docker:

- Verify `docker compose up --build`.
- Verify API health route.
- Verify web route loads.
- Verify PostgreSQL volume persists data.
