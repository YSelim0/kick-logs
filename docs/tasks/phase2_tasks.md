# Phase 2 Tasks: Domain, Database, Repositories

## Scope

Add domain models, async database infrastructure, Alembic migration, repositories, and unit of work. This phase owns persistence foundation only.

Do not implement HTTP auth, admin endpoints, Kick listener websocket code, frontend, or real API business routes beyond what is needed to test persistence.

## Inputs

- Completed Phase 1 backend scaffold.
- Core table list from `docs/architecture.md`.
- Search/data retention decisions from `docs/project_plan.md`.

## Tasks

- [x] Domain layer:
  - [x] Add `User`, `Channel`, `Sender`, `ChatMessage`, and `Emote` entities.
  - [x] Add value objects/enums for roles, search filters, and cursor pagination.
  - [x] Keep domain free of FastAPI, SQLAlchemy, Pydantic, HTTP clients, and websocket imports.
- [x] Database infrastructure:
  - [x] Configure async SQLAlchemy engine/session.
  - [x] Add SQLAlchemy models for `users`, `channels`, `senders`, and `chat_messages`.
  - [x] Use PostgreSQL `JSONB` for raw payloads, badges, emotes, reply metadata, and raw profile payloads.
  - [x] Use timezone-aware timestamps.
- [x] Alembic:
  - [x] Configure Alembic for async SQLAlchemy metadata.
  - [x] Create initial migration.
  - [x] Enable `pg_trgm`.
  - [x] Add indexes for message timestamp, Kick message id, channel slug, sender username/slug, and text search.
  - [x] Add unique constraints for Kick ids where required to deduplicate messages.
- [x] Repositories and unit of work:
  - [x] Define application ports for user, channel, sender, and message repositories.
  - [x] Implement SQLAlchemy repositories.
  - [x] Implement async unit of work with commit/rollback boundaries.
- [x] Tests:
  - [x] Unit tests for domain/value object behavior.
  - [x] Repository tests for create/read/update flows.
  - [x] Migration metadata smoke test.

## Acceptance Criteria

- [x] Alembic migration applies cleanly to local PostgreSQL.
- [x] Repository tests pass against a test database or isolated transaction strategy.
- [x] Domain imports remain framework-independent.
- [x] No API auth/search/listener behavior is implemented yet.
- [ ] Docs/context are updated with the actual DB details.

## Handoff

Phase 3 can assume database tables, repositories, and unit of work exist and are safe to use for auth/admin user flows.
