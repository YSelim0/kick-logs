# Post-MVP Feature 8 Tasks: Final Smoke And Documentation

## Goal

Verify the completed post-MVP feature set and update public/operator
documentation.

## Scope

This feature owns verification and docs only. Do not introduce new product
features here unless fixing a bug found during smoke.

## Verification Tasks

- [ ] Run backend formatting, lint, and test suite.
- [ ] Run frontend formatting, tests, typecheck, lint, and build.
- [ ] Run Docker Compose full stack.
- [ ] Smoke admin operations dashboard.
- [ ] Smoke improved search filters, highlights, presets, and export.
- [ ] Smoke landing page analytics.
- [ ] Smoke user profile page.
- [ ] Smoke channel profile page.
- [ ] Smoke admin data management dry-run flow.
- [ ] Verify public routes remain unauthenticated and admin routes remain authenticated.

## Documentation Tasks

- [ ] Update README feature list, route list, startup notes, and operations docs.
- [ ] Update `docs/design/design.md` if UI decisions changed during implementation.
- [ ] Update architecture docs if new persistence or API patterns were added.
- [ ] Update living context and recent changes with final state.
- [ ] Ensure archived MVP docs remain clearly marked as historical.

## Acceptance

- [ ] All required checks pass.
- [ ] Docker Compose starts the full system.
- [ ] A fresh reader can understand the current feature set without reading archived MVP task
      files.
- [ ] Working tree is ready for the user's manual commit/push workflow.
