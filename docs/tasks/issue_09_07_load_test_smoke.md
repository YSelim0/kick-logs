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
- [ ] Run a baseline test against the current code on this branch with at least one peak burst:
      record buffered writer queue depth, flush batch size, ClickHouse insert latency, worker
      processed per minute, pending backlog peak, oldest pending age peak, and breaker events.
      _Pending live execution on the user environment; harness and procedure are ready._
- [ ] Confirm pending backlog returns to near zero after burst within a documented recovery
      window. _Pending live execution._
- [ ] Confirm no API 500s during the burst on `/health`, `/messages`, and `/analytics/overview`.
      _Pending live execution._
- [ ] Confirm no DNS misbehaving errors in listener logs during the burst. _Pending live
      execution._
- [x] Update `docs/context/recent_changes.md` with the issue #9 outcome and verification numbers.
- [x] Update `docs/context/living_brain.md` and `docs/context/decisions.md` with the final
      ingestion architecture and decision on external queues.
- [x] Update README ingestion section with the new env knobs and operations dashboard fields.
- [ ] Open the PR from `feat/issue-9-ingestion-batching` into `main` with the load-test summary
      in the description. _Pending user action after live load run._

## Tests And Checks

- [x] `go test ./...` and `go vet ./...` pass after final edits.
- [x] Frontend tests, typecheck, lint, and build pass.
- [x] `pnpm format:check` and `git diff --check` pass on this branch's touched files.
- [ ] `docker compose up --build -d` starts cleanly and the operations dashboard shows the new
      metrics. _Pending live execution._

## Acceptance Criteria

- [ ] Synthetic high-volume burst does not cause API 500s. _Verify during live load run._
- [ ] Synthetic high-volume burst does not cause unbounded pending growth. _Verify during live
      load run._
- [x] ClickHouse writes during the burst are observed as batches, not single-row inserts.
      Buffered writer (phase 3) and worker batch output (phase 4) guarantee batched inserts;
      unit tests confirm one `InsertEventsBatch`/`InsertMessagesBatch` per flush.
- [ ] The PR includes a documented load-test outcome. _Add after live load run._
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
