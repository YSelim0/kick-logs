# Post-MVP Feature 3 Tasks: Analytics Foundation

## Goal

Create shared analytics APIs and backend query infrastructure for landing,
user profile, and channel profile screens.

## Scope

This feature owns backend analytics contracts and tests. Build only minimal API
surface validation; do not build final landing/profile pages here.

## Backend Tasks

- [x] Add analytics application DTOs and use cases for overview, message volume, top senders,
      top channels, and top emotes.
- [x] Add repository query methods for aggregate counts over `chat_messages`.
- [x] Support date range filters where relevant.
- [x] Support optional channel scope for volume, top senders, and top emotes.
- [x] Support optional sender scope for volume, top channels, and top emotes.
- [x] Expose public read-only analytics routes.
- [x] Keep route responses compact enough for dashboard/profile use.
- [x] Add backend tests for aggregate correctness, empty datasets, date range filtering,
      channel scope, sender scope, and limit handling.

## Frontend Foundation Tasks

- [x] Add typed analytics API wrappers and response types.
- [x] Add tests for API parameter mapping where practical.

## Docs And Acceptance

- [x] Document analytics API shape in README or docs.
- [x] Update context docs.
- [x] Verify backend tests and frontend typecheck/lint if frontend types are touched.
- [x] Acceptance: landing, user profile, and channel profile features can consume analytics
      data without inventing new backend contracts.
