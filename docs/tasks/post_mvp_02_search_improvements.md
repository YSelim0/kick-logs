# Post-MVP Feature 2 Tasks: Search Improvements

## Goal

Improve the public `/search` workflow with better result readability, quick date
controls, richer filters, and export.

## Scope

This feature owns search behavior only. Do not build landing analytics, user
profiles, channel profiles, or admin data cleanup here.

## Backend Tasks

- [ ] Extend public message search filters with `reply_only` and `emote_only`.
- [ ] Keep existing optional `AND` semantics for all filters.
- [ ] Add filtered export support for CSV and JSON with a safe maximum row limit.
- [ ] Reuse the same search filter semantics for export and on-screen search.
- [ ] Add backend tests for reply-only search, emote-only search, combined filters, export
      authorization-free access, export row limit, CSV shape, and JSON shape.

## Frontend Tasks

- [ ] Add date presets: last 1 hour, last 24 hours, last 7 days, and last 30 days.
- [ ] Add reply-only and emote-only controls to the search form.
- [ ] Highlight matched `q` text inside message content without breaking inline emote
      rendering.
- [ ] Add CSV and JSON export actions that use the current submitted filters.
- [ ] Keep URL query state shareable for the new filters.
- [ ] Add tests for date presets, query state, filter mapping, highlight rendering, emote
      compatibility, and export button behavior.

## Docs And Acceptance

- [ ] Update `docs/design/design.md` for the new search controls.
- [ ] Update README search usage.
- [ ] Update context docs.
- [ ] Verify frontend tests/typecheck/lint/build and relevant backend tests.
- [ ] Acceptance: users can refine searches faster, visually identify matched text, and export
      the currently filtered result set.
