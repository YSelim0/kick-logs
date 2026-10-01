# Public Analytics Performance

## Scope

Issue #27's owner-approved first pass covers the homepage, identity directories and profile loading
visuals. Message search/export and profile all-time analytics remain unchanged. No message or
subscription history is deleted, re-keyed, or backfilled. The later SQL-only profile improvement
from issue #29 preserves the all-time contract; its implementation and measurements are below.

## Runtime

- `GET /analytics/homepage` reads a memory snapshot, not ClickHouse. It covers today and the preceding
  13 UTC calendar days through `as_of`; late arrivals appear on the next successful refresh.
- The API refreshes once at startup if needed, then every 15 minutes. It uses the existing `FINAL`
  queries, preserving replacement/deduplication semantics and computing period-wide distinct users
  directly (not by summing daily unique counts).
- Five queries run sequentially: overview, volume, top channels, top users, top emotes. Each is limited
  to two threads, 384 MiB and 15 seconds; the whole refresh has a 90-second deadline. This avoids
  visitor-driven concurrency, but periodic aggregate cost still depends on recent traffic volume.
- A complete result is published atomically. Refresh errors retain the previous result and retry
  with 1, 2, 4, 8, then 15-minute delays. No partial snapshot is published.
- `homepage-analytics-v1.json` beside `SQLITE_PATH` is an atomic, size-bounded, versioned cache. The
  default Docker data volume preserves it across API recreation. Missing, corrupt, future-dated,
  incompatible or older-than-24-hour snapshots are ignored and rebuilt. A file-write failure does
  not discard a valid memory snapshot.
- Ready responses include actual dates and `stale`; stale data is at most 24 hours old and may belong
  to the previous UTC day. HTTP 202 + `Retry-After: 5` means no valid snapshot exists yet. Browser
  retries are bounded; after readiness there is no homepage polling.
- The scheduler is per API process. Multiple API replicas would need shared refresh coordination;
  this change targets the existing single-API Compose deployment.

## Directory Coverage

`GET /directory/users` reads `sender_profiles`; `/directory/channels` reads `followed_channels`.
Migration 9 indexes normalized name/slug expressions. Prefix queries never read chat messages and
return bounded cursor pages without counts. ASCII Kick identity matching is case-insensitive;
SQLite's built-in case folding is not full Unicode folding. `_` and `-` normalize to the same form.

Retained inactive channels and old cached users remain discoverable, irrespective of message age.
This is not a historical identity backfill: a sender present only in ClickHouse but missing from the
best-effort SQLite metadata cache is absent from directory results. Profile and message lookup
contracts remain available as before. Assess metadata coverage before any later backfill project.

## Rollout And Verification

1. Back up the existing SQLite and ClickHouse volumes as usual. Do not run `down -v`.
2. Rebuild/recreate API and web from the same revision: `docker compose up -d --build api web`.
3. API startup applies migration 9 to existing SQLite metadata; allow the one-time index build.
4. Check `/analytics/homepage`: expect 200 ready, or 202 while the first snapshot is prepared.
   Verify 14 ordered daily bins, matching `start`/`end`/`as_of`, and `Son 14 gün` labels on `/`.
   The UI intentionally omits the explicit date range and update timestamp.
5. Check prefix search and pagination on `/users` and `/channels`; no `top-*` analytics request
   should occur there. Check both profile loading layouts and all-time values after loading.
6. Recreate only the API and verify a valid persisted snapshot remains available without waiting
   for an aggregate refresh. Inspect warnings for refresh failures or unwritable cache storage.

Unit/HTTP/frontend tests cover failure, retry, cancellation, pagination and snapshot boundaries.
The opt-in ClickHouse CI suite verifies bounded dates and duplicate redelivery with real queries.
Directory query-plan tests verify indexed ranges, not full metadata scans.

Before claiming production acceptance, record p50/p95 response latency, peak memory and ingestion
backlog under the same workload as the prior version. Small isolated fixtures prove behavior, not
the performance of an 18-million-message VPS. The profile optimization below addresses one query
pair, not every profile query. Metadata completeness/backfill and production load benchmarking
remain separate follow-up work under issue #29.

