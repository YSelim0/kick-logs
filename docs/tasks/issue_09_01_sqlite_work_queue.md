# Issue 9 Phase 1: SQLite Raw-Event Work Queue

## Scope

Move raw-event work-queue state out of ClickHouse into SQLite. ClickHouse keeps the durable
archive role for `raw_kick_events`, `chat_messages`, and `raw_event_attempts`. SQLite owns
pending list, claim ownership, attempt counter, status, and last error for each raw event.

After this phase, the listener and workers no longer run `GROUP BY` + `LEFT JOIN` queries
against ClickHouse `raw_event_attempts` to find pending work or count backlog.

## Out Of Scope

- Do not introduce ClickHouse batch insert methods yet.
- Do not change the websocket callback to buffered batching yet.
- Do not batch worker output writes yet.
- Do not add exponential backoff or circuit breaker yet.
- Do not change ClickHouse schemas beyond keeping the archive tables as-is.

## Checklist

- [ ] Add SQLite migration creating `raw_event_queue` with columns: `raw_event_id` (PK),
      `channel_id`, `chatroom_id`, `status`, `attempts`, `claimed_by`, `claimed_at`,
      `enqueued_at`, `last_error`.
- [ ] Add SQLite indexes: `(status, enqueued_at)` for pending scan, `(claimed_by, claimed_at)`
      for stale-claim recovery.
- [ ] Add domain status constants `pending`, `claimed`, `processed`, `failed`.
- [ ] Add `RawEventQueueRepository` port covering enqueue, list pending, claim, release,
      mark processed, mark failed, count pending, oldest pending age, and stale-claim recovery.
- [ ] Implement SQLite `RawEventQueueRepository`.
- [ ] Update listener websocket path so every stored raw event enqueues a queue row in the same
      logical unit. Failure to enqueue must not be silently swallowed.
- [ ] Replace `RawEventRepository.ListUnprocessed`/`CountUnprocessed` usage in the worker loop
      with the SQLite queue repository.
- [ ] Replace `RawEventClaimRepository` interaction with the queue repository if the queue
      repository fully covers the claim contract; otherwise keep the existing claim repo and
      route the worker through the queue repo for read paths only.
- [ ] Keep ClickHouse `raw_event_attempts` writes from `markRawEventProcessed`/`markRawEventFailed`
      unchanged for now (still single-row inserts). Phase 4 will batch them.
- [ ] Add a startup recovery step that resets `status='claimed'` rows whose `claimed_at` is older
      than `RawEventProcessingTimeout` back to `pending`.

## Tests And Checks

- [ ] Unit tests for SQLite migration apply on an empty database.
- [ ] Repository tests: enqueue then list pending in FIFO order by `enqueued_at`.
- [ ] Repository tests: claim transitions `pending`→`claimed`, sets `claimed_by`, `claimed_at`.
- [ ] Repository tests: double-claim returns false for the second caller.
- [ ] Repository tests: release returns row to `pending` and clears claim fields.
- [ ] Repository tests: mark processed sets terminal status and is idempotent.
- [ ] Repository tests: mark failed increments `attempts`, stores `last_error`, returns row to
      `pending` until `attempts >= RawEventMaxAttempts`.
- [ ] Repository tests: stale-claim recovery resets rows older than the configured timeout.
- [ ] Listener service tests: websocket callback enqueues a queue row alongside the ClickHouse
      raw insert.
- [ ] Listener service tests: worker uses SQLite queue list/count and does not query ClickHouse
      attempts JOIN.

## Acceptance Criteria

- [ ] Worker tick does not execute the ClickHouse `LEFT JOIN raw_event_attempts` query.
- [ ] Pending count comes from SQLite, not ClickHouse.
- [ ] Existing functional behavior is preserved: messages still dedupe by `kick_message_id`,
      raw archive still receives every event, attempt audit rows still get written.
- [ ] Restarting the listener does not lose pending raw events; queue rows survive restart and
      stale claims recover.
- [ ] All current Go tests pass.

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
feat(listener): move raw-event queue to SQLite
```
