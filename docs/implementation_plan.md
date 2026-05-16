# Implementation Plan

There is no active feature implementation plan right now.

The current application runtime is already cut over to:

- Go API and listener under `apps/api-go`
- ClickHouse for chat messages, raw Kick events, exports, analytics, and profile aggregates
- SQLite for admin/control-plane state
- Next.js web UI under `apps/web`

Completed implementation tracks are archived under `docs/archive/`:

- `mvp/`: original Python/FastAPI/PostgreSQL MVP
- `post_mvp/`: completed post-MVP feature roadmap
- `go_rewrite/`: completed Go + ClickHouse rewrite, API parity, listener parity, data migration,
  and cutover plan

Future work should start by replacing this file with a new scoped plan and adding matching task
files under `docs/tasks/`. Until then, `docs/tasks/` has no active implementation scope.
