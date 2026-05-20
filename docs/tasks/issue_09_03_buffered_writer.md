# Issue 9 Phase 3: Buffered Websocket Raw-Event Writer

## Scope

Replace the per-event ClickHouse insert in the websocket callback with a buffered batch writer.
The websocket callback enqueues raw events to an in-memory channel; a dedicated goroutine flushes
the buffer to ClickHouse using the Phase 2 batch insert, then enqueues the same set of events
into the SQLite work queue from Phase 1.

The durable-inbox guarantee remains: a websocket event is acknowledged only after it lands in
both ClickHouse archive and SQLite work queue. If either write fails, the writer retries the
batch.

## Out Of Scope

- Do not change worker output batching yet.
- Do not add full backoff or circuit breaker yet (basic retry only; Phase 5 adds proper backoff).
- Do not introduce external queues.
- Do not change SQLite queue schema beyond what Phase 1 added.

## Checklist

- [ ] Add config knobs with sane defaults: - `LISTENER_RAW_EVENT_WRITE_BATCH_SIZE` (default 500) - `LISTENER_RAW_EVENT_WRITE_FLUSH_INTERVAL_MS` (default 500) - `LISTENER_RAW_EVENT_WRITE_QUEUE_SIZE` (default 50000) - `LISTENER_RAW_EVENT_WRITE_MAX_RETRIES` (default 10)
- [ ] Add a `bufferedRawWriter` type with start/stop lifecycle tied to the listener context.
- [ ] Websocket callback enqueues to the buffered writer channel and returns immediately.
- [ ] When the in-memory queue is full, drop the oldest buffered event, log a warning with the
      drop counter, and continue. Document the policy in the listener doc.
- [ ] Flush triggers: batch size reached, flush interval elapsed, or shutdown.
- [ ] On flush: call `InsertEventsBatch` against ClickHouse, then enqueue the same slice into the
      SQLite work queue inside one SQLite transaction.
- [ ] If ClickHouse batch fails: retry the same batch with a short delay up to
      `LISTENER_RAW_EVENT_WRITE_MAX_RETRIES`, then log and drop with a metric counter.
- [ ] If SQLite enqueue fails after a successful ClickHouse write: retry SQLite enqueue. Never
      lose track of an event already archived to ClickHouse; recover by re-reading the archived
      ClickHouse rows on the next startup for any IDs missing from SQLite.
- [ ] Update `.env.example` with the new variables.
- [ ] Update `docs/context/decisions.md` and `docs/context/living_brain.md` with the new
      ingestion ordering and buffered writer guarantees.

## Tests And Checks

- [ ] Writer test: flush-on-size triggers a single ClickHouse batch insert with N rows.
- [ ] Writer test: flush-on-interval triggers a batch even when size threshold is not reached.
- [ ] Writer test: shutdown drains the buffer before returning.
- [ ] Writer test: full queue drops the oldest event and increments the drop counter.
- [ ] Writer test: ClickHouse batch failure retries the same batch and eventually drops with a
      counter after max retries.
- [ ] Writer test: SQLite enqueue failure after ClickHouse success retries until success.
- [ ] Listener service tests updated to reflect buffered writer ordering.

## Acceptance Criteria

- [ ] Websocket callback no longer calls `InsertEvent` directly.
- [ ] Under steady load the buffered writer flushes batches of up to
      `LISTENER_RAW_EVENT_WRITE_BATCH_SIZE` rows per ClickHouse insert.
- [ ] No raw event is acknowledged before both ClickHouse archive and SQLite queue writes
      succeed.
- [ ] Restart drains gracefully when context is cancelled.
- [ ] All Go tests pass.

## Verification

```powershell
cd apps/api-go
go test ./...
go vet ./...
pnpm format:check
git diff --check
docker compose up --build -d
docker compose ps
```

## Commit Boundary

Commit message:

```text
feat(listener): buffer raw events into ClickHouse batches
```
