# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest

- Phase 3 auth/admin user foundation is complete.
- Implemented security services:
  - `PasswordHasher` port
  - `TokenService` port
  - `PasslibPasswordHasher`
  - `JwtTokenService`
- Implemented idempotent startup super admin seed.
- Implemented auth/admin use cases and routes:
  - `POST /auth/login`
  - `POST /auth/logout`
  - `GET /auth/me`
  - `GET /admin/users`
  - `POST /admin/users`
- Admin user creation requires `super_admin`.
- Public `GET /health` remains unauthenticated.
- Docker API startup now runs `alembic upgrade head` before Uvicorn so startup seed has the required schema.
- Full backend tests pass: 45 passed.
- `ruff check .` passes.
- Docker rebuild/start and real default super admin login/me smoke passed.

## Commit Context

- Last committed Phase 2 unit:
  - `e8f0f71 feat(auth): add http admin sessions`
- Next commit should cover Phase 3 completion docs only.
