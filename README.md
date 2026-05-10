# Kick Logs

Kick Logs is an MVP monorepo for collecting public Kick chat messages from followed channels, storing them in PostgreSQL, and searching historical chat logs through a web UI.

## Current Status

Backend implementation is complete and verified through Phase 6.
Frontend foundation is complete through Phase 7.

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
- Kick listener worker with Pusher websocket ingestion runtime.
- Listener Docker Compose service.
- Backend test/tooling setup.
- Backend Docker/API acceptance checks.
- pnpm workspace.
- Next.js App Router frontend shell.
- Tailwind/shadcn/ui base setup with the dark-only Kick Logs palette.
- Shared typed frontend API client.
- Frontend Docker Compose service.

Final `/search` and `/admin` UI workflows are intentionally implemented in later phases.

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

The current frontend routes are foundation shells only:

```text
/
/search
/login
/admin
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
pnpm --filter @kick-logs/web build
```

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

## Kick Integration Notes

The MVP uses Kick web endpoints, Kick Pusher chat events, and inferred emote image URLs. These are not a stable official API contract. If channel resolution, websocket subscription, sender enrichment, or emote images fail after a Kick-side change, inspect:

- `KICK_PUSHER_URL`
- `apps/api/src/kick_logs/infrastructure/kick/channel_resolver.py`
- `apps/api/src/kick_logs/infrastructure/kick/pusher_client.py`
- `apps/api/src/kick_logs/infrastructure/kick/sender_profile_resolver.py`

## Git Workflow

Agents create local commits only. The user pushes manually.

Commit messages use:

```text
feat(scope): title
```
