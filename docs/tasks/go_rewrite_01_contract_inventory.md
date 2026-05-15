# Go Rewrite Phase 1: Contract Inventory

## Scope

Capture the current Python backend API contract before writing replacement Go behavior.

This phase owns documentation and test fixtures for API compatibility only.

## Out Of Scope

- Do not create the Go backend workspace yet.
- Do not change Docker runtime.
- Do not change frontend behavior.
- Do not change current Python route behavior.

## Checklist

- [ ] Record all public and admin endpoint paths, methods, and access rules.
- [ ] Capture request body shapes for login, admin channel creation, admin user creation,
      retention settings, cleanup preview, and cleanup confirm.
- [ ] Capture query parameters for messages, export, analytics, user profile, and channel profile
      endpoints.
- [ ] Capture response body shapes for every endpoint used by the frontend.
- [ ] Capture expected auth cookie name, path, HttpOnly flag, secure flag, same-site behavior, and
      expiry behavior.
- [ ] Capture practical status-code expectations for success, unauthenticated, unauthorized,
      validation failure, not found, and conflict cases.
- [ ] Create fixture JSON files for representative successful responses.
- [ ] Create fixture JSON files or notes for frontend-sensitive error responses.
- [ ] Document the current cursor format for message pagination.
- [ ] Document CSV export column order and JSON export shape.
- [ ] Document current date parsing expectations for `start` and `end`.
- [ ] Document sender exact-match behavior and channel/content matching behavior.
- [ ] Document reply and emote response fields, including `reply_metadata`, `thread_parent_id`,
      and emote image data.

## Tests And Checks

- [ ] Run current backend tests before capturing fixtures if the local environment supports it.
- [ ] Verify captured endpoint list against route files under `apps/api/src`.
- [ ] Verify frontend API wrappers under `apps/web/src` are represented in the contract notes.

## Acceptance Criteria

- [ ] A future Go implementer can build route handlers without reading Python route files for basic
      request/response shapes.
- [ ] The contract inventory clearly identifies which behavior is strict compatibility and which
      behavior is best-effort compatibility.
- [ ] No active application code is changed in this phase.

## Commit Boundary

Commit this phase as a documentation and fixture-only unit.
