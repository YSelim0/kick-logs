# Implementation Plan

## Active Feature

No active feature. The previous issue #9 plan is archived under
`docs/archive/issue_09/implementation_plan.md`.

## Completed Plans

- Issue #9: Stabilize high-volume Kick chat ingestion with ClickHouse batching and backpressure.
  Archived under `docs/archive/issue_09/`. Branch: `feat/issue-9-ingestion-batching`.
- Go + ClickHouse rewrite (phases 1-9). Archived under `docs/archive/go_rewrite/`.
- Post-MVP features (features 1-8). Archived under `docs/archive/post_mvp/`.
- MVP (phases 1-10). Archived under `docs/archive/mvp/`.

## Known Follow-Up Items

These were identified during issue #9 and deferred. They are candidates for the next issue:

1. **Worker hot-path sender resolution** — `senderResolver.ResolveSender` hits the Kick web API
   once per message inside the worker tick. Should be removed from the hot path; payload sender
   data (`username`, `slug`, `color`, `profile_picture`) is sufficient. A background enrichment
   worker can refresh profile images separately.

2. **Batch duplicate check** — `messages.ExistsByKickMessageID` runs one ClickHouse query per
   message per tick. Replace with `ExistingKickMessageIDs(ctx, ids []string) (map[string]bool,
error)` for a single `IN (...)` query per tick.

3. **Batch raw event load** — `rawEvents.GetByID` runs one ClickHouse point-lookup per queue
   item per tick. Replace with `GetByIDs(ctx, ids []string) ([]domain.RawKickEvent, error)` for
   a single query per tick.