Rollback can redeploy the previous revision without deleting data. The extra SQLite indexes and
homepage cache do not alter stored message/subscription rows; neither needs removal for rollback.

## All-Time Profile Query Optimization

User/channel profiles now request overview plus the top five counterpart channels/senders in a
single ClickHouse query through the optional `ProfileSummaryRepository` port. `GROUP BY ... WITH
TOTALS` returns both ranked rows and exact totals across every matching group before `LIMIT`.
The native driver reads the totals after consuming the rows. One tuple `argMax` selects all identity
metadata using the existing `(message_created_at, ingested_at, id)` ordering.

The query retains `FINAL`, deleted-row filtering, exact distinct counts, nullable identity rules,
slug matching, and ranking order. There is no recent-window restriction on profile summaries.
Volume, top emotes, latest messages, UI, HTTP responses and profile cache behavior are unchanged.
The older independent queries remain for other analytics consumers and as a fallback if the
combined query fails or a repository does not implement the optional port. On failure this can
cost one attempted combined query plus the two original queries; existing partial-result behavior
is intentionally retained rather than redesigned in this SQL-only change.

No migration, index, projection, backfill, or historical data rewrite is needed. Recreate only the
API to deploy this change: `docker compose up -d --build --no-deps api`. Rolling back the API is
sufficient to restore the previous query path.

### Local Measurement

Measured on 2026-10-02 against local ClickHouse 24.8, using all history before
`2026-10-01T20:00:00Z`. This fixed cutoff excludes live arrivals from the comparison. The channel
matched 379,367 messages; the user matched 8,567. Both complete result objects matched the old
queries before measurement, including metadata and ranking order.

| Summary query pair | Old median | Combined median | Old read rows | Combined read rows | Old read bytes | Combined read bytes |
| ------------------ | ---------- | --------------- | ------------- | ------------------ | -------------- | ------------------- |
| Channel            | 1,040 ms   | 807 ms          | 2,800,248     | 1,400,124          | 501,009,881    | 297,725,542         |
| User               | 760 ms     | 558 ms          | 2,800,248     | 1,400,124          | 660,628,981    | 502,467,440         |

These are medians of three benchmark runs with three iterations each, one query thread, a 256 MiB
memory cap, 10-second execution limit and ClickHouse query cache disabled. Filesystem/page caches
were not cleared. Concurrent local ingestion and Docker/network overhead affect timing. This is
about 22%/26% less elapsed time for the changed pair and 50% fewer rows read, not a whole-page or
production p95 claim. Remaining profile queries still scan history and require separate evidence
before any further optimization.

### Reproduce Safely

From `apps/api-go`, point `CLICKHOUSE_ADDR`, `CLICKHOUSE_DATABASE`, `CLICKHOUSE_USERNAME` and
`CLICKHOUSE_PASSWORD` at the dataset to inspect using the normal application configuration. Prefer
a read-only database account. Set these additional variables (PowerShell example):

```powershell
$env:KICK_LOGS_PROFILE_BENCHMARK_CHANNEL = 'hype'
$env:KICK_LOGS_PROFILE_BENCHMARK_SENDER = 'botrix'
$env:KICK_LOGS_PROFILE_BENCHMARK_END = '2026-10-01T20:00:00Z'
go test ./internal/infra/clickhouse -run '^$' -bench '^BenchmarkProfileSummaries$' -benchtime=3x -count=3 -v
```

Choose existing representative slugs and a fixed historical cutoff. The benchmark runs SELECTs
only, does not migrate/insert, rejects empty profiles, and stops if old/new results differ. Run
during a suitable maintenance/test window because both variants intentionally read history.
Do not enable `KICK_LOGS_RUN_CLICKHOUSE_TESTS` or run the integration-test command against real
application data; that separate suite inserts fixtures and belongs in an isolated database.

Regression coverage includes empty profiles, data from 2020, identity fallbacks/nulls, renamed
metadata, duplicate replacements, tombstones, ties, limits of 1/5/default, identical HTTP responses,
unchanged cache hits and fallback after a combined-query error.
