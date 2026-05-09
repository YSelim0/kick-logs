# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest

- Added `docs/architecture.md`.
- Architecture locks backend clean architecture layers:
  - domain
  - application
  - infrastructure
  - presentation
- Backend ORM choice is SQLAlchemy 2.x async ORM with asyncpg and Alembic.
- API and listener run as separate Docker services but share one Python backend package.
- Frontend architecture uses Next.js App Router with feature-oriented folders.
- `AGENTS.md` now requires agents to read `docs/architecture.md`.

## Commit Context

- Last committed docs unit:
  - `a6b5105 feat(docs): add project planning context`
