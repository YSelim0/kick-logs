# Phase 9 Tasks: Authenticated Admin Dashboard

## Scope

Implement `/login` and `/admin` for backend operational management. This phase owns admin authentication UI, route guarding, followed-channel management, and super admin user creation.

Do not redesign public `/search`, implement landing page content, or change backend contracts unless a documented backend bug blocks the admin UI.

## Inputs

- Completed public search UI from Phase 8.
- Backend auth/admin/channel APIs.
- Admin requirements from `docs/project_plan.md`.

## Tasks

- [x] Login UI:
  - [x] Email/password form.
  - [x] Submit to `POST /auth/login`.
  - [x] Store session through HttpOnly cookie returned by backend.
  - [x] Show compact error state.
  - [x] Redirect authenticated user to `/admin`.
- [x] Auth state:
  - [x] Use `GET /auth/me` to load current user.
  - [x] Guard `/admin`; unauthenticated users go to `/login`.
  - [x] Keep `/search` public and unaffected.
  - [x] Add logout action using `POST /auth/logout`.
- [x] Admin dashboard layout:
  - [x] Serious, dense, dark-only operations UI.
  - [x] No landing-page hero sections.
  - [x] Clear distinction between channel management and user management.
- [x] Followed channel management:
  - [x] List followed channels with enabled state and Kick metadata.
  - [x] Add channel by slug/nickname.
  - [x] Show resolver/loading/error state.
  - [x] Remove/disable channel.
  - [x] Refresh list after mutations.
- [x] Admin user management:
  - [x] Show only when current user role is `super_admin`.
  - [x] List admin users.
  - [x] Create new admin user.
  - [x] Do not expose password hashes or secrets.
- [x] Tests/checks:
  - [x] Login success/failure.
  - [x] Admin route guard.
  - [x] Channel add/remove flow with mocked API.
  - [x] Super admin-only user creation visibility.
  - [x] Public search still loads without auth.

## Acceptance Criteria

- [x] Admin can log in and manage followed channels.
- [x] Super admin can create admin users.
- [x] Non-authenticated users cannot access `/admin`.
- [x] `/search` remains public.
- [x] Frontend typecheck/build passes.

## Handoff

Phase 10 can perform full-stack smoke testing and final MVP cleanup.
