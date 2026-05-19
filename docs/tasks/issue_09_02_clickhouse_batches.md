# Issue 9 Phase 2: ClickHouse Batch Insert Repositories

## Scope

Add batch insert methods to ClickHouse repositories so callers can flush many rows in a single
`PrepareBatch`/`Send` cycle for `raw_kick_events`, `chat_messages`, and `raw_event_attempts`.

Repository batch methods are introduced in this phase; callers that use them live in Phase 3
(buffered websocket writer) and Phase 4 (worker batch output).

## Out Of Scope

- Do not change the websocket callback flow yet.
- Do not change worker output flow yet.
- Do not add backoff or circuit breaker yet.
- Do not change SQLite work queue behavior.

## Checklist

- [ ] Add `InsertEventsBatch(ctx, []RawKickEvent) error` to `RawEventRepository`.
- [ ] Add `InsertAttemptsBatch(ctx, []RawEventAttempt) error` to `RawEventRepository`.
- [ ] Add `InsertMessagesBatch(ctx, []ChatMessage) error` to `MessageRepository`.
- [ ] Update the matching ports in `internal/ports` so use cases can depend on the new methods.
- [ ] Implement each method with a single `PrepareBatch`, loop `Append`, then one `Send`.
- [ ] Preserve all field defaulting currently done in single-row insert paths (UUID generation,
      empty payload JSON, status default, UTC time normalization, nullable helpers).
- [ ] Tolerate empty slices: return `nil` immediately without calling `PrepareBatch`.
- [ ] Keep existing single-row insert methods intact during this phase to avoid breaking the
      hot path before Phases 3 and 4 land.

## Tests And Checks

- [ ] Repository tests: batch insert of N raw events stores all rows with correct field values.
- [ ] Repository tests: batch insert of N messages stores all rows with correct field values.
- [ ] Repository tests: batch insert of N attempts stores all rows with correct field values.
- [ ] Repository tests: empty slice returns nil and does not error.
- [ ] Repository tests: batch insert is safe to call repeatedly without duplicate-row failures
      under existing schema engines.

## Acceptance Criteria

- [ ] Callers can insert any of the three row types in batches without changes to ClickHouse
      schema.
- [ ] Single-row insert methods still work and continue to be used by current call sites.
- [ ] `go test ./...` and `go vet ./...` pass.

## Verification

```powershell
cd apps/api-go
go test ./...
go vet ./...
pnpm format:check
git diff --check
```

## Commit Boundary

Commit message:

```text
feat(clickhouse): add batch insert repositories
```
