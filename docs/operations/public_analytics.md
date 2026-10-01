# Public Analytics Performance

## Scope

Issue #27's owner-approved first pass covers the homepage, identity directories and profile loading
visuals. Message search/export and profile all-time analytics remain unchanged. No message or
subscription history is deleted, re-keyed, or backfilled.

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
   Verify 14 ordered daily bins, matching `start`/`end`/`as_of`, and labeled dates on `/`.
5. Check prefix search and pagination on `/users` and `/channels`; no `top-*` analytics request
   should occur there. Check both profile loading layouts and all-time values after loading.
6. Recreate only the API and verify a valid persisted snapshot remains available without waiting
   for an aggregate refresh. Inspect warnings for refresh failures or unwritable cache storage.

Unit/HTTP/frontend tests cover failure, retry, cancellation, pagination and snapshot boundaries.
The opt-in ClickHouse CI suite verifies bounded dates and duplicate redelivery with real queries.
Directory query-plan tests verify indexed ranges, not full metadata scans.

Before claiming production acceptance, record p50/p95 response latency, peak memory and ingestion
backlog under the same workload as the prior version. Small isolated fixtures prove behavior, not
the performance of an 18-million-message VPS. Profile-query optimization, metadata completeness
backfill, and production load benchmarking from the wider issue remain separate follow-up work.

Rollback can redeploy the previous revision without deleting data. The extra SQLite indexes and
homepage cache do not alter stored message/subscription rows; neither needs removal for rollback.
