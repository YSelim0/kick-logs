# Kick Logs Project Plan

## Summary

Kick Logs is an MVP monorepo for collecting public Kick chat messages from followed channels, storing the full useful payload in PostgreSQL, and searching historical messages through a Next.js web UI.

The ingestion method resolves Kick channel/chatroom metadata through Kick web endpoints, subscribes to Kick chat Pusher channels, parses `App\Events\ChatMessageEvent` payloads, enriches senders when possible, and persists messages.

Default local startup:

```powershell
docker compose up --build
```

## MVP Goals

- Track one or more Kick channels.
- Add/remove followed channels from an admin panel.
- Persist messages indefinitely.
- Store all useful available message data, including raw payloads.
- Search messages with optional filters:
  - sender nickname
  - channel nickname/slug
  - message content
  - start datetime
  - end datetime
- Show results with infinite scroll.
- Provide admin user management with a default super admin.
- Run locally through Docker Compose.

## Architecture

- `apps/api`: FastAPI backend for auth, search, channel admin, user admin, health checks, and database access.
- `apps/web`: Next.js frontend using pnpm, Tailwind, shadcn/ui, and lucide-react.
- `listener` Docker service: Python Kick chat ingestion worker entrypoint from the backend package.
- PostgreSQL runs in Docker with a named volume.
- Python dependency/project tooling uses `uv`.
- Frontend package management uses `pnpm`.
- Detailed backend/frontend structure lives in `docs/architecture.md`.
- Active post-MVP feature planning and handoff task files live in `docs/implementation_plan.md` and `docs/tasks/`.
- The completed MVP implementation plan is archived under `docs/archive/`.

## Docker Services

- `postgres`: PostgreSQL database.
- `api`: FastAPI dev server with hot reload.
- `listener`: single Python worker that subscribes to all enabled followed channels.
- `web`: Next.js dev server.

## Auth And Admin

- Full login system is required for MVP.
- Login is required only for admin/backend management flows, not for public search.
- Roles:
  - `super_admin`
  - `admin`
- Default super admin:
  - email: `admin@kicklogs.local`
  - password: `admin123`
- Default credentials must be overridable with env:
  - `DEFAULT_SUPER_ADMIN_EMAIL`
  - `DEFAULT_SUPER_ADMIN_PASSWORD`
- Passwords must be hashed, never stored as plain text.
- Super admin can create new admin users from the UI.
- Admin dashboard route: `/admin`.
- Admin dashboard manages backend operational state, including followed channels for ingestion.

## Search Behavior

Search route: `/search`.

The `/search` screen is public and does not require login. It exposes historical chat search to any visitor.

Search API:

```text
GET /messages?sender=&channel=&q=&start=&end=&cursor=&limit=
```

Filter semantics:

- All filters are optional.
- Non-empty filters combine with `AND`.
- Empty `sender` means all users.
- Empty `channel` means all channels.
- Empty `q` means all message contents.
- Empty all filters returns latest messages across all channels.
- `sender` uses case-insensitive exact matching against sender username/slug snapshots.
- `channel` and `q` use case-insensitive contains matching.
- `start` and `end` filter by message timestamp.
- Results are ordered newest-first.
- Infinite scroll uses cursor pagination based on `(created_at, id)`.
- The public `/search` UI defaults `start` to 7 days before the current local date/time and `end` to the current local date/time. Users can clear those fields to omit date filters.
- Bare `/search` does not automatically fetch latest messages; the user must submit the form or open a URL with query parameters.

Example queries:

- `sender=yavuz`: all messages from sender username/slug exactly matching `yavuz` across all channels.
- `sender=yavuz&q=selam`: messages from sender username/slug exactly matching `yavuz` containing `selam`.
- `channel=exampleChannel&q=hello`: messages in channels matching `exampleChannel` containing `hello`.
- `q=hello`: all messages containing `hello` across all channels.

## Data Model Draft

- `users`: admin accounts with email, password hash, role, timestamps.
- `channels`: followed Kick channels with slug, Kick channel id, Kick chatroom id, display metadata, profile image/banner when available, enabled status, timestamps.
- `chat_messages`: normalized message records with Kick message id, channel reference, chatroom id, content, message type, sender reference fields, parsed emotes, reply/thread metadata, raw payload, message timestamp, ingestion timestamp.
- `senders` or sender snapshot fields: store sender id, username, slug, color, badges, and profile image when available.

Store raw payload as JSONB so future fields remain queryable without needing to re-ingest historical messages.

## Kick Payload And Enrichment

Live chat payload fields observed from Pusher:

- `id`
- `chatroom_id`
- `content`
- `type`
- `created_at`
- `sender.id`
- `sender.username`
- `sender.slug`
- `sender.identity.color`
- `sender.identity.badges`
- `metadata.message_ref`
- reply metadata such as `metadata.original_sender`, `metadata.original_message`
- `thread_parent_id` when present

Sender profile images are not reliably present in chat events. Enrich sender profiles separately through Kick web endpoints using sender slug when possible and cache the result.

Emotes arrive in message content as:

```text
[emote:37226:KEKW]
```

Parse and store emotes as structured data:

- `id`
- `name`
- original token
- inferred image URL

Render emote images with:

```text
https://files.kick.com/emotes/{id}/fullsize
```

If the image fails, fall back to emote name or original token.

## API Draft

- `GET /health`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /auth/me`
- `GET /messages`
- `GET /admin/channels`
- `POST /admin/channels`
  - body: channel slug/nickname
  - resolves Kick metadata and stores/enables channel
- `DELETE /admin/channels/{id}`
  - disables or removes a followed channel for MVP
- `GET /admin/users`
- `POST /admin/users`
  - super admin only
  - creates admin users
- `GET /admin/operations/summary`
  - admin only
  - returns listener freshness, storage size, raw event backlog/status, and ingest timestamps

## Frontend Draft

- `/search`: primary app search screen.
- `/admin`: authenticated admin dashboard for backend operations.
- `/`: redirects to `/search` until a future landing page is intentionally designed.

Search UI follows the dark professional palette documented in `docs/design/design.md`:

- `Kullanıcı Adı`
- `Kanal Adı`
- `Aramak istediğiniz kelime`
- `Başlangıç`
- `Bitiş`
- yellow search button with lucide search icon

Result rows should render inside one shared outer list container and show:

- sender avatar
- sender nickname
- channel nickname/slug
- timestamp
- message content with emote image rendering/fallback

Do not render each message as its own modal-like card. The list should stay dense and efficient for many messages.

Admin UI should support:

- login
- backend operational management
- followed channel list
- add channel by slug/nickname
- remove/disable channel
- create admin user when current user is super admin
- view operations health, storage growth, raw event backlog, and listener freshness

## Test Plan

- Backend:
  - auth login and role checks
  - default super admin seed
  - channel slug resolution
  - message search filter combinations
  - date range filtering
  - cursor pagination
  - message normalization and deduplication
- Listener:
  - Kick payload parsing
  - emote token parsing
  - multi-channel subscription from enabled channels
  - reconnect behavior
  - sender profile enrichment fallback
- Frontend:
  - `/search` filter form
  - infinite scroll
  - emote fallback rendering
  - `/admin` login, channel management, user creation
- Docker:
  - `docker compose up --build`
  - API health check passes
  - web loads
  - postgres volume persists data

## MVP Constraints

- No official Kick OAuth/webhook integration in MVP; use Kick web Pusher events.
- Kick web endpoints and emote image URLs are undocumented and may change; code must fail gracefully.
- No push is performed by agents; user manually pushes commits.
- Prefer small, commit-sized implementation steps.
