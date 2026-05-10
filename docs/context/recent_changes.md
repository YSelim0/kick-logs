# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest

- `/search` date inputs now default to a 7-day range.
- Behavior:
  - `Başlangıç`: current local date/time minus 7 days.
  - `Bitiş`: current local date/time.
  - missing URL date params receive these defaults.
  - manually cleared date fields are still omitted from backend query params.
- Updated frontend tests and project/context/design docs for the date default behavior.
- Verification so far:
  - `pnpm --filter @kick-logs/web test`: 2 files, 9 tests passed
  - `pnpm --filter @kick-logs/web typecheck`: passed
  - `pnpm --filter @kick-logs/web lint`: passed
  - `pnpm --filter @kick-logs/web build`: passed
- `/admin` workflow is still deferred to Phase 9.

## Commit Context

- Previous committed units:
  - `3c9178b feat(backend): complete phase six acceptance`
  - `c8c5eb9 feat(web): scaffold frontend foundation`
  - `f2250a9 feat(docs): complete phase seven foundation`
  - `619f4f9 feat(search): add public message search ui`
- Latest completed unit:
  - `/search` default date range behavior
- Commit message for this unit:
  - `feat(search): default date range filters`
