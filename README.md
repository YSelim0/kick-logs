<p align="center">
  <img src="./apps/web/public/app-logo.png" alt="Kick Logs logo" width="128" />
</p>

<h1 align="center">Kick Logs</h1>

<p align="center">
  A self-hosted Kick chat logger with durable ingestion, PostgreSQL storage, and a searchable web UI.
</p>

<p align="center">
  <a href="https://github.com/YSelim0/kick-logs">Repository</a>
  ·
  <a href="https://github.com/YSelim0/kick-logs/issues">Issues</a>
  ·
  <a href="https://github.com/YSelim0/kick-logs/pulls">Pull Requests</a>
</p>

<p align="center">
  <a href="https://github.com/YSelim0/kick-logs/actions/workflows/python-tests.yml">
    <img src="https://github.com/YSelim0/kick-logs/actions/workflows/python-tests.yml/badge.svg" alt="Python CI" />
  </a>
  <a href="https://buymeacoffee.com/yavuzselim">
    <img src="https://img.shields.io/badge/Buy%20Me%20a%20Coffee-yavuzselim-FFF600?style=for-the-badge&logo=buymeacoffee&logoColor=000000" alt="Buy Me a Coffee" />
  </a>
</p>

> Kick Logs is an unofficial community project. It uses Kick web endpoints,
> Kick Pusher chat events, and inferred emote image URLs. These are not a
> stable official Kick API contract and can change without notice.

## Overview

Kick Logs collects public chat messages from followed Kick channels, stores the
useful payload in PostgreSQL, and lets users search historical messages from a
Next.js web interface.

The project is built as a monorepo:

- `apps/api`: FastAPI backend, PostgreSQL persistence, auth, admin APIs, message search
- `apps/web`: Next.js frontend, public search UI, admin dashboard
- `listener`: Docker service that subscribes to Kick chat events and stores raw events first
- `docs`: architecture, implementation notes, task plans, and design decisions

## Features

- Public `/` landing page with self-hosted positioning and live analytics summary.
- Public `/search` page with optional filters:
  - sender nickname
  - channel nickname/slug
  - message text
  - start datetime
  - end datetime
- Infinite-scroll results ordered newest first.
- Dense message rows with circular sender avatars.
- Inline Kick emote rendering with text fallback.
- Reply rendering: replied-to sender/content is shown above reply messages.
- Public `/users/[slug]` pages with sender identity, analytics, top channels, top emotes, and
  latest messages.
- Sender names and avatars in search results link to public user profiles.
- Public `/channels/[slug]` pages with stored Kick channel metadata, activity analytics, top
  senders, top emotes, and latest messages.
- Channel labels in search results and admin channel rows link to public channel profiles.
- Admin login with HttpOnly JWT cookie sessions.
- Default local super admin seed.
- Admin dashboard for followed-channel management.
- Super-admin-only admin user creation.
- Admin operations dashboard for listener freshness, storage growth, raw event backlog, and
  ingest timestamps.
- Admin data management panel for database/table sizes, retention settings, dry-run cleanup
  previews, and explicit confirmed deletion.
- Durable raw event inbox:
  - websocket reader stores raw chat events first
  - workers process raw events into normalized messages
  - stale processing rows can be reclaimed
  - duplicate message writes are avoided by Kick message id
- Docker Compose runtime for PostgreSQL, API, listener, and web.

## Tech Stack

- Backend: Python 3.12, FastAPI, SQLAlchemy 2.x async ORM, Alembic, asyncpg
- Frontend: Next.js App Router, TypeScript, Tailwind CSS, shadcn/ui primitives, lucide-react
- Database: PostgreSQL 16 with JSONB and `pg_trgm`
- Tooling: `uv` for Python, `pnpm` for frontend packages
- Runtime: Docker Compose

## Quick Start

Clone the repository:

```powershell
git clone https://github.com/YSelim0/kick-logs.git
cd kick-logs
```

Create a local environment file:

```powershell
Copy-Item .env.example .env
```

On macOS/Linux:

```bash
cp .env.example .env
```

Start the full stack:

```powershell
docker compose up --build -d
```

Open:

- Landing page: http://localhost:3000
- Public search: http://localhost:3000/search
- Public user profile: http://localhost:3000/users/{sender-slug}
- Public channel profile: http://localhost:3000/channels/{channel-slug}
- Admin login: http://localhost:3000/login
- API health: http://localhost:8000/health

Default local admin:

```text
email: admin@kicklogs.local
password: admin123
```

Before using the project outside local development, change these values in
`.env`:

```text
JWT_SECRET_KEY
DEFAULT_SUPER_ADMIN_EMAIL
DEFAULT_SUPER_ADMIN_PASSWORD
POSTGRES_PASSWORD
```

## Basic Usage

1. Start the stack with Docker Compose.
2. Open `http://localhost:3000/login`.
3. Login with the local admin credentials.
4. Go to `/admin`.
5. Check the operations dashboard for listener freshness, storage size, raw event status, and
   latest ingest timestamps.
