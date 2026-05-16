# Post-MVP Feature 8 Tasks: Final Smoke And Documentation

## Goal

Verify the completed post-MVP feature set and update public/operator
documentation.

## Scope

This feature owns verification and docs only. Do not introduce new product
features here unless fixing a bug found during smoke.

## Verification Tasks

- [x] Run backend formatting, lint, and test suite.
- [x] Run frontend formatting, tests, typecheck, lint, and build.
- [x] Run Docker Compose full stack.
- [x] Smoke admin operations dashboard.
- [x] Smoke improved search filters, highlights, presets, and export.
- [x] Smoke landing page analytics.
- [x] Smoke user profile page.
- [x] Smoke channel profile page.
- [x] Smoke admin data management dry-run flow.
- [x] Verify public routes remain unauthenticated and admin routes remain authenticated.

## Documentation Tasks

- [x] Update README feature list, route list, startup notes, and operations docs.
- [x] Update `docs/design/design.md` if UI decisions changed during implementation.
- [x] Update architecture docs if new persistence or API patterns were added.
- [x] Update living context and recent changes with final state.
- [x] Ensure archived MVP docs remain clearly marked as historical.

## Acceptance

- [x] All required checks pass.
- [x] Docker Compose starts the full system.
- [x] A fresh reader can understand the current feature set without reading archived MVP task
      files.
- [x] Working tree is ready for the user's manual commit/push workflow.
