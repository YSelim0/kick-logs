# Issue 9 Phase 4: Worker Batch Normalization Output

## Scope

Refactor the raw-event worker so each tick claims up to N queue rows, normalizes them, and writes
the resulting `chat_messages` and `raw_event_attempts` rows as ClickHouse batches. SQLite work
queue rows are then marked processed or failed in a single transaction.

The per-event ClickHouse single-row inserts inside `processRawEvent` are removed from the hot
path.

## Out Of Scope

- Do not change buffered websocket writer behavior beyond what Phase 3 introduced.
- Do not add backoff or circuit breaker yet.
- Do not add new operational metrics yet.
- Do not introduce external queues.

## Checklist

- [x] Worker tick uses SQLite work queue to claim a batch of up to
      `LISTENER_RAW_EVENT_BATCH_SIZE` rows in a single transaction.
- [x] For each claimed row, fetch the raw event payload from the buffered writer cache when
      available; otherwise load by ID from ClickHouse with a single multi-key query.
- [x] Normalize each event into a `chat_messages` row and a `raw_event_attempts` row in memory.
      Sender resolution and `sender_profiles` upserts still happen per row.
- [x] After the loop, call `InsertMessagesBatch` once for the collected messages.
- [x] Call `InsertAttemptsBatch` once for the collected attempts (`processed` and `failed`).
- [x] Mark SQLite queue rows processed or failed in a single SQLite transaction per tick.
- [x] If batch ClickHouse write fails, release all claimed queue rows for the tick so they can be
      retried on the next worker pass. Do not partially mark some rows processed.
- [x] Remove the per-row `InsertAttempt` and `Insert(message)` calls from the hot path. Keep the
      single-row methods only if other use cases still need them.
- [x] Update listener service tests to reflect batched processing.

## Tests And Checks

- [x] Service test: a tick that claims 50 events produces exactly one
      `InsertMessagesBatch` and one `InsertAttemptsBatch` call.
- [x] Service test: when normalization fails for some rows, only those rows are marked failed and
      the rest are processed in the same tick.
- [x] Service test: ClickHouse batch failure releases every claimed row in the tick back to
      `pending` without losing the attempt counter.
- [x] Service test: duplicate `kick_message_id` events still dedupe and mark processed without
      double-inserting the visible message.
- [x] Existing dedupe and reply/emote tests still pass.

## Acceptance Criteria

- [x] One worker tick produces at most one ClickHouse insert per output table.
- [x] Pending count and oldest pending age come from SQLite.
- [x] Functional behavior is unchanged: messages, replies, emotes, and sender profile
      enrichment still work as before.
- [x] All Go tests pass.

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
feat(listener): batch worker output writes to ClickHouse
```
