# ClickHouse Logging

## Policy

Routine query/profiling history is disabled in the Compose deployment. Application SELECT/INSERT
queries continue to work, but they no longer produce historical system-log rows. This avoids
unbounded diagnostic storage and background merges on the small VPS.

- `clickhouse/config.d/logging.xml` removes the query, thread, view, trace, text, part, metric,
  asynchronous metric, error-counter, processor-profile, asynchronous-insert, and OpenTelemetry
  log collectors.
- `clickhouse/users.d/logging.xml` disables query logging and sampling in the `default` profile,
  also used by the Docker-created `CLICKHOUSE_USER`. It does not change credentials or grants.
- Server warnings/errors still go to the console and the existing server log files. Each file
  stream rotates at 10 MiB with three archives; this is not a total disk quota.
- Docker console logs have their own rotation: `json-file`, 10 MiB, three files.
- Live diagnostics (`system.processes`, `system.metrics`, `system.events`, `system.errors`,
  `system.parts`, `system.merges`) remain available. Crash and backup logs remain enabled.
- Application history (`chat_messages`, `raw_kick_events`, `raw_event_attempts`, subscription
  periods, requests) and all SQLite/NATS state are unchanged.

Tradeoff: historical query analysis and ClickHouse dashboards backed by disabled log tables no
longer collect new data. Diagnose performance with controlled `EXPLAIN`, client timings, live
metrics, or temporarily re-enable the required collector during an investigation.

## Deploy

After deploying these files, recreate ClickHouse to apply the new mounts and Docker log rotation.
A plain `docker compose restart` does not add mounts or change the logging driver configuration.
There is a short API outage. The listener can continue publishing to JetStream during this window;
monitor queue capacity and keep the maintenance window short.

Run from the repository root on the VPS:

```bash
docker compose config --quiet
docker compose stop api processor
docker compose up -d --no-deps --force-recreate --wait clickhouse
# Continue only after ClickHouse is healthy.
docker compose start api processor
docker compose ps
docker compose logs --tail=50 clickhouse api processor
```

Do not use `down -v`, volume pruning, or data-directory deletion. No application schema migration
or historical message rewrite is required.

## Verify

Define this helper in the same Bash terminal. It uses the container's existing credentials without
printing them or requiring a password in shell history:

```bash
ch() {
  docker compose exec -T clickhouse sh -c '
    exec clickhouse-client --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" \
      --receive_timeout 3600 --send_timeout 3600 --max_execution_time 0 "$@"
  ' sh "$@"
}

ch --query "SELECT 1"
ch --query "
  SELECT name, value FROM system.settings
  WHERE name IN (
    'log_queries', 'log_query_threads', 'log_query_views', 'log_processors_profiles',
    'query_profiler_real_time_period_ns', 'query_profiler_cpu_time_period_ns'
  ) ORDER BY name FORMAT PrettyCompact
"
```

All six values should be zero. Also confirm the merged server configuration has no query-log
collector (the following command should print nothing):

```bash
docker compose exec -T clickhouse clickhouse extract-from-config \
  --config-file=/etc/clickhouse-server/config.xml --key=query_log.database --try
```

Exercise a search and inspect processor health. On an existing installation, log tables can still
exist with old rows: presence alone does not mean collection is enabled. Compare row counts before
and after a test query and `SYSTEM FLUSH LOGS`; disabled collectors should not add rows. On a fresh
volume the disabled log tables normally do not exist.

## Optional Existing History Cleanup

Disabling collectors does **not** delete old history. First inspect the largest system log tables:

```bash
ch --query "
  SELECT table, sum(rows) AS rows, formatReadableSize(sum(bytes_on_disk)) AS size
  FROM system.parts
  WHERE database = 'system' AND active
    AND (endsWith(table, '_log') OR match(table, '_log_[0-9]+$'))
  GROUP BY table ORDER BY sum(bytes_on_disk) DESC FORMAT PrettyCompact
"
```

Only after verifying the new configuration and deciding that this diagnostic history is no longer
needed, the following **irreversibly removes old rows from the explicitly named system logs**.
Export required diagnostics first. It does not touch any application table:

```bash
for table in query_log query_thread_log query_views_log trace_log text_log part_log \
  metric_log asynchronous_metric_log error_log processors_profile_log \
  asynchronous_insert_log opentelemetry_span_log; do
  ch --query "TRUNCATE TABLE IF EXISTS system.${table} SYNC" || break
done
```

Older versions may have renamed log tables such as `query_log_0`. They are intentionally not
automatically selected for deletion: inspect them and explicitly truncate only reviewed diagnostic
tables. Physical space reclamation may lag while files are still referenced. This procedure does
not remove pre-existing text-log archives or orphaned Docker volumes; inspect their exact paths
separately and never prune application volumes to reclaim logs.

## Rollback / Temporary Diagnostics

Remove only the two logging XML mounts from Compose and recreate ClickHouse using the same
maintenance sequence above. Keep the memory configuration and all data volumes. This restores
image-default logging, which can grow rapidly, so use it only for a bounded investigation. Already
truncated history cannot be restored without a backup.

## References

- [ClickHouse 24.8 server configuration](https://github.com/ClickHouse/ClickHouse/blob/v24.8.14.39-lts/programs/server/config.xml)
- [System-log collector creation in 24.8](https://github.com/ClickHouse/ClickHouse/blob/v24.8.14.39-lts/src/Interpreters/SystemLog.cpp)
- [Docker user profile initialization in 24.8](https://github.com/ClickHouse/ClickHouse/blob/v24.8.14.39-lts/docker/server/entrypoint.sh)
