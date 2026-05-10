# Phase 3 Tasks: Auth And Admin Users

## Scope

Implement authentication and admin user management. This phase owns login/session behavior, default super admin seed, role checks, and admin user APIs.

Do not implement followed channel management, message search, Kick ingestion, listener runtime, or frontend screens.

## Inputs

- Completed Phase 2 database/repository foundation.
- Default credentials decision: `admin@kicklogs.local` / `admin123`, overridable by env.
- Auth contract from `docs/architecture.md`.

## Tasks

- [x] Security services:
  - [x] Password hasher port and Passlib-based implementation.
  - [x] JWT token service port and signed JWT implementation.
  - [x] HttpOnly cookie configuration from settings.
- [x] Super admin seed:
  - [x] Seed default super admin at startup or explicit seed command.
  - [x] Use env overrides `DEFAULT_SUPER_ADMIN_EMAIL` and `DEFAULT_SUPER_ADMIN_PASSWORD`.
  - [x] Store password hash only.
  - [x] Keep seed idempotent.
- [x] Use cases:
  - [x] Login.
  - [x] Get current user.
  - [x] List admin users.
  - [x] Create admin user.
  - [x] Enforce `super_admin` required for admin user creation.
- [x] HTTP routes:
  - [x] `POST /auth/login`
  - [x] `POST /auth/logout`
  - [x] `GET /auth/me`
  - [x] `GET /admin/users`
  - [x] `POST /admin/users`
- [x] Dependencies/schemas:
  - [x] Auth dependency that loads current user from cookie.
  - [x] Role dependency for admin and super admin.
  - [x] Pydantic request/response schemas.
- [x] Tests:
  - [x] Login succeeds for seeded super admin.
  - [x] Invalid login fails safely.
  - [x] `/auth/me` works with cookie.
  - [x] Admin routes reject unauthenticated requests.
  - [x] Admin user creation requires `super_admin`.

## Acceptance Criteria

- [x] Default super admin is available after startup/seed.
- [x] Admin user APIs are protected.
- [x] Public routes are not accidentally protected.
- [x] Tests for auth and role checks pass.
- [x] Docs/context are updated with implemented auth behavior.

Verification note:

- `uv run pytest` passes with 45 tests.
- `uv run ruff check .` passes.
- `alembic current` reports `20260510_0001 (head)`.
- `docker compose up --build -d postgres api` succeeds.
- Real API smoke passed with default credentials:
  - `POST /auth/login` using `admin@kicklogs.local` / `admin123`
  - `GET /auth/me` using the returned HttpOnly session cookie

## Handoff

Phase 4 can use authenticated admin dependencies for channel management and can rely on public routes remaining unauthenticated.
