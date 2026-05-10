# Kick Logs

Kick Logs is an MVP monorepo for collecting public Kick chat messages from followed channels, storing them in PostgreSQL, and searching historical chat logs through a web UI.

## Current Status

Backend implementation is complete through Phase 5.

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

Frontend is intentionally added in later phases.

## Prerequisites

- Docker Desktop
- Python 3.12+
- `uv`

If `uv` was installed through `python -m pip install --user uv` and is not on `PATH`, either add the Python user `Scripts` directory to `PATH` or run commands as `python -m uv ...`.

## Environment

Create a local `.env` from the committed example:

```powershell
Copy-Item .env.example .env
```

`.env` is ignored by Git and must not be committed.

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

Phase 4 backend API routes:

```text
GET    /messages
GET    /admin/channels
POST   /admin/channels
DELETE /admin/channels/{id}
```

Public message search example:

```powershell
curl "http://localhost:8000/messages?sender=yavuz&q=selam&limit=50"
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

## Git Workflow

Agents create local commits only. The user pushes manually.

Commit messages use:

```text
feat(scope): title
```