6. Review the data management panel before changing retention or cleanup settings.
7. Add a Kick channel by slug/nickname.
8. Open `/channels/{channel-slug}` to inspect stored channel activity.
9. Keep the `listener` service running.
10. Search collected messages from `/search`.

Useful listener logs:

```powershell
docker compose logs -f listener
```

Useful service status:

```powershell
docker compose ps
```

Stop the stack:

```powershell
docker compose down
```

Stop the stack and remove persisted PostgreSQL data:

```powershell
docker compose down -v
```

## Services

| Service    | Purpose                    | Local URL               |
| ---------- | -------------------------- | ----------------------- |
| `postgres` | PostgreSQL database        | `localhost:5432`        |
| `api`      | FastAPI HTTP API           | `http://localhost:8000` |
| `listener` | Kick chat ingestion worker | background service      |
| `web`      | Next.js web app            | `http://localhost:3000` |

The API and listener automatically run Alembic migrations before startup in
Docker Compose.

## API Surface

Public:

```text
GET /health
GET /messages
GET /messages/export
GET /analytics/overview
GET /analytics/message-volume
GET /analytics/top-senders
GET /analytics/top-channels
GET /analytics/top-emotes
GET /users/{slug}/analytics
GET /channels/{slug}/analytics
```

Auth:

```text
POST /auth/login
POST /auth/logout
GET  /auth/me
```

Admin:

```text
GET    /admin/channels
POST   /admin/channels
DELETE /admin/channels/{id}
GET    /admin/users
POST   /admin/users
GET    /admin/operations/summary
GET    /admin/data-management/summary
PUT    /admin/data-management/retention-settings
POST   /admin/data-management/cleanup/preview
POST   /admin/data-management/cleanup/confirm
```

Example public search:

```powershell
curl "http://localhost:8000/messages?sender=yavuz&q=selam&limit=50"
```

Example filtered export:

```powershell
curl "http://localhost:8000/messages/export?format=csv&q=selam&reply_only=true&limit=500"
```

Search filters combine with `AND`. Empty filters are omitted. The `sender`
filter matches username/slug exactly, case-insensitively. Channel and content
filters use case-insensitive contains matching. Empty all filters returns latest
messages across all followed channels. `reply_only=true` limits results to reply
messages, and `emote_only=true` limits results to messages with parsed emotes.
Exports use the same filters and are capped by `MESSAGE_EXPORT_MAX_ROWS`.

Analytics endpoints are public read-only endpoints for landing/profile features.
They accept optional `start`, `end`, `channel`, and `sender` query params where
relevant. Message volume also accepts `bucket=hour|day`; top lists accept
`limit` up to 100. Analytics `sender` scope uses case-insensitive exact matching
against sender username/slug snapshots; `channel` scope uses case-insensitive
exact matching against channel slug/display name.

Example analytics calls:

```powershell
curl "http://localhost:8000/analytics/overview"
curl "http://localhost:8000/analytics/message-volume?bucket=day&channel=hype"
curl "http://localhost:8000/analytics/top-senders?channel=hype&limit=10"
curl "http://localhost:8000/analytics/top-channels?sender=yavuz&limit=10"
curl "http://localhost:8000/analytics/top-emotes?channel=hype&sender=yavuz"
curl "http://localhost:8000/users/yavuz/analytics"
curl "http://localhost:8000/channels/hype/analytics"
```

## Local Development

Install frontend dependencies:

```powershell
pnpm install
```

Run the Next.js app:

```powershell
pnpm --filter @kick-logs/web dev
```

Backend commands are run from `apps/api`:

```powershell
cd apps/api
python -m uv run alembic upgrade head
python -m uv run pytest
python -m uv run ruff check .
python -m uv run ruff format --check .
```

Frontend checks are run from the repository root:

```powershell
pnpm --filter @kick-logs/web test
pnpm --filter @kick-logs/web typecheck
pnpm --filter @kick-logs/web lint
pnpm --filter @kick-logs/web build
pnpm format:check
```

Run `typecheck` and `build` sequentially. Running both at the same time can
race on Next.js generated `.next/types` files.

Formatting:

```powershell
pnpm format
cd apps/api
python -m uv run ruff format .
```

Use Prettier for frontend, JSON, YAML, and Markdown files. Use Ruff Format for
Python files. Both formatters are configured to use 100-column line width and
the repository's current double-quote style.

## Continuous Integration

GitHub Actions runs backend and formatting checks on pull requests targeting
`main` or `dev`, and on pushes to `main` or `dev`.

The Python workflow starts a PostgreSQL 16 service, installs backend
dependencies with `uv`, applies Alembic migrations, then runs:

```powershell
python -m uv run ruff format --check .
python -m uv run ruff check .
python -m uv run pytest
```

The code style workflow installs frontend dependencies and runs:

```powershell
pnpm format:check
```

## Repository Structure

```text
kick-logs/
  apps/
    api/
      alembic/
      src/kick_logs/
      tests/
    web/
      public/
      src/
  docs/
    context/
    design/
    tasks/
  compose.yaml
  README.md
```

Backend dependency direction:

```text
presentation -> application -> domain
infrastructure -> application/domain
```

Domain code stays independent from FastAPI, SQLAlchemy, Pydantic, and external
Kick clients.

