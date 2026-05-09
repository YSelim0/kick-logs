# Phase 2 Tasks: Domain, Database, Repositories

## Scope

Add domain models, async database infrastructure, Alembic migration, repositories, and unit of work. This phase owns persistence foundation only.

Do not implement HTTP auth, admin endpoints, Kick listener websocket code, frontend, or real API business routes beyond what is needed to test persistence.

## Inputs

- Completed Phase 1 backend scaffold.
- Core table list from `docs/architecture.md`.
- Search/data retention decisions from `docs/project_plan.md`.

## Tasks

- [ ] Domain layer:
  - [ ] Add `User`, `Channel`, `Sender`, `ChatMessage`, and `Emote` entities.
  - [ ] Add value objects/enums for roles, search filters, and cursor pagination.
  - [ ] Keep domain free of FastAPI, SQLAlchemy, Pydantic, HTTP clients, and websocket imports.
- [ ] Database infrastructure:
  - [ ] Configure async SQLAlchemy engine/session.
  - [ ] Add SQLAlchemy models for `users`, `channels`, `senders`, and `chat_messages`.
  - [ ] Use PostgreSQL `JSONB` for raw payloads, badges, emotes, reply metadata, and raw profile payloads.
  - [ ] Use timezone-aware timestamps.
- [ ] Alembic:
  - [ ] Configure Alembic for async SQLAlchemy metadata.
  - [ ] Create initial migration.
  - [ ] Enable `pg_trgm`.
  - [ ] Add indexes for message timestamp, Kick message id, channel slug, sender username/slug, and text search.
  - [ ] Add unique constraints for Kick ids where required to deduplicate messages.
- [ ] Repositories and unit of work:
  - [ ] Define application ports for user, channel, sender, and message repositories.
  - [ ] Implement SQLAlchemy repositories.
  - [ ] Implement async unit of work with commit/rollback boundaries.
- [ ] Tests:
  - [ ] Unit tests for domain/value object behavior.
  - [ ] Repository tests for create/read/update flows.
  - [ ] Migration metadata smoke test.

## Acceptance Criteria

- [ ] Alembic migration applies cleanly to local PostgreSQL.
- [ ] Repository tests pass against a test database or isolated transaction strategy.
- [ ] Domain imports remain framework-independent.
- [ ] No API auth/search/listener behavior is implemented yet.
- [ ] Docs/context are updated with the actual DB details.

## Handoff

Phase 3 can assume database tables, repositories, and unit of work exist and are safe to use for auth/admin user flows.
