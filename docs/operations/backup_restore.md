# Backup And Restore

Kick Logs stores runtime data in Docker volumes:

- `api_go_data`: SQLite control-plane state (`admin_users`, followed channels, sender cache,
  heartbeats, webhook inbox/registry).
- `clickhouse_data`: durable chat messages, raw Kick events, attempts, analytics/profile history,
  and subscription periods.
- `nats_data`: JetStream durable backlog for raw chat events that reached the listener but are not
  fully processed yet.

Back up all three volumes plus `.env` if you want a new machine to resume from the backup point.

## Before Backup

For the cleanest backup, stop the app first:

```bash
docker compose down
```

Do not run `docker compose down -v` unless you intentionally want to delete stored data.

## Backup

Adjust `BACKUP_DIR` and volume names if your Compose project name is not `kick-logs`.

```bash
BACKUP_DIR="$HOME/kick-logs-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

cp .env "$BACKUP_DIR/.env"

docker run --rm -v kick-logs_api_go_data:/volume -v "$BACKUP_DIR":/backup alpine \
  tar -czf /backup/api_go_data.tar.gz -C /volume .

docker run --rm -v kick-logs_clickhouse_data:/volume -v "$BACKUP_DIR":/backup alpine \
  tar -czf /backup/clickhouse_data.tar.gz -C /volume .

docker run --rm -v kick-logs_nats_data:/volume -v "$BACKUP_DIR":/backup alpine \
  tar -czf /backup/nats_data.tar.gz -C /volume .
```

`nats_data` is not the long-term archive, but it can contain unprocessed chat events. If it is not
backed up while backlog exists, those unprocessed events cannot be replayed on the restored machine.

## Restore

Clone the repository on the target machine, copy the backup directory into the repo root, then run:

```bash
docker compose down

docker volume create kick-logs_api_go_data
docker volume create kick-logs_clickhouse_data
docker volume create kick-logs_nats_data

cp "$BACKUP_DIR/.env" .env

docker run --rm -v kick-logs_api_go_data:/volume -v "$BACKUP_DIR":/backup alpine \
  sh -c "rm -rf /volume/* && tar -xzf /backup/api_go_data.tar.gz -C /volume"

docker run --rm -v kick-logs_clickhouse_data:/volume -v "$BACKUP_DIR":/backup alpine \
  sh -c "rm -rf /volume/* && tar -xzf /backup/clickhouse_data.tar.gz -C /volume"

docker run --rm -v kick-logs_nats_data:/volume -v "$BACKUP_DIR":/backup alpine \
  sh -c "rm -rf /volume/* && tar -xzf /backup/nats_data.tar.gz -C /volume"

docker compose up --build -d
```

After restore, check:

```bash
docker compose ps
docker compose logs -n 100 api listener processor
```

Then open `/admin/operations` and verify listener heartbeat, processor heartbeat, JetStream backlog,
latest message time, and ClickHouse storage summary.
