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
