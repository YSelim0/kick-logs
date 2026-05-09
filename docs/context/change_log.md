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
