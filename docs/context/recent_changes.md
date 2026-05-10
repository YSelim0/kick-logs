# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest

- Phase 9 admin dashboard UI is complete.
- Latest additions:
  - super-admin-only `UserAdmin` section in `/admin`
  - `GET /admin/users` user list
  - `POST /admin/users` create-admin form
  - secret-safe user rows showing only email, role, and active state
  - dashboard test coverage for hiding user management from regular admins
- Verification:
  - `pnpm --filter @kick-logs/web test`: 6 files, 20 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
  - `docker compose up --build -d web`: passed
  - `GET http://localhost:3000/search`: HTTP 200 without login
  - `GET http://localhost:3000/login`: HTTP 200
  - `GET http://localhost:3000/admin`: HTTP 200
  - `GET http://localhost:8000/health`: `{"status":"ok"}`
- Phase 10 is next: full-stack polish, final README cleanup, and end-to-end smoke path.

## Commit Context

- Previous committed units:
  - `3c9178b feat(backend): complete phase six acceptance`
  - `c8c5eb9 feat(web): scaffold frontend foundation`
  - `f2250a9 feat(docs): complete phase seven foundation`
  - `619f4f9 feat(search): add public message search ui`
  - `2ab7c91 feat(search): default date range filters`
  - `813d713 feat(auth): add admin login guard`
  - `823a8ee feat(admin): add channel management ui`
- Latest completed unit:
  - Phase 9 super-admin user management UI
- Commit message for this unit:
  - `feat(admin): add user management ui`
