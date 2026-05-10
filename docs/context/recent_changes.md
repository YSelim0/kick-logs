# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest

- Phase 5 is complete and committed through:
  - `b0eee69 feat(listener): add worker foundation`
  - `29abaf8 feat(listener): add pusher runtime`
  - `1f98b3a feat(listener): add docker service`
  - `c80afaa feat(listener): align pusher subscription`
- Phase 6 backend verification and acceptance is complete.
- Phase 6 fixes:
  - default local/Compose `JWT_SECRET_KEY` is now at least 32 bytes
  - `bcrypt` is pinned to `>=4.0.1,<4.1` for Passlib compatibility
- Phase 6 verification:
  - `python -m uv run pytest`: 83 passed
  - `python -m uv run ruff check .`: passed
  - `python -m uv run alembic current`: `20260510_0001 (head)`
  - `docker compose up --build -d postgres api listener`: passed
  - `docker compose ps`: `postgres`, `api`, and `listener` up
  - `GET /health`: passed
  - unauthenticated `GET /admin/channels`: 401
  - default super admin login and `GET /auth/me`: passed
  - admin channel add/disable smoke with slug `hype`: passed
  - public `GET /messages?limit=1`: passed without login
  - listener logs useful idle status when no enabled channels are ready
- README now documents backend startup, verification, route access, env/local secrets, and Kick integration fragility.
- Scope remains backend-only: no frontend or web Docker service.

## Commit Context

- Previous committed unit:
  - `c80afaa feat(listener): align pusher subscription`
- Latest completed unit:
  - Phase 6 backend acceptance
- Commit message for this unit:
  - `feat(backend): complete phase six acceptance`
