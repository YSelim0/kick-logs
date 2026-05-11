# Kick Logs

Kick Logs is an MVP monorepo for collecting public Kick chat messages from followed channels, storing them in PostgreSQL, and searching historical chat logs through a web UI.

## Current Status

The MVP implementation plan is complete through Phase 10.

Implemented so far:

- FastAPI backend skeleton.
- `GET /health`.
- PostgreSQL and API services in Docker Compose.
- SQLAlchemy async persistence and Alembic initial migration.
- Admin authentication with HttpOnly JWT cookies.
- Default super admin seed.
- Admin user list/create APIs.
- Admin followed-channel management APIs.
- Kick channel metadata resolver.
- Message ingestion use case and emote parser.
- Public `GET /messages` search API with optional filters and cursor pagination.
- Kick listener worker with durable raw event inbox and Pusher websocket ingestion runtime.
- Listener Docker Compose service.
- Backend test/tooling setup.
- Backend Docker/API acceptance checks.
- pnpm workspace.
- Next.js App Router frontend shell.
- Tailwind/shadcn/ui base setup with the dark-only Kick Logs palette.
- Shared typed frontend API client.
- Frontend Docker Compose service.
- Public `/search` UI with filters, default last-7-days date range, URL state, infinite scroll, dense rows, circular avatars, and inline emotes.
- `/login` UI wired to backend auth.
- Authenticated `/admin` dashboard with followed-channel management.
- Super-admin-only admin user management UI.
- Full Docker Compose smoke and sample ingestion-to-search smoke.

The repository is ready for user-managed push.

## Prerequisites

- Docker Desktop
- Python 3.12+
- `uv`
- Node.js 20+
- pnpm 8.11+

If `uv` was installed through `python -m pip install --user uv` and is not on `PATH`, either add the Python user `Scripts` directory to `PATH` or run commands as `python -m uv ...`.

## Environment

Create a local `.env` from the committed example:

```powershell
Copy-Item .env.example .env
```

`.env` is ignored by Git and must not be committed.

Local secrets and credentials live in `.env`. Keep `JWT_SECRET_KEY` at least 32 bytes for HS256, and override the default super admin credentials before using the stack outside local development:

```text
JWT_SECRET_KEY
DEFAULT_SUPER_ADMIN_EMAIL
DEFAULT_SUPER_ADMIN_PASSWORD
```

## Start Backend Stack

Start PostgreSQL, the API, and the listener:

```powershell
docker compose up --build postgres api listener
```

The Docker API and listener services apply Alembic migrations before starting.
When no channels are enabled, the listener stays alive and periodically checks again.
When channels are enabled, the websocket reader persists supported raw chat events before message normalization, then background workers process the durable inbox into `chat_messages`.

To apply migrations manually from the backend project directory:

```powershell
cd apps/api
uv run alembic upgrade head
```

Health check:

```powershell
curl http://localhost:8000/health
```

Expected response:

```json
{"status":"ok"}
```

Default admin login:

```text
email: admin@kicklogs.local
password: admin123
```

Backend API access model:

```text
Public:
GET    /health
GET    /messages

Authentication:
POST   /auth/login
POST   /auth/logout
GET    /auth/me

Admin-only:
GET    /admin/users
POST   /admin/users
GET    /admin/channels
POST   /admin/channels
DELETE /admin/channels/{id}
```

Public message search example:

```powershell
curl "http://localhost:8000/messages?sender=yavuz&q=selam&limit=50"
```

## Start Web App

Install frontend dependencies from the repository root:

```powershell
pnpm install
```

Run the web app locally:

```powershell
pnpm --filter @kick-logs/web dev
```

The web app reads the API URL from:

```text
NEXT_PUBLIC_API_BASE_URL
```

Default local value:

```text
http://localhost:8000
```

Current frontend routes:

```text
/       redirects to /search until future landing content exists
/search  public message search
/login   admin login
/admin   authenticated backend management
```

## Start Full Dev Stack

From the repository root:

```powershell
docker compose up --build
```

Services:

```text
postgres  http://localhost:5432
api       http://localhost:8000
listener  background worker
web       http://localhost:3000
```

For detached local development:

```powershell
docker compose up --build -d
docker compose ps
```

## Backend Tests

From the backend project directory:

```powershell
cd apps/api
uv run pytest
```

Equivalent fallback when `uv` is installed but not on `PATH`:

```powershell
cd apps/api
python -m uv run pytest
```

Lint:

```powershell
cd apps/api
uv run ruff check .
```

## Frontend Checks

From the repository root:

```powershell
pnpm --filter @kick-logs/web typecheck
pnpm --filter @kick-logs/web lint
pnpm --filter @kick-logs/web test
pnpm --filter @kick-logs/web build
```

Run `typecheck` and `build` sequentially. Running both at the same time can race on Next.js generated `.next/types` files.

## Backend Verification

Phase 6 backend acceptance was verified with:

```powershell
cd apps/api
python -m uv run pytest
python -m uv run ruff check .
cd ..\..
docker compose up --build -d postgres api listener
docker compose ps
```

