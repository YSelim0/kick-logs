# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest

- Added repository agent instructions in `AGENTS.md`.
- Added `CLAUDE.md` to route Claude agents through `AGENTS.md`.
- Added `docs/project_plan.md` with the Kick Logs MVP plan.
- Added context memory files under `docs/context`.
- Removed references to the external/non-repo prototype folder from project docs.
- Expanded the MVP plan with:
  - Docker Compose dev stack
  - FastAPI backend
  - Next.js frontend
  - PostgreSQL persistence
  - full admin login
  - default super admin credentials
  - `/search` and `/admin` routes
  - optional AND-based search filters
  - date range filters
  - one listener worker for all enabled channels
  - raw Kick payload storage
  - sender profile image enrichment
  - emote parsing and image fallback rendering

## Commit Context

- This context scaffold is the next docs unit of work after:
  - `679d936 feat(repo): add commit convention skill`
