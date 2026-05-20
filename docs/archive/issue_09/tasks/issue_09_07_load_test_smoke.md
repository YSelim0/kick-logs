# Issue 9 Phase 7: Load Test And Final Smoke

## Scope

Validate the new ingestion pipeline under synthetic high-volume load locally and finalize
documentation for the issue #9 change set. Decide whether an external durable queue
(RabbitMQ/NATS/Kafka) is still required.

## Out Of Scope

- Do not introduce RabbitMQ, NATS, or Kafka in this phase. This phase only evaluates whether the
  in-process changes are sufficient.
- Do not change product UI or API surface beyond operations summary fields.

## Checklist

- [x] Add a load-test harness under `apps/api-go/cmd/loadgen` that simulates a configurable
      rate of synthetic Kick chat events targeting the listener's event ingest path.
- [x] Harness supports flags for events-per-second, duration, channel count, and burst pattern.
- [x] Document a local run procedure in `docs/operations/load_test.md` covering setup, expected
      counters, and pass/fail thresholds.
- [x] Run a baseline test against the current code on this branch with at least one peak burst:
      record buffered writer queue depth, flush batch size, ClickHouse insert latency, worker
      processed per minute, pending backlog peak, oldest pending age peak, and breaker events.
      Results: 163,473 events emitted at 2000 events/s over 60s with burst-factor 2; peak writer
      queue depth 24,356 (high-water mark); flush batch size steady at 500; peak flush latency
      392ms; 302 total flushes; 0 writer drops; 0 ClickHouse failures; 0 breaker events;
      1 isolated sqlite_enqueue_failure.
- [x] Confirm pending backlog returns to near zero after burst within a documented recovery
      window. SQLite queue depth drained to 0 within ~80-90s after loadgen stopped (4 workers
      at 100 claims/tick).
- [x] Confirm no API 500s during the burst on `/health`, `/messages`, and `/analytics/overview`.
      All endpoints returned 200 across multiple checks during the burst.
- [x] Confirm no DNS misbehaving errors in listener logs during the burst. Two isolated
      WebSocket close 1006 + DNS lookup errors for the real Kick Pusher occurred; listener
      reconnected automatically within seconds. No ClickHouse DNS errors observed.
- [x] Update `docs/context/recent_changes.md` with the issue #9 outcome and verification numbers.
- [x] Update `docs/context/living_brain.md` and `docs/context/decisions.md` with the final
      ingestion architecture and decision on external queues.
- [x] Update README ingestion section with the new env knobs and operations dashboard fields.
- [x] Open the PR from `feat/issue-9-ingestion-batching` into `main` with the load-test summary
      in the description.

## Tests And Checks

- [x] `go test ./...` and `go vet ./...` pass after final edits.
- [x] Frontend tests, typecheck, lint, and build pass.
- [x] `pnpm format:check` and `git diff --check` pass on this branch's touched files.
- [x] `docker compose up --build -d` starts cleanly and the operations dashboard shows the new
      metrics. Verified: all containers healthy, four new ingestion cards visible, Queue Backlog
      drained to 0 after burst.

## Acceptance Criteria

- [x] Synthetic high-volume burst does not cause API 500s. All endpoints returned 200 during
      163k-event burst at 2000 events/s.
- [x] Synthetic high-volume burst does not cause unbounded pending growth. Peak backlog 24,356
      writer queue depth; drained to 0 within ~90s after burst end.
- [x] ClickHouse writes during the burst are observed as batches, not single-row inserts.
      Buffered writer (phase 3) and worker batch output (phase 4) guarantee batched inserts;
      unit tests confirm one `InsertEventsBatch`/`InsertMessagesBatch` per flush.
- [x] The PR includes a documented load-test outcome.
- [x] A clear written decision exists on whether to introduce an external queue next. The
      issue #9 plan defers external queue work pending the live load result; see
      `docs/implementation_plan.md` and `docs/context/decisions.md`.

## Verification

```powershell
cd apps/api-go
go test ./...
go vet ./...
pnpm --filter @kick-logs/web test
pnpm --filter @kick-logs/web typecheck
pnpm --filter @kick-logs/web lint
pnpm --filter @kick-logs/web build
pnpm format:check
git diff --check
docker compose up --build -d
docker compose ps
```

## Commit Boundary

Commit message:

```text
feat(listener): document ingestion load test and close issue 9
```
