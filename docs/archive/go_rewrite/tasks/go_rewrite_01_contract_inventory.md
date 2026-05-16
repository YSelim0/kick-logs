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

- [x] Record all public and admin endpoint paths, methods, and access rules.
- [x] Capture request body shapes for login, admin channel creation, admin user creation,
      retention settings, cleanup preview, and cleanup confirm.
- [x] Capture query parameters for messages, export, analytics, user profile, and channel profile
      endpoints.
- [x] Capture response body shapes for every endpoint used by the frontend.
- [x] Capture expected auth cookie name, path, HttpOnly flag, secure flag, same-site behavior, and
      expiry behavior.
- [x] Capture practical status-code expectations for success, unauthenticated, unauthorized,
      validation failure, not found, and conflict cases.
- [x] Create fixture JSON files for representative successful responses.
- [x] Create fixture JSON files or notes for frontend-sensitive error responses.
- [x] Document the current cursor format for message pagination.
- [x] Document CSV export column order and JSON export shape.
- [x] Document current date parsing expectations for `start` and `end`.
- [x] Document sender exact-match behavior and channel/content matching behavior.
- [x] Document reply and emote response fields, including `reply_metadata`, `thread_parent_id`,
      and emote image data.

## Tests And Checks

- [x] Run current backend tests before capturing fixtures if the local environment supports it.
- [x] Verify captured endpoint list against route files under `apps/api/src`.
- [x] Verify frontend API wrappers under `apps/web/src` are represented in the contract notes.

## Acceptance Criteria

- [x] A future Go implementer can build route handlers without reading Python route files for basic
      request/response shapes.
- [x] The contract inventory clearly identifies which behavior is strict compatibility and which
      behavior is best-effort compatibility.
- [x] No active application code is changed in this phase.

## Commit Boundary

Commit this phase as a documentation and fixture-only unit.
