# Phase 1 Tasks: Backend/Docker Foundation

## Scope

Create the minimum runnable backend stack. This phase owns only root runtime scaffolding, the `apps/api` project shell, Docker Compose for `postgres` + `api`, and backend health verification.

Do not implement database tables, auth, admin APIs, message search, listener logic, or frontend UI in this phase.

## Inputs

- `docs/project_plan.md`
- `docs/architecture.md`
- `docs/implementation_plan.md`
- Existing `.env` decisions and default admin credentials from docs

## Tasks

- [ ] Create root development files:
  - [x] `.gitignore` excluding `.env`, virtual environments, `__pycache__`, `.pytest_cache`, build outputs, logs, dependency folders.
  - [x] `.env.example` with non-secret local defaults for database URL, API settings, JWT placeholder, and default super admin env names.
  - [x] `README.md` with local prerequisites, Docker startup command, backend-only health check, and commit/push note.
- [x] Create `compose.yaml` with:
  - [x] `postgres` service using PostgreSQL and a named volume.
  - [x] `api` service built from `apps/api`, depending on `postgres`.
  - [x] API env wired from `.env`/defaults.
  - [x] API port exposed for local development.
  - [x] No `web`, `listener`, or placeholder services in Phase 1.
- [x] Scaffold `apps/api` as a `uv` Python project:
  - [x] `pyproject.toml` with FastAPI, Uvicorn, Pydantic settings, pytest, pytest-asyncio, ruff or equivalent dev tooling.
  - [x] Package layout under `apps/api/src/kick_logs/`.
  - [x] Empty clean architecture folders matching `docs/architecture.md`.
- [x] Implement backend foundation:
  - [x] Settings object in `core/config.py`.
  - [x] Logging setup in `core/logging.py`.
  - [x] FastAPI app factory in `presentation/http/app.py`.
  - [x] Entrypoint in `main.py`.
  - [x] `GET /health` returning stable JSON such as `{"status":"ok"}`.
- [x] Add container files:
  - [x] `apps/api/Dockerfile` for local dev.
  - [x] Volume mount or dev command for hot reload.
- [x] Add minimal tests:
  - [x] Health route test.
  - [x] Settings import test.
  - [x] App factory import test.

## Acceptance Criteria

- [x] `docker compose up --build postgres api` starts without frontend/listener.
- [x] `GET /health` returns success.
- [x] `uv run pytest` from `apps/api` passes.
- [x] No business logic from later phases is implemented.
- [x] Docs/context are updated with what was created.

Note: `docker compose config --services` was verified and returned only `postgres` and `api`. Live Docker verification was completed with `docker compose up --build -d postgres api`, followed by `GET /health` returning `{"status":"ok"}`.

## Handoff

Phase 2 can assume a runnable FastAPI skeleton, working config, Dockerized PostgreSQL service, and a package structure ready for domain/database code.