Manual smoke checks:

- `GET /health` returns `{"status":"ok"}`.
- `GET /admin/channels` returns `401` without login.
- Default super admin can login and call `GET /auth/me`.
- Admin can add and disable a followed Kick channel.
- `GET /messages?limit=1` works without login.
- Listener starts through Docker and stays alive when no enabled channels are ready.

Phase 7 frontend foundation was verified with:

- `pnpm --filter @kick-logs/web typecheck`
- `pnpm --filter @kick-logs/web lint`
- `pnpm --filter @kick-logs/web build`
- `docker compose up --build -d web`
- `GET http://localhost:3000` returns HTTP 200.

Phase 8 public search UI was verified with:

- `pnpm --filter @kick-logs/web test`
- `pnpm --filter @kick-logs/web typecheck`
- `pnpm --filter @kick-logs/web lint`
- `pnpm --filter @kick-logs/web build`
- `docker compose up --build -d web`
- `GET http://localhost:3000/search` returns HTTP 200 without login.
- `GET http://localhost:3000/search?sender=yavuz&q=selam` returns the search page and does not render admin placeholder content.

Phase 9 admin dashboard UI was verified with:

- `pnpm --filter @kick-logs/web test`
- `pnpm --filter @kick-logs/web typecheck`
- `pnpm --filter @kick-logs/web lint`
- `pnpm --filter @kick-logs/web build`
- `docker compose up --build -d web`
- `GET http://localhost:3000/search` returns HTTP 200 without login.
- `GET http://localhost:3000/login` returns HTTP 200.
- `GET http://localhost:3000/admin` returns HTTP 200; the client guard redirects unauthenticated users.
- `GET http://localhost:8000/health` returns `{"status":"ok"}`.

Phase 10 final MVP smoke was verified with:

- `python -m uv run pytest` from `apps/api`: 83 tests passed.
- `python -m uv run ruff check .` from `apps/api`: passed.
- `pnpm --filter @kick-logs/web test`: 6 files, 20 tests passed.
- `pnpm --filter @kick-logs/web typecheck`: passed.
- `pnpm --filter @kick-logs/web lint`: passed.
- `pnpm --filter @kick-logs/web build`: passed.
- `docker compose up --build -d`: starts `postgres`, `api`, `listener`, and `web`.
- `GET http://localhost:8000/health`: `{"status":"ok"}`.
- `GET http://localhost:3000/`: HTTP 307 to `/search`.
- `GET http://localhost:3000/search`, `/login`, and `/admin`: HTTP 200.
- Default super admin login succeeds with `admin@kicklogs.local` / `admin123`.
- Authenticated channel add smoke stores Kick channel metadata for `hype`.
- Sample message ingestion through the backend ingestion use case stores a searchable message.
- Public `GET /messages?q=phase10-smoke-20260510235338&limit=5` finds the sample message without authentication.
- Restarting PostgreSQL preserves the sample message in the named volume.

Issue #1 durable ingestion work was verified with:

- `python -m uv run ruff check .` from `apps/api`: passed.
- `python -m uv run alembic upgrade head` from `apps/api`: applied `20260511_0002`.
- `python -m uv run alembic current` from `apps/api`: `20260511_0002 (head)`.
- `python -m uv run pytest` from `apps/api`: 94 tests passed.
- `python -m uv run pytest tests/listener tests/domain tests/database/test_models_metadata.py tests/database/test_alembic_migration.py`: 43 tests passed.
- `python -m uv run pytest tests/database/test_repositories.py tests/messages/test_ingest_message.py tests/listener/test_listener_service.py`: 19 tests passed against local PostgreSQL.
- `docker compose up --build -d postgres api listener`: passed.
- `GET http://localhost:8000/health`: `{"status":"ok"}`.
- Listener logs show raw event storage and raw event worker processing with `pending=0`.

## Kick Integration Notes

The MVP uses Kick web endpoints, Kick Pusher chat events, and inferred emote image URLs. These are not a stable official API contract. If channel resolution, websocket subscription, raw inbox processing, sender profile data, or emote images fail after a Kick-side change, inspect:

- `KICK_PUSHER_URL`
- `LISTENER_WORKER_COUNT`
- `LISTENER_RAW_EVENT_BATCH_SIZE`
- `LISTENER_RAW_EVENT_PROCESSING_TIMEOUT_SECONDS`
- `LISTENER_RAW_EVENT_MAX_ATTEMPTS`
- `LISTENER_CHANNEL_RESYNC_INTERVAL_SECONDS`
- `apps/api/src/kick_logs/infrastructure/kick/channel_resolver.py`
- `apps/api/src/kick_logs/infrastructure/kick/pusher_client.py`
- `apps/api/src/kick_logs/presentation/worker/listener_service.py`
- `apps/api/src/kick_logs/infrastructure/database/repositories/sqlalchemy_raw_event_repository.py`

## Git Workflow

Agents create local commits only. The user pushes manually.

Commit messages use:

```text
feat(scope): title
```
