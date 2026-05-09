# Kick Logs

Kick Logs is an MVP monorepo for collecting public Kick chat messages from followed channels, storing them in PostgreSQL, and searching historical chat logs through a web UI.

## Current Phase

Development is currently in Phase 1: backend and Docker foundation.

Phase 1 contains only:

- FastAPI backend skeleton.
- `GET /health`.
- PostgreSQL and API services in Docker Compose.
- Backend test/tooling setup.

Frontend, listener runtime, auth, database models, admin APIs, and message search are intentionally added in later phases.

## Prerequisites

- Docker Desktop
- Python 3.12+
- `uv`

## Environment

Create a local `.env` from the committed example:

```powershell
Copy-Item .env.example .env
```

`.env` is ignored by Git and must not be committed.

## Start Backend Stack

After Phase 1 files are complete, start PostgreSQL and the API:

```powershell
docker compose up --build postgres api
```

Health check:

```powershell
curl http://localhost:8000/health
```

Expected response:

```json
{"status":"ok"}
```

## Backend Tests

From the backend project directory:

```powershell
cd apps/api
uv run pytest
```

## Git Workflow

Agents create local commits only. The user pushes manually.

Commit messages use:

```text
feat(scope): title
```
