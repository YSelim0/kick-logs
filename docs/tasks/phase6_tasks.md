# Phase 6 Tasks: Backend Verification And Acceptance

## Scope

Stabilize and verify the completed backend before frontend work begins. This phase owns tests, Docker backend acceptance, README/backend instructions, and known-risk documentation.

Do not add frontend implementation, change UI design, or introduce new product features beyond backend fixes needed to pass acceptance.

## Inputs

- Completed Phases 1-5.
- All backend routes, listener runtime, migrations, and Docker services.

## Tasks

- [x] Test suite stabilization:
  - [x] Run backend unit and integration tests.
  - [x] Fix failing tests without expanding scope.
  - [x] Ensure tests cover health, auth, admin users, admin channels, search, ingestion, listener parser, and reconnect behavior.
- [x] Docker backend verification:
  - [x] Run `docker compose up --build postgres api listener`.
  - [x] Confirm API connects to PostgreSQL.
  - [x] Confirm migrations can run in Docker workflow.
  - [x] Confirm listener starts and logs useful status.
- [x] Manual/API smoke checks:
  - [x] `GET /health`.
  - [x] Default super admin login.
  - [x] `GET /auth/me` with cookie.
  - [x] Add followed channel.
  - [x] Disable/remove followed channel.
  - [x] Ingest sample message through use case or listener test harness.
  - [x] Public `GET /messages` without auth.
- [x] Backend docs:
  - [x] Update `README.md` with backend startup and verification steps.
  - [x] Document env vars and local secrets expectations.
  - [x] Document known Kick endpoint/websocket fragility.
  - [x] Document public `/messages` access and admin-only `/admin/*` access.
- [x] Cleanup:
  - [x] Remove unused scaffold files.
  - [x] Confirm no `.env`, virtualenvs, caches, logs, or dependency folders are tracked.

## Acceptance Criteria

- [x] Backend test suite passes.
- [x] Docker backend stack starts cleanly.
- [x] Public search API works without login.
- [x] Admin APIs require login.
- [x] Listener is wired through Docker and ingestion use cases.
- [x] README/backend docs are enough for frontend implementation to begin.

## Handoff

Phase 7 can scaffold frontend against a verified backend API and must not revisit backend architecture unless a real API contract bug is found.
