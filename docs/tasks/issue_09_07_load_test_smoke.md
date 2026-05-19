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

- [ ] Add a load-test harness under `apps/api-go/cmd/loadgen` that simulates a configurable
      rate of synthetic Kick chat events targeting the listener's event ingest path.
- [ ] Harness supports flags for events-per-second, duration, channel count, and burst pattern.
- [ ] Document a local run procedure in `docs/operations/load_test.md` covering setup, expected
      counters, and pass/fail thresholds.
- [ ] Run a baseline test against the current code on this branch with at least one peak burst:
      record buffered writer queue depth, flush batch size, ClickHouse insert latency, worker
      processed per minute, pending backlog peak, oldest pending age peak, and breaker events.
- [ ] Confirm pending backlog returns to near zero after burst within a documented recovery
      window.
- [ ] Confirm no API 500s during the burst on `/health`, `/messages`, and `/analytics/overview`.
- [ ] Confirm no DNS misbehaving errors in listener logs during the burst.
- [ ] Update `docs/context/recent_changes.md` with the issue #9 outcome and verification numbers.
- [ ] Update `docs/context/living_brain.md` and `docs/context/decisions.md` with the final
      ingestion architecture and decision on external queues.
- [ ] Update README ingestion section with the new env knobs and operations dashboard fields.
- [ ] Open the PR from `feat/issue-9-ingestion-batching` into `main` with the load-test summary
      in the description.

## Tests And Checks

- [ ] `go test ./...` and `go vet ./...` pass after final edits.
- [ ] Frontend tests, typecheck, lint, and build pass.
- [ ] `pnpm format:check` and `git diff --check` pass.
- [ ] `docker compose up --build -d` starts cleanly and the operations dashboard shows the new
      metrics.

## Acceptance Criteria

- [ ] Synthetic high-volume burst does not cause API 500s.
- [ ] Synthetic high-volume burst does not cause unbounded pending growth.
- [ ] ClickHouse writes during the burst are observed as batches, not single-row inserts.
- [ ] The PR includes a documented load-test outcome.
- [ ] A clear written decision exists on whether to introduce an external queue next.

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
