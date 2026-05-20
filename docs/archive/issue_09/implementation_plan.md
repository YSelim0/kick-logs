# Issue #9: Stabilize High-Volume Chat Ingestion

Status: **Completed** on branch `feat/issue-9-ingestion-batching`. All 7 phases merged into
`main`. Live load test validated the pipeline at 2000 events/s with 0 drops and 0 API 500s.

## Problem Summary

Under high-volume Kick streams the listener degraded and the API returned 500s because:

- ClickHouse `raw_kick_events`/`raw_event_attempts` are used as a work queue. Every worker tick
  runs two heavy `GROUP BY` + `LEFT JOIN` queries over the attempts table to find pending rows
  and to count backlog. This pattern does not scale.
- Raw events, normalized messages, and attempt rows are inserted one row at a time. ClickHouse
  prefers fewer large inserts; single-row inserts under burst load create many parts, contention,
  and DNS pressure on the Docker network.
- On ClickHouse/DNS failures, every worker retries on a tight idle delay, hammering the network
  and amplifying the outage.
- Operations dashboard does not expose write-queue depth, oldest pending age, or flush health, so
  incidents are diagnosed from logs only.

## Architectural Direction

- Keep ClickHouse as the durable archive and analytical store for `raw_kick_events`,
  `chat_messages`, and `raw_event_attempts`. Stop using it as the hot-path work queue.
- Move raw-event work-queue state to SQLite: pending list, claim ownership, attempt counter,
  status, error message. SQLite indexes and updates make pending lookup and count O(log n) and
  remove the heavy ClickHouse JOIN from the hot path.
- Batch all ClickHouse writes from listener and workers using `PrepareBatch`. Buffer websocket
  events in memory and flush on size or interval. Workers collect normalized messages and
  attempts per cycle and write them as batches.
- Wrap ClickHouse operations with bounded exponential backoff plus a shared circuit breaker so
  workers stop tight-looping during transient ClickHouse/DNS failures.
- Expose write-queue depth, oldest pending age, flush size/latency, and ClickHouse error counters
  through the admin operations summary.
- Defer adding an external durable queue (RabbitMQ/NATS/Kafka) until the in-process changes
  above are validated under load.

## Phases

1. Phase 1: SQLite raw-event work queue (`tasks/issue_09_01_sqlite_work_queue.md`) — completed.
2. Phase 2: ClickHouse batch insert repositories (`tasks/issue_09_02_clickhouse_batches.md`) —
   completed.
3. Phase 3: Buffered websocket raw-event writer (`tasks/issue_09_03_buffered_writer.md`) —
   completed.
4. Phase 4: Worker batch normalization output (`tasks/issue_09_04_worker_batch_output.md`) —
   completed.
5. Phase 5: Exponential backoff and shared circuit breaker
   (`tasks/issue_09_05_backoff_circuit_breaker.md`) — completed.
6. Phase 6: Operational metrics and admin operations summary
   (`tasks/issue_09_06_operational_metrics.md`) — completed.
7. Phase 7: Load test and final smoke (`tasks/issue_09_07_load_test_smoke.md`) — completed.
   Live run: 163,473 events at 2000 events/s, 0 drops, 0 CH failures, 0 breaker events, full
   drain within ~90s post-burst.

## Load Test Results

Run: `go run ./cmd/loadgen -events-per-second=2000 -duration=60s -channels=5 -burst-factor=2`

| Metric | Value |
|--------|-------|
| Total events emitted | 163,473 |
| Writer drops | 0 |
| ClickHouse failures | 0 |
| Circuit breaker events | 0 |
| Peak writer queue depth | 24,356 |
| Writer high-water mark | 24,356 |
| Total flushes | 302 |
| Flush batch size (steady) | 500 |
| Peak flush latency | 392 ms |
| sqlite_enqueue_failures | 1 (isolated context timeout) |
| API 500s during burst | 0 |
| Queue drain time post-burst | ~90 s |

## Known Follow-Up Items

The following were identified after the load test and deferred to separate issues:

- `senderResolver.ResolveSender` is called per message in the worker hot path, hitting the Kick
  web API for every event. The loadgen masked this with a `noopSenderResolver`. Should be moved
  to a background enrichment worker.
- `ExistsByKickMessageID` runs one ClickHouse query per message per tick. Should be batched as
  `ExistingKickMessageIDs(ctx, ids []string) (map[string]bool, error)`.
- `rawEvents.GetByID` runs one ClickHouse query per queue item per tick. Should be batched as
  `GetByIDs(ctx, ids []string) ([]domain.RawKickEvent, error)`.

## Out Of Scope

- Adding RabbitMQ, NATS, Kafka, or other external queues.
- Changing the public API surface or the frontend contract.
- Changing ClickHouse table schemas in ways that break the archive role.
- Changing PostgreSQL legacy migration behavior.
