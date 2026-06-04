# Ingestion Load Test

This document describes how to run the synthetic ingestion load test introduced as part of the
issue #23 JetStream ingestion work. The harness exercises the listener capture path, JetStream
durable backlog, processor batch output, and ClickHouse circuit breaker using a deterministic event
emitter so peak throughput can be measured without depending on live Kick streams.

The harness lives at `apps/api-go/cmd/loadgen`. It wires the real listener service with the same
JetStream publisher used by `cmd/listener` and the same processor service used by `cmd/processor`,
but replaces the Pusher websocket client with an in-process emitter that produces synthetic chat
events at a configurable rate.

## Prerequisites

- Local NATS JetStream, ClickHouse, and SQLite are reachable. The simplest setup is the default
  Docker stack:

  ```powershell
  docker compose up --build -d nats clickhouse
  ```

- The Go toolchain is installed and `apps/api-go` builds cleanly:

  ```powershell
  cd apps/api-go
  go build ./...
  ```

- NATS, ClickHouse, and SQLite configuration follows the standard `.env`/`.env.example` values. The
  loadgen binary reuses `config.Load` so it honors `NATS_*`, `LISTENER_*`, and `CLICKHOUSE_*`
  variables.

## Run

From the repository root:

```powershell
cd apps/api-go
go run ./cmd/loadgen `
  -events-per-second=2000 `
  -duration=60s `
  -channels=5 `
  -burst-factor=2 `
  -report-every=5s
```

Flag reference:

- `-events-per-second` (int, default 500): synthetic chat events per second across all
  channels during the baseline window.
- `-duration` (duration, default 30s): total wall-clock time the emitter runs.
- `-channels` (int, default 3): number of synthetic followed channels seeded into SQLite as
  enabled.
- `-burst-factor` (float, default 1): multiplier applied to the configured rate during the
  second half of the run, simulating a high-volume peak.
- `-report-every` (duration, default 5s): interval between metrics snapshots written to the
  log stream.

The loadgen seeds channels with slugs `loadgen-1`, `loadgen-2`, ... and Kick ids in the
`1_000_000+` range so it does not collide with real production channels. Its listener heartbeat is
recorded under `service_name = 'loadgen'`, while its processor heartbeat is recorded under
`service_name = 'loadgen-processor'`.

## What to watch

While the load is running:

- Tail the loadgen logs for the `loadgen snapshot` info lines. They report emitted event count,
  JetStream message count, consumer pending count, ack-pending count, redelivery count, and oldest
  pending event age.
- Open `/admin` in a browser. The operations dashboard surfaces the same fields plus queue
  backlog and oldest pending age (see `GET /admin/operations/summary` and the `ingestion`
  block).
- Watch `docker logs clickhouse` and `docker stats` for ClickHouse CPU, memory, and DNS
  pressure. The pre-issue-9 baseline showed `dial tcp: lookup clickhouse on 127.0.0.11:53:
server misbehaving` under burst load; that error should not reappear.

## Pass / fail thresholds

A successful run satisfies all of the following:

- No API `5xx` responses on `/health`, `/messages`, or `/analytics/overview` during the run.
- Listener log lines show successful JetStream publish capture. Processor log lines show ClickHouse
  insert batches sized in the hundreds (matching `NATS_RAW_EVENT_FETCH_BATCH_SIZE`) rather than
  single-row inserts.
- JetStream backlog may grow during the burst window, but it must drain after the emitter stops.
- There is no acceptable event-drop counter in the JetStream path. A non-zero drop/error metric is
  a bug or infrastructure failure.
- The ClickHouse circuit breaker stays `closed`. If it opens, the configured backoff should
  recover automatically and the breaker should close on the next probe.
- JetStream consumer pending count returns to near zero within `2x` the run duration after the
  loadgen stops.
- No `lookup clickhouse on 127.0.0.11:53: server misbehaving` errors in any container log.

## Cleanup

The loadgen channels and synthetic messages are kept in ClickHouse and SQLite so a follow-up
test can verify search and analytics work over generated data. To remove them after the run:

```powershell
docker compose exec clickhouse clickhouse-client --query="ALTER TABLE chat_messages DELETE WHERE channel_slug LIKE 'loadgen-%'"
docker compose exec clickhouse clickhouse-client --query="ALTER TABLE raw_kick_events DELETE WHERE channel_slug LIKE 'loadgen-%'"
```

Followed-channel rows can be removed with the admin disable flow or directly through the
SQLite CLI if a clean baseline is required.
