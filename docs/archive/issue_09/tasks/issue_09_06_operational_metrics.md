# Issue 9 Phase 6: Operational Metrics And Admin Operations Summary

## Scope

Expose ingestion health to operators through structured listener logs and the
`GET /admin/operations/summary` endpoint plus the `/admin` frontend operations dashboard.

## Out Of Scope

- Do not add a third-party metrics backend (Prometheus, OpenTelemetry exporter) in this phase.
- Do not change the API contract for non-operations endpoints.
- Do not change ClickHouse schema.

## Checklist

- [x] Add an in-process metrics struct in the listener tracking: - buffered writer queue depth (current and high-water mark) - last flush batch size - last flush duration - dropped event counter - ClickHouse insert failure counter per table - worker processed/failed per minute - circuit breaker state and current backoff delay
- [x] Worker batch log line includes batch size, duration, queue depth, pending backlog, and
      oldest pending age.
- [x] Add SQLite query for `oldest_pending_age_seconds` using the work-queue `enqueued_at`.
- [x] Extend `GET /admin/operations/summary` with new fields: - `raw_event_queue_depth` - `raw_event_oldest_pending_age_seconds` - `raw_event_write_queue_depth` - `raw_event_write_drop_count` - `clickhouse_insert_failures` (per table) - `clickhouse_breaker_state` - `worker_processed_per_minute` - `worker_failed_per_minute`
- [x] Update the operations schema struct and HTTP response shape.
- [x] Update the Next.js operations dashboard to render the new fields with the existing card
      layout and warning states (stale listener, growing backlog, breaker open).
- [x] Update README and `docs/context/decisions.md` with the new operations fields.

## Tests And Checks

- [x] Listener metrics test: counters increment on simulated drop, failure, and recovery events.
- [x] Operations summary handler test: response includes the new fields with expected types.
- [x] Frontend operations dashboard test: renders the new cards and warning states.
- [x] Existing admin auth/route tests still pass.

## Acceptance Criteria

- [x] An operator can read backlog, oldest pending age, writer drops, ClickHouse failures, and
      breaker state from `/admin` without reading Docker logs.
- [x] Listener log lines include enough fields to diagnose ingestion pressure under load.
- [x] All Go and frontend tests pass.

## Verification

```powershell
cd apps/api-go
go test ./...
go vet ./...
pnpm --filter @kick-logs/web test
pnpm --filter @kick-logs/web typecheck
pnpm --filter @kick-logs/web lint
pnpm format:check
git diff --check
```

## Commit Boundary

Commit message:

```text
feat(ops): expose ingestion backlog and breaker metrics
```
