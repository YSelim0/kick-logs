# Go Rewrite Phase 4: Auth And Admin API

## Scope

Rebuild auth, admin users, followed channel management, and basic operations summary in Go.

This phase should make the admin shell usable against the Go API for login and channel management.

## Out Of Scope

- Do not implement message search/export yet.
- Do not implement analytics/profile endpoints yet.
- Do not implement data migration yet.
- Do not remove Python endpoints.

## Checklist

- [ ] Implement password hashing and verification compatible with stored bcrypt hashes.
- [ ] Implement JWT creation and verification.
- [ ] Preserve auth cookie name, path, HttpOnly flag, secure flag, same-site value, and expiry.
- [ ] Implement `POST /auth/login`.
- [ ] Implement `POST /auth/logout`.
- [ ] Implement `GET /auth/me`.
- [ ] Implement admin auth middleware.
- [ ] Implement super admin authorization checks.
- [ ] Implement `GET /admin/users`.
- [ ] Implement `POST /admin/users`.
- [ ] Implement Kick channel resolver for admin channel add.
- [ ] Implement `GET /admin/channels`.
- [ ] Implement `POST /admin/channels`.
- [ ] Implement `DELETE /admin/channels/{channel_id}` as disable behavior.
- [ ] Implement basic `GET /admin/operations/summary` fields backed by available SQLite and
      ClickHouse counts.
- [ ] Keep CORS behavior compatible with local frontend dev.

## Tests And Checks

- [ ] Login succeeds with seeded default super admin.
- [ ] Login fails with invalid password.
- [ ] `GET /auth/me` returns current user when authenticated.
- [ ] Logout clears the cookie.
- [ ] Admin routes reject unauthenticated users.
- [ ] Admin user creation requires super admin.
- [ ] Channel add resolves Kick metadata and persists followed channel data.
- [ ] Deleting a channel disables it without deleting historical messages.
- [ ] Contract fixtures for auth/admin routes match existing frontend expectations.

## Acceptance Criteria

- [ ] The frontend login and admin channel/user workflows can call the Go API without response
      shape changes.
- [ ] Default super admin behavior remains available and env-overridable.
- [ ] Admin operations summary returns a compatible response, even if some counters are zero before
      message ingestion exists.

## Commit Boundary

Commit auth/admin parity separately from message and listener work.
