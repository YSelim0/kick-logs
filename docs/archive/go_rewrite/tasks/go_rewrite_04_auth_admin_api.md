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

- [x] Implement password hashing and verification compatible with stored bcrypt hashes.
- [x] Implement JWT creation and verification.
- [x] Preserve auth cookie name, path, HttpOnly flag, secure flag, same-site value, and expiry.
- [x] Implement `POST /auth/login`.
- [x] Implement `POST /auth/logout`.
- [x] Implement `GET /auth/me`.
- [x] Implement admin auth middleware.
- [x] Implement super admin authorization checks.
- [x] Implement `GET /admin/users`.
- [x] Implement `POST /admin/users`.
- [x] Implement Kick channel resolver for admin channel add.
- [x] Implement `GET /admin/channels`.
- [x] Implement `POST /admin/channels`.
- [x] Implement `DELETE /admin/channels/{channel_id}` as disable behavior.
- [x] Implement basic `GET /admin/operations/summary` fields backed by available SQLite and
      ClickHouse counts.
- [x] Keep CORS behavior compatible with local frontend dev.

## Tests And Checks

- [x] Login succeeds with seeded default super admin.
- [x] Login fails with invalid password.
- [x] `GET /auth/me` returns current user when authenticated.
- [x] Logout clears the cookie.
- [x] Admin routes reject unauthenticated users.
- [x] Admin user creation requires super admin.
- [x] Channel add resolves Kick metadata and persists followed channel data.
- [x] Deleting a channel disables it without deleting historical messages.
- [x] Contract fixtures for auth/admin routes match existing frontend expectations.

## Acceptance Criteria

- [x] The frontend login and admin channel/user workflows can call the Go API without response
      shape changes.
- [x] Default super admin behavior remains available and env-overridable.
- [x] Admin operations summary returns a compatible response, even if some counters are zero before
      message ingestion exists.

## Commit Boundary

Commit auth/admin parity separately from message and listener work.
