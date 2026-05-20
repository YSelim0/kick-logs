# Issue 9 Phase 5: Exponential Backoff And Shared Circuit Breaker

## Scope

Wrap ClickHouse operations in the listener and worker hot paths with bounded exponential backoff
plus a shared circuit breaker. While the circuit is open, all listener goroutines (buffered
writer flush, workers, pending count) sleep on the breaker instead of tight-looping retries.

## Out Of Scope

- Do not redesign the buffered writer or worker batching.
- Do not add new metrics endpoints; metrics land in Phase 6.
- Do not change SQLite queue behavior.

## Checklist

- [ ] Add a reusable `backoff` helper with initial delay, max delay, multiplier, and jitter.
      Defaults: 1s initial, 30s max, multiplier 2, full jitter.
- [ ] Add a shared `clickhouseBreaker` (atomic state) with `Allow`, `RecordSuccess`,
      `RecordFailure`, and `WaitUntilAllowed` methods.
- [ ] Breaker opens after N consecutive failures (default 5) and stays open for the current
      backoff delay before allowing a single probe call.
- [ ] Apply the breaker around: - buffered writer batch insert - worker `InsertMessagesBatch` / `InsertAttemptsBatch` - any remaining ClickHouse calls in the hot path
- [ ] Workers and the buffered writer call `WaitUntilAllowed` before issuing the ClickHouse
      operation. Successful calls reset the breaker and backoff.
- [ ] Log breaker open/half-open/closed transitions at info level with the current delay.
- [ ] Add config knobs: - `LISTENER_CLICKHOUSE_BACKOFF_INITIAL_MS` (default 1000) - `LISTENER_CLICKHOUSE_BACKOFF_MAX_MS` (default 30000) - `LISTENER_CLICKHOUSE_BACKOFF_MULTIPLIER` (default 2) - `LISTENER_CLICKHOUSE_BREAKER_FAILURE_THRESHOLD` (default 5)

## Tests And Checks

- [ ] Backoff helper test: produces non-decreasing delays capped at max with jitter inside the
      configured window.
- [ ] Breaker test: opens after threshold failures and rejects calls during the open window.
- [ ] Breaker test: allows one probe after the open window, closes on success, re-opens on
      failure.
- [ ] Service test: simulated ClickHouse failure does not cause a tight retry loop; workers
      observe the breaker delay.
- [ ] Service test: workers share one breaker; opening it in one goroutine pauses the others.

## Acceptance Criteria

- [ ] Under simulated ClickHouse outage, the listener does not spam retries faster than the
      configured backoff window.
- [ ] When ClickHouse recovers, the breaker closes and normal throughput resumes within one
      probe cycle.
- [ ] All Go tests pass.

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
feat(listener): add ClickHouse backoff and circuit breaker
```
