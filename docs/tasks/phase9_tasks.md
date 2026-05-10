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
- [ ] Admin dashboard layout:
  - [ ] Serious, dense, dark-only operations UI.
  - [ ] No landing-page hero sections.
  - [ ] Clear distinction between channel management and user management.
- [ ] Followed channel management:
  - [ ] List followed channels with enabled state and Kick metadata.
  - [ ] Add channel by slug/nickname.
  - [ ] Show resolver/loading/error state.
  - [ ] Remove/disable channel.
  - [ ] Refresh list after mutations.
- [ ] Admin user management:
  - [ ] Show only when current user role is `super_admin`.
  - [ ] List admin users.
  - [ ] Create new admin user.
  - [ ] Do not expose password hashes or secrets.
- [ ] Tests/checks:
  - [x] Login success/failure.
  - [x] Admin route guard.
  - [ ] Channel add/remove flow with mocked API.
  - [ ] Super admin-only user creation visibility.
  - [ ] Public search still loads without auth.

## Acceptance Criteria

- [ ] Admin can log in and manage followed channels.
- [ ] Super admin can create admin users.
- [ ] Non-authenticated users cannot access `/admin`.
- [ ] `/search` remains public.
- [ ] Frontend typecheck/build passes.

## Handoff

Phase 10 can perform full-stack smoke testing and final MVP cleanup.
