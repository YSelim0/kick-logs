# Kick Logs Implementation Plan

This document is the implementation source of truth. Each phase is sequential and has a matching task file under `docs/tasks/`.

## Execution Rules

- Work phases in order. Do not start a phase until the previous phase acceptance criteria are met.
- Each phase must stay inside its scope and must not pre-build later phases.
- Backend comes first. Do not implement frontend screens until backend API, database, and listener behavior are working.
- Use small commits after completed units when requested. User pushes manually.
- Keep `docs/context/recent_changes.md` and `docs/context/change_log.md` current after meaningful changes.

## Phase Map

| Phase | Task File | Goal | Depends On |
| --- | --- | --- | --- |
| 1 | `docs/tasks/phase1_tasks.md` | Backend/Docker foundation | Existing docs |
| 2 | `docs/tasks/phase2_tasks.md` | Domain, DB schema, repositories | Phase 1 |
| 3 | `docs/tasks/phase3_tasks.md` | Auth, admin users, super admin seed | Phase 2 |
| 4 | `docs/tasks/phase4_tasks.md` | Channel management, public message search, ingestion use cases | Phase 3 |
| 5 | `docs/tasks/phase5_tasks.md` | Kick listener worker and ingestion runtime | Phase 4 |
| 6 | `docs/tasks/phase6_tasks.md` | Backend verification, Docker readiness, API acceptance | Phase 5 |
| 7 | `docs/tasks/phase7_tasks.md` | Frontend project foundation and API client | Phase 6 |
| 8 | `docs/tasks/phase8_tasks.md` | Public `/search` UI | Phase 7 |
| 9 | `docs/tasks/phase9_tasks.md` | Authenticated `/admin` dashboard | Phase 8 |
| 10 | `docs/tasks/phase10_tasks.md` | Full-stack polish, docs, final smoke checks | Phase 9 |

## Phase 1: Backend/Docker Foundation

Create the minimum runnable backend stack:

- Root dev files and startup docs.
- `apps/api` Python project managed by `uv`.
- FastAPI app with `/health`.
- PostgreSQL and API running through Docker Compose.
- Test/lint commands documented but no business logic yet.

Output must make it possible to run the backend skeleton locally and verify health.

## Phase 2: Domain, Database, Repositories

Add the backend persistence core:

- Domain entities/value objects independent from frameworks.
- SQLAlchemy async session, models, repositories, and unit of work.
- Alembic initial migration for `users`, `channels`, `senders`, and `chat_messages`.
- PostgreSQL `pg_trgm` extension and search/dedup indexes.

Output must make the data model migratable and repository behavior testable.

## Phase 3: Auth And Admin Users

Implement the authenticated admin foundation:

- Password hashing.
- JWT HttpOnly cookie session service.
- Default super admin seed.
- Auth routes: login, logout, me.
- Admin user list/create APIs with role checks.

Output must protect admin routes while keeping public search unauthenticated.

## Phase 4: Channels, Message Search, Ingestion Use Cases

Implement the backend product API surface:

- Admin followed-channel management.
- Kick channel metadata resolver.
- Message normalization and idempotent ingestion use case.
- Public `GET /messages` search with optional filters and cursor pagination.
- Emote token parser.

Output must allow admin-managed channels and public historical search over stored messages.

## Phase 5: Kick Listener Worker

Build the runtime that fills the database:

- Load enabled channels.
- Resolve missing channel/chatroom metadata.
- Connect to Kick Pusher websocket.
- Subscribe to enabled chatrooms.
- Parse `App\Events\ChatMessageEvent`.
- Enrich sender profile image when possible.
- Persist through the existing ingestion use case.
- Add listener Docker service.

Output must run ingestion without duplicating API/business logic.

## Phase 6: Backend Verification And Acceptance

Close the backend MVP before frontend work:

- Run and stabilize backend tests.
- Verify Docker Compose backend services.
- Verify health, auth, admin channels, ingestion, and public search manually or through tests.
- Update README startup and troubleshooting docs.

Output must be a working backend API/listener system that frontend can consume.

## Phase 7: Frontend Foundation

Scaffold frontend only after backend is accepted:

- pnpm workspace.
- Next.js App Router app.
- Tailwind, shadcn/ui base, lucide-react.
- Shared API client and response types.
- Route shells for `/search`, `/login`, `/admin`, and reserved `/`.

Output must compile and call backend health without implementing final screens.

## Phase 8: Public Search UI

Implement the public search experience from `docs/design/design.md` and `docs/design/design.pen`:

- `/search` public route with no login requirement.
- Search filters mapped to backend query params.
- Default date range set to the last 7 days in the UI.
- Infinite-scroll message rows.
- Circular avatars.
- Inline emote rendering.
- Dense results inside one outer list container.

Output must let any visitor search historical messages.

## Phase 9: Admin Dashboard UI

Implement authenticated backend management:

- Login flow.
- `/admin` guarded route.
- Followed channel list/add/remove.
- Super admin user creation.
- Clear handling of auth errors and role restrictions.

Output must let admins manage backend operational state.

## Phase 10: Full-Stack Polish And Final Smoke

Finish MVP readiness:

- Full `docker compose up --build` with postgres, api, listener, and web.
- End-to-end smoke path from admin channel add to public search.
- README final startup instructions.
- Context/docs cleanup.

Output must be a locally runnable MVP ready for manual push.
