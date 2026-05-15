# Post-MVP Feature 7 Tasks: Data Management

## Goal

Give admins controlled tools to understand and manage data growth without
accidental destructive actions.

## Scope

This feature owns admin-only retention and cleanup. Do not build public
analytics/profile pages here.

## Backend Tasks

- [x] Add retention settings persistence for messages and raw events, defaulting to keep
      forever until an admin changes it.
- [x] Add admin data summary endpoint with database size, key table sizes, row counts, and
      current retention settings.
- [x] Add dry-run cleanup endpoint that previews affected rows for old raw events, old
      messages, a specific channel, or a specific sender.
- [x] Add confirmed cleanup endpoint that requires explicit confirmation input before deleting
      data.
- [x] Keep destructive cleanup admin-only.
- [x] Add backend tests for permissions, settings defaults, settings updates, dry-run counts,
      confirmed cleanup, and refusal without confirmation.

## Frontend Tasks

- [x] Add admin data management section.
- [x] Show database/table sizes and current retention settings.
- [x] Add retention controls for forever, 30 days, and 90 days.
- [x] Add dry-run preview before cleanup.
- [x] Add explicit confirmation UI for destructive cleanup.
- [x] Add success/error states that clearly show what was deleted.
- [x] Add frontend tests for settings display, dry-run preview, blocked deletion without
      confirmation, confirmed deletion, and API errors.

## Docs And Acceptance

- [ ] Add backup/restore guidance for Docker Compose PostgreSQL.
- [ ] Add cleanup/retention operational notes to README.
- [ ] Update context docs.
- [ ] Verify backend tests and frontend tests/typecheck/lint/build.
- [ ] Acceptance: admins can preview and run cleanup intentionally, and no destructive action
      can run from a single accidental click.
