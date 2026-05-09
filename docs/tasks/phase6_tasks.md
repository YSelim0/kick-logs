# Phase 6 Tasks: Backend Verification And Acceptance

## Scope

Stabilize and verify the completed backend before frontend work begins. This phase owns tests, Docker backend acceptance, README/backend instructions, and known-risk documentation.

Do not add frontend implementation, change UI design, or introduce new product features beyond backend fixes needed to pass acceptance.

## Inputs

- Completed Phases 1-5.
- All backend routes, listener runtime, migrations, and Docker services.

## Tasks

- [ ] Test suite stabilization:
  - [ ] Run backend unit and integration tests.
  - [ ] Fix failing tests without expanding scope.
  - [ ] Ensure tests cover health, auth, admin users, admin channels, search, ingestion, listener parser, and reconnect behavior.
- [ ] Docker backend verification:
  - [ ] Run `docker compose up --build postgres api listener`.
  - [ ] Confirm API connects to PostgreSQL.
  - [ ] Confirm migrations can run in Docker workflow.
  - [ ] Confirm listener starts and logs useful status.
- [ ] Manual/API smoke checks:
  - [ ] `GET /health`.
  - [ ] Default super admin login.
  - [ ] `GET /auth/me` with cookie.
  - [ ] Add followed channel.
  - [ ] Disable/remove followed channel.
  - [ ] Ingest sample message through use case or listener test harness.
  - [ ] Public `GET /messages` without auth.
- [ ] Backend docs:
  - [ ] Update `README.md` with backend startup and verification steps.
  - [ ] Document env vars and local secrets expectations.
  - [ ] Document known Kick endpoint/websocket fragility.
  - [ ] Document public `/messages` access and admin-only `/admin/*` access.
- [ ] Cleanup:
  - [ ] Remove unused scaffold files.
  - [ ] Confirm no `.env`, virtualenvs, caches, logs, or dependency folders are tracked.

## Acceptance Criteria

- [ ] Backend test suite passes.
- [ ] Docker backend stack starts cleanly.
- [ ] Public search API works without login.
- [ ] Admin APIs require login.
- [ ] Listener is wired through Docker and ingestion use cases.
- [ ] README/backend docs are enough for frontend implementation to begin.

## Handoff

Phase 7 can scaffold frontend against a verified backend API and must not revisit backend architecture unless a real API contract bug is found.
