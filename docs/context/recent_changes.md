# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest

- Phase 3 implementation has started.
- Auth security ports and infrastructure services were added:
  - `PasswordHasher` port
  - `TokenService` port
  - `PasslibPasswordHasher`
  - `JwtTokenService`
- JWT/cookie settings were added to config and `.env.example`.
- Auth/admin application use cases and idempotent super admin seed support are being added.
- HTTP auth/admin user routes and dependencies are being added.

## Commit Context

- Last committed Phase 2 unit:
  - `cbef860 feat(auth): add admin user use cases`
- Next commit should cover Phase 3 HTTP auth/admin routes and integration tests.
