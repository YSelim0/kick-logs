# Phase 10 Tasks: Full-Stack Polish And Final Smoke

## Scope

Finish local MVP readiness. This phase owns full Docker Compose verification, end-to-end smoke paths, final documentation, and cleanup.

Do not add new product features, redesign UI, or change backend/frontend contracts unless needed to fix acceptance failures.

## Inputs

- Completed Phases 1-9.
- Full Docker services: `postgres`, `api`, `listener`, `web`.

## Tasks

- [x] Full Docker Compose:
  - [x] `docker compose up --build` starts all services.
  - [x] API health route passes from host.
  - [x] Web route loads from host.
  - [x] Listener starts and logs channel subscription status.
  - [x] PostgreSQL volume persists data across restart.
- [x] End-to-end MVP smoke:
  - [x] Seed or log in as default super admin.
  - [x] Add followed channel in `/admin`.
  - [x] Confirm backend stores channel metadata.
  - [x] Ingest a sample or live message.
  - [x] Find message through public `/search`.
  - [x] Confirm `/search` works without auth.
- [x] Documentation:
  - [x] Finalize `README.md` with local setup, env, Docker, API, listener, frontend, and test commands.
  - [x] Document default super admin credentials and env overrides.
  - [x] Document manual push workflow.
  - [x] Update context files with final MVP state.
- [x] Cleanup:
  - [x] Remove unused scaffold files.
  - [x] Ensure no generated caches/logs/dependency folders are tracked.
  - [x] Ensure `.env` and secrets are untracked.
  - [x] Run final status check before commit.
- [x] Final checks:
  - [x] Backend tests pass.
  - [x] Frontend typecheck/build passes.
  - [x] Docker smoke passes or documented blocker exists.

## Acceptance Criteria

- [x] Project can be started locally with documented commands.
- [x] Public search and authenticated admin flows both work.
- [x] Listener can ingest messages into searchable storage.
- [x] Docs reflect actual startup and operational behavior.
- [x] Repo is ready for user-managed push.

## Handoff

This phase ends the MVP implementation plan. Any work after this should be planned as a separate post-MVP enhancement.
