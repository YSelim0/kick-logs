# Phase 10 Tasks: Full-Stack Polish And Final Smoke

## Scope

Finish local MVP readiness. This phase owns full Docker Compose verification, end-to-end smoke paths, final documentation, and cleanup.

Do not add new product features, redesign UI, or change backend/frontend contracts unless needed to fix acceptance failures.

## Inputs

- Completed Phases 1-9.
- Full Docker services: `postgres`, `api`, `listener`, `web`.

## Tasks

- [ ] Full Docker Compose:
  - [ ] `docker compose up --build` starts all services.
  - [ ] API health route passes from host.
  - [ ] Web route loads from host.
  - [ ] Listener starts and logs channel subscription status.
  - [ ] PostgreSQL volume persists data across restart.
- [ ] End-to-end MVP smoke:
  - [ ] Seed or log in as default super admin.
  - [ ] Add followed channel in `/admin`.
  - [ ] Confirm backend stores channel metadata.
  - [ ] Ingest a sample or live message.
  - [ ] Find message through public `/search`.
  - [ ] Confirm `/search` works without auth.
- [ ] Documentation:
  - [ ] Finalize `README.md` with local setup, env, Docker, API, listener, frontend, and test commands.
  - [ ] Document default super admin credentials and env overrides.
  - [ ] Document manual push workflow.
  - [ ] Update context files with final MVP state.
- [ ] Cleanup:
  - [ ] Remove unused scaffold files.
  - [ ] Ensure no generated caches/logs/dependency folders are tracked.
  - [ ] Ensure `.env` and secrets are untracked.
  - [ ] Run final status check before commit.
- [ ] Final checks:
  - [ ] Backend tests pass.
  - [ ] Frontend typecheck/build passes.
  - [ ] Docker smoke passes or documented blocker exists.

## Acceptance Criteria

- [ ] Project can be started locally with documented commands.
- [ ] Public search and authenticated admin flows both work.
- [ ] Listener can ingest messages into searchable storage.
- [ ] Docs reflect actual startup and operational behavior.
- [ ] Repo is ready for user-managed push.

## Handoff

This phase ends the MVP implementation plan. Any work after this should be planned as a separate post-MVP enhancement.
