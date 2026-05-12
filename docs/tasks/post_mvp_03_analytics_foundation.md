# Post-MVP Feature 3 Tasks: Analytics Foundation

## Goal

Create shared analytics APIs and backend query infrastructure for landing,
user profile, and channel profile screens.

## Scope

This feature owns backend analytics contracts and tests. Build only minimal API
surface validation; do not build final landing/profile pages here.

## Backend Tasks

- [ ] Add analytics application DTOs and use cases for overview, message volume, top senders,
      top channels, and top emotes.
- [ ] Add repository query methods for aggregate counts over `chat_messages`.
- [ ] Support date range filters where relevant.
- [ ] Support optional channel scope for volume, top senders, and top emotes.
- [ ] Support optional sender scope for volume, top channels, and top emotes.
- [ ] Expose public read-only analytics routes.
- [ ] Keep route responses compact enough for dashboard/profile use.
- [ ] Add backend tests for aggregate correctness, empty datasets, date range filtering,
      channel scope, sender scope, and limit handling.

## Frontend Foundation Tasks

- [ ] Add typed analytics API wrappers and response types.
- [ ] Add tests for API parameter mapping where practical.

## Docs And Acceptance

- [ ] Document analytics API shape in README or docs.
- [ ] Update context docs.
- [ ] Verify backend tests and frontend typecheck/lint if frontend types are touched.
- [ ] Acceptance: landing, user profile, and channel profile features can consume analytics
      data without inventing new backend contracts.
