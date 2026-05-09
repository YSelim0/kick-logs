# Phase 3 Tasks: Auth And Admin Users

## Scope

Implement authentication and admin user management. This phase owns login/session behavior, default super admin seed, role checks, and admin user APIs.

Do not implement followed channel management, message search, Kick ingestion, listener runtime, or frontend screens.

## Inputs

- Completed Phase 2 database/repository foundation.
- Default credentials decision: `admin@kicklogs.local` / `admin123`, overridable by env.
- Auth contract from `docs/architecture.md`.

## Tasks

- [ ] Security services:
  - [ ] Password hasher port and Passlib-based implementation.
  - [ ] JWT token service port and signed JWT implementation.
  - [ ] HttpOnly cookie configuration from settings.
- [ ] Super admin seed:
  - [ ] Seed default super admin at startup or explicit seed command.
  - [ ] Use env overrides `DEFAULT_SUPER_ADMIN_EMAIL` and `DEFAULT_SUPER_ADMIN_PASSWORD`.
  - [ ] Store password hash only.
  - [ ] Keep seed idempotent.
- [ ] Use cases:
  - [ ] Login.
  - [ ] Get current user.
  - [ ] List admin users.
  - [ ] Create admin user.
  - [ ] Enforce `super_admin` required for admin user creation.
- [ ] HTTP routes:
  - [ ] `POST /auth/login`
  - [ ] `POST /auth/logout`
  - [ ] `GET /auth/me`
  - [ ] `GET /admin/users`
  - [ ] `POST /admin/users`
- [ ] Dependencies/schemas:
  - [ ] Auth dependency that loads current user from cookie.
  - [ ] Role dependency for admin and super admin.
  - [ ] Pydantic request/response schemas.
- [ ] Tests:
  - [ ] Login succeeds for seeded super admin.
  - [ ] Invalid login fails safely.
  - [ ] `/auth/me` works with cookie.
  - [ ] Admin routes reject unauthenticated requests.
  - [ ] Admin user creation requires `super_admin`.

## Acceptance Criteria

- [ ] Default super admin is available after startup/seed.
- [ ] Admin user APIs are protected.
- [ ] Public routes are not accidentally protected.
- [ ] Tests for auth and role checks pass.
- [ ] Docs/context are updated with implemented auth behavior.

## Handoff

Phase 4 can use authenticated admin dependencies for channel management and can rely on public routes remaining unauthenticated.
