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

## Commit Context

- Last committed Phase 2 unit:
  - `85dc302 feat(docs): complete phase two persistence`
- Next commit should cover Phase 3 security services and tests.
