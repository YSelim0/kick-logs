# Post-MVP Feature 2 Tasks: Search Improvements

## Goal

Improve the public `/search` workflow with better result readability, quick date
controls, richer filters, and export.

## Scope

This feature owns search behavior only. Do not build landing analytics, user
profiles, channel profiles, or admin data cleanup here.

## Backend Tasks

- [x] Extend public message search filters with `reply_only` and `emote_only`.
- [x] Keep existing optional `AND` semantics for all filters.
- [x] Add filtered export support for CSV and JSON with a safe maximum row limit.
- [x] Reuse the same search filter semantics for export and on-screen search.
- [x] Add backend tests for reply-only search, emote-only search, combined filters, export
      authorization-free access, export row limit, CSV shape, and JSON shape.

## Frontend Tasks

- [x] Add date presets: last 1 hour, last 24 hours, last 7 days, and last 30 days.
- [x] Add reply-only and emote-only controls to the search form.
- [x] Highlight matched `q` text inside message content without breaking inline emote
      rendering.
- [x] Render URLs inside message content as clickable links without breaking inline emote or
      highlighted text rendering.
- [x] Add CSV and JSON export actions that use the current submitted filters.
- [x] Keep URL query state shareable for the new filters.
- [x] Add tests for date presets, query state, filter mapping, highlight rendering, emote
      compatibility, clickable link rendering, and export button behavior.

## Docs And Acceptance

- [x] Update `docs/design/design.md` for the new search controls.
- [x] Update README search usage.
- [x] Update context docs.
- [x] Verify frontend tests/typecheck/lint/build and relevant backend tests.
- [x] Acceptance: users can refine searches faster, visually identify matched text, open links
      from messages safely, and export the currently filtered result set.
