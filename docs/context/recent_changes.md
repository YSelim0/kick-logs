# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest

- Added `docs/implementation_plan.md` as the sequential MVP execution plan.
- Added phase-scoped task files under `docs/tasks/`:
  - `phase1_tasks.md` through `phase10_tasks.md`
- Phase order is backend-first:
  - backend/Docker foundation
  - database/domain/repositories
  - auth/admin users
  - channel/search/ingestion APIs
  - listener worker
  - backend acceptance
  - frontend foundation
  - public search UI
  - admin dashboard
  - full-stack polish
- `AGENTS.md` now requires agents to read the implementation plan and the matching phase task file before implementation work.
- Task files explicitly forbid crossing into later phase scope unless the user changes the plan.

## Commit Context

- Last committed docs unit:
  - `8a32d8d feat(docs): add search design guidelines`
