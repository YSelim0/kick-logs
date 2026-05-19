# Implementation Plan

## Active Feature

GitHub issue #9: stabilize high-volume Kick chat ingestion with ClickHouse batching and
backpressure.

Branch: `feat/issue-9-ingestion-batching`.

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

1. Phase 1: SQLite raw-event work queue (`docs/tasks/issue_09_01_sqlite_work_queue.md`).
2. Phase 2: ClickHouse batch insert repositories (`docs/tasks/issue_09_02_clickhouse_batches.md`).
3. Phase 3: Buffered websocket raw-event writer (`docs/tasks/issue_09_03_buffered_writer.md`).
4. Phase 4: Worker batch normalization output (`docs/tasks/issue_09_04_worker_batch_output.md`).
5. Phase 5: Exponential backoff and shared circuit breaker
   (`docs/tasks/issue_09_05_backoff_circuit_breaker.md`).
6. Phase 6: Operational metrics and admin operations summary
   (`docs/tasks/issue_09_06_operational_metrics.md`).
7. Phase 7: Load test and final smoke (`docs/tasks/issue_09_07_load_test_smoke.md`).

Each phase is a single commit boundary using `feat(scope): title`. Verification commands run at
the end of each phase before commit.

## Out Of Scope

- Adding RabbitMQ, NATS, Kafka, or other external queues.
- Changing the public API surface or the frontend contract.
- Changing ClickHouse table schemas in ways that break the archive role (additive columns are
  acceptable when a phase needs them).
- Changing PostgreSQL legacy migration behavior.

## Verification Baseline

Every phase must pass:

```powershell
cd apps/api-go
go test ./...
go vet ./...
pnpm format:check
git diff --check
```

Phases that change Compose, env, or runtime behavior also run:

```powershell
docker compose up --build -d
docker compose ps
```
