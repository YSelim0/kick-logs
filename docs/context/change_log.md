# Change Log

This is a living implementation log. Add new entries for each meaningful project change.

## 2026-05-09

- Created project context structure request.
- Created commit convention skill and committed it:
  - `679d936 feat(repo): add commit convention skill`
- Planned Docker Compose dev stack:
  - `postgres`
  - `api`
  - `listener`
  - `web`
- Expanded MVP plan with auth, search semantics, date filters, full payload storage, sender profile enrichment, emote rendering fallback, and one-worker listener model.
- Added `docs/context/recent_changes.md` as the short latest-change handoff file and linked it from `AGENTS.md`.
- Added architecture plan covering clean architecture backend structure, SQLAlchemy/Alembic ORM choice, listener entrypoint, frontend structure, and Docker runtime shape.
- Added UI design guide under `docs/design/design.md` and documented the backend-first development rule.
- Documented that multi-agent development is allowed for non-overlapping work scopes.
- Added search screen design to `docs/design/design.pen` and updated UI palette/rules.
- Refined search design guidance so the provided reference image is used for form structure only, while the app keeps its dark `#26001B` / `#FFF600` palette and avoids blur, glow, and oversized typography.
- Refined `/search` result design to use one outer list container with stacked message rows, circular avatars, inline emotes, and adjusted spacing below the search button.
- Clarified route access: `/search` is public, while `/admin` is the authenticated backend management dashboard for operational tasks like followed-channel management.
- Added `docs/implementation_plan.md` and phase-scoped task files from `docs/tasks/phase1_tasks.md` through `docs/tasks/phase10_tasks.md`.
- Updated agent instructions so implementation agents read the plan and only the matching phase task file before working.

## 2026-05-10

- Locked Phase 1 Docker Compose scope to `postgres` and `api` only; `web` and `listener` services must be added later in their owning phases, with no placeholder services.
- Started Phase 1 by adding root local development defaults:
  - `.gitignore`
  - `.env.example`
  - `README.md`
- Added the initial `apps/api` FastAPI project skeleton with:
  - `uv` project metadata in `apps/api/pyproject.toml`
  - clean architecture package folders
  - settings and logging core modules
  - FastAPI app factory and `GET /health`
  - minimal tests for settings, app factory, and health route
- Added Phase 1 Docker runtime files:
  - root `compose.yaml` with `postgres` and `api` services only
  - `apps/api/Dockerfile`
  - `apps/api/.dockerignore`
  - API hot reload volume setup through Docker Compose
- Updated backend lint instructions in `README.md`.
- Verified backend package import through `uv`.
- Verified `uv run pytest` from `apps/api`: 3 tests passed.
- Verified `uv run ruff check .` from `apps/api`: all checks passed.
- Verified `docker compose config --services`: only `postgres` and `api` are present.
- Attempted `docker compose up --build -d postgres api`, but live Docker startup is pending because the local Docker daemon was not running.
- Retried Docker Compose after daemon access was available. The API image build exposed two container-build issues:
  - `apps/api/pyproject.toml` referenced root `README.md`, which is outside the Docker build context.
  - `uv sync` attempted editable project installation before `src/` was copied into the image.
- Updated API packaging/build flow so dependency sync happens before source copy with `--no-install-project`, then installs the project after `src/` is present.
- Re-ran `docker compose up --build -d postgres api` successfully.
- Verified `GET http://localhost:8000/health` returned `{"status":"ok"}`.
- Phase 1 acceptance is complete.
- Started Phase 2 by adding framework-independent domain entities/value objects and application repository/unit-of-work ports.
- Added Phase 2 SQLAlchemy/Alembic foundation:
  - SQLAlchemy async engine/session factory
  - ORM models for `users`, `channels`, `senders`, and `chat_messages`
  - Alembic async environment
  - initial migration with `pg_trgm`, JSONB columns, dedupe constraints, and search indexes
- Verified the initial migration applies cleanly to local Docker PostgreSQL with `alembic upgrade head`.
- Added SQLAlchemy repository implementations and async unit of work wiring for users, channels, senders, and chat messages.
- Added repository tests for create/read/update flows and message search/pagination repository behavior using isolated transactions.
- Verified full backend test suite: 27 tests passed.
- Verified `ruff check .`: all checks passed.
- Verified `alembic current`: `20260510_0001 (head)`.
- Phase 2 acceptance is complete.
- Started Phase 3 by adding auth security ports and infrastructure services:
  - Passlib password hasher
  - PyJWT token service
  - JWT cookie/session settings
- Added Phase 3 application use cases and seed support:
  - login
  - get current user
  - list admin users
  - create admin user
  - idempotent default super admin seed
- Added Phase 3 HTTP auth/admin user surface:
  - `POST /auth/login`
  - `POST /auth/logout`
  - `GET /auth/me`
  - `GET /admin/users`
  - `POST /admin/users`
  - cookie-based current-user and role dependencies
  - startup super admin seed wiring