## Data Captured

Kick Logs stores normalized fields and the raw payload so the project can adapt
as Kick message shapes evolve.

Stored data includes:

- channel metadata
- sender metadata and profile image URL when available
- message content
- message type
- sender badges
- parsed emotes
- reply metadata
- thread parent id
- original raw Kick payload

Messages are persisted indefinitely unless the operator removes data manually or
runs a confirmed retention cleanup from the admin data management panel.

## Data Management And Backups

The admin data management panel lives under `/admin`. It is admin-only and shows
database size, table sizes, row counts, and the active retention settings for
messages and raw Kick events.

Retention settings support:

- keep forever
- 30 days
- 90 days

Retention settings do not delete data by themselves. They define the cutoff used
by the old-message and old-raw-event cleanup actions. A destructive cleanup must
first run a dry-run preview, then the admin must type the exact confirmation text
returned by the backend.

Cleanup targets:

- old messages according to message retention
- old raw events according to raw-event retention
- all stored messages/raw events for a specific channel slug
- all stored messages/raw events for a specific sender

Back up PostgreSQL before running large cleanup operations. The Compose service
uses the values from `.env`; adjust the `-U` and `-d` values below if you changed
`POSTGRES_USER` or `POSTGRES_DB`.

Create a compressed PostgreSQL dump:

```powershell
New-Item -ItemType Directory -Force backups
docker compose exec postgres pg_dump -U kick_logs -d kick_logs -Fc -f /tmp/kick_logs.dump
docker compose cp postgres:/tmp/kick_logs.dump .\backups\kick_logs.dump
docker compose exec postgres rm /tmp/kick_logs.dump
```

Restore a dump into the Docker Compose database:

```powershell
docker compose stop api listener web
docker compose cp .\backups\kick_logs.dump postgres:/tmp/kick_logs.dump
docker compose exec postgres pg_restore -U kick_logs -d kick_logs --clean --if-exists /tmp/kick_logs.dump
docker compose exec postgres rm /tmp/kick_logs.dump
docker compose up -d api listener web
```

`docker compose down -v` removes the PostgreSQL volume. Use it only when you
intentionally want to delete all stored data.

## Configuration

Copy `.env.example` to `.env` and adjust values as needed.

Important variables:

```text
BACKEND_CORS_ORIGINS=http://localhost:3000
NEXT_PUBLIC_API_BASE_URL=http://localhost:8000
DATABASE_URL=postgresql+asyncpg://kick_logs:kick_logs@postgres:5432/kick_logs
JWT_SECRET_KEY=change-me-for-local-development-secret-key
KICK_PUSHER_URL=...
LISTENER_WORKER_COUNT=4
LISTENER_RAW_EVENT_BATCH_SIZE=100
LISTENER_CHANNEL_RESYNC_INTERVAL_SECONDS=60
```

Never commit `.env`, secrets, local database dumps, virtual environments, or
generated dependency/build folders.

## Support

Kick Logs is an open source self-hosted project. If it helps your workflow or
you want to support continued development, you can
[buy me a coffee](https://buymeacoffee.com/yavuzselim).

## Contributing

Contributions are welcome. The goal is for this repository to be easy to fork,
run locally, and improve.

Recommended flow:

1. Fork https://github.com/YSelim0/kick-logs
2. Create a feature branch from `main`.
3. Open or pick an issue before larger changes.
4. Keep changes scoped and reviewable.
5. Add or update tests for behavior changes.
6. Update docs when behavior, setup, or architecture changes.
7. Run the relevant checks.
8. Open a pull request with a clear summary.

Commit format:

```text
feat(scope): title
fix(scope): title
docs(scope): title
test(scope): title
refactor(scope): title
```

Examples:

```text
feat(search): render reply context
fix(listener): reconnect after channel changes
docs(readme): improve setup guide
```

For UI changes, read `docs/design/design.md` first. For backend architecture
changes, read `docs/architecture.md` first.

## Suggested First Issues

Good contribution areas:

- better Kick payload fixtures
- more listener resilience tests
- richer sender profile enrichment
- search performance tuning
- export tools for logs
- UI polish for long messages and mobile rows
- deployment examples for VPS environments
- CI workflow setup

## Security And Operations

- Change default credentials before any non-local use.
- Use a strong `JWT_SECRET_KEY`.
- Treat Kick integration failures as expected operational events because the
  ingestion path relies on unofficial web behavior.
- Use `/admin` to check listener freshness, database/table size, failed raw events, pending raw
  events, and the latest ingest timestamps before digging into Docker logs.
- Keep PostgreSQL backups if the logs matter.
- Review data retention expectations before running against large channels.

## Project Status

Kick Logs is an MVP. It is usable locally through Docker Compose, but the Kick
integration should be considered best-effort because it depends on undocumented
Kick web behavior.

Current quality gates used during development:

- backend test suite with `pytest`
- backend lint with `ruff`
- frontend tests with Vitest and React Testing Library
- frontend TypeScript typecheck
- frontend lint
- frontend production build
- Docker Compose smoke checks

## License

Kick Logs is released under the MIT License. See [LICENSE](./LICENSE).
