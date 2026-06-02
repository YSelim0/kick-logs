# JetStream Cutover

This runbook is for deploying the issue #23 ingestion shape:

```text
listener -> NATS JetStream -> processor -> ClickHouse
```

## Preconditions

- A current backup exists for `.env`, `api_go_data`, `clickhouse_data`, and `nats_data`.
- `.env` contains the expected `NATS_*`, `CLICKHOUSE_*`, and `SQLITE_PATH` values.
- Docker Compose renders the expected services:

```bash
docker compose config --services
```

Expected default services:

```text
nats
clickhouse
api
listener
processor
web
```

## Deploy

Short downtime is acceptable for this cutover.

```bash
git pull
docker compose down
docker compose up --build -d
```

## Verify

```bash
docker compose ps
docker compose logs -n 100 nats listener processor api
```

Then open `/admin/operations` and verify:

- listener heartbeat is fresh,
- processor heartbeat is fresh,
- JetStream pending and ack-pending counts are visible,
- redelivery count is not growing continuously,
- latest raw event time advances,
- latest message time advances,
- legacy SQLite queue is not presented as the active live queue.

Public smoke:

```text
/health
/search
/admin/operations
```

## Rollback

If capture or processing fails and cannot be fixed quickly:

```bash
docker compose down
git checkout <known-good-commit-or-branch>
docker compose up --build -d
```

Do not delete `nats_data` during investigation. It can contain unprocessed raw chat events from the
failed cutover window.

If the old version does not understand JetStream backlog, keep `nats_data` for later replay or
manual inspection and confirm ClickHouse latest message time before deciding whether to discard it.
