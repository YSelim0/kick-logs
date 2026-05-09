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
  - [ ] `.gitignore` excluding `.env`, virtual environments, `__pycache__`, `.pytest_cache`, build outputs, logs, dependency folders.
  - [ ] `.env.example` with non-secret local defaults for database URL, API settings, JWT placeholder, and default super admin env names.
  - [ ] `README.md` with local prerequisites, Docker startup command, backend-only health check, and commit/push note.
- [ ] Create `compose.yaml` with:
  - [ ] `postgres` service using PostgreSQL and a named volume.
  - [ ] `api` service built from `apps/api`, depending on `postgres`.
  - [ ] API env wired from `.env`/defaults.
  - [ ] API port exposed for local development.
  - [ ] No `web`, `listener`, or placeholder services in Phase 1.
- [ ] Scaffold `apps/api` as a `uv` Python project:
  - [ ] `pyproject.toml` with FastAPI, Uvicorn, Pydantic settings, pytest, pytest-asyncio, ruff or equivalent dev tooling.
  - [ ] Package layout under `apps/api/src/kick_logs/`.
  - [ ] Empty clean architecture folders matching `docs/architecture.md`.
- [ ] Implement backend foundation:
  - [ ] Settings object in `core/config.py`.
  - [ ] Logging setup in `core/logging.py`.
  - [ ] FastAPI app factory in `presentation/http/app.py`.
  - [ ] Entrypoint in `main.py`.
  - [ ] `GET /health` returning stable JSON such as `{"status":"ok"}`.
- [ ] Add container files:
  - [ ] `apps/api/Dockerfile` for local dev.
  - [ ] Volume mount or dev command for hot reload.
- [ ] Add minimal tests:
  - [ ] Health route test.
  - [ ] Settings import test.
  - [ ] App factory import test.

## Acceptance Criteria

- [ ] `docker compose up --build postgres api` starts without frontend/listener.
- [ ] `GET /health` returns success.
- [ ] `uv run pytest` from `apps/api` passes.
- [ ] No business logic from later phases is implemented.
- [ ] Docs/context are updated with what was created.

## Handoff

Phase 2 can assume a runnable FastAPI skeleton, working config, Dockerized PostgreSQL service, and a package structure ready for domain/database code.
