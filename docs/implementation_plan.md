# Storage Hot Path Hardening

## Summary

Kick Logs already moved the main chat archive to ClickHouse, but some high-frequency listener paths
still write too much long-lived state to SQLite. This plan hardens the storage split without changing
the public API contract or deleting historical chat data.

Status: implemented on branch `feat/issue-23-storage-hot-path-hardening`.

Primary goal:

- ClickHouse remains the durable data-plane store for chat events, visible messages, processing
  attempts, and subscription periods.
- SQLite remains the control-plane store plus temporary work queues.
- Message ingestion should prefer preserving visible `chat_messages` over updating optional caches.

This plan is intentionally safe for production deployments with existing ClickHouse data. It must not
mutate or drop `chat_messages`.

## Current Problem

Observed production behavior:

- `chat_messages` is the actual table served to users.
- `raw_kick_events` archives every received raw Kick chat event.
- `raw_event_attempts` records processing history.
- SQLite `raw_event_queue` currently keeps processed rows forever.
- SQLite `sender_profiles` can receive one upsert per processed message.
- SQLite `kick_webhook_events` keeps processed and ignored webhook inbox rows forever.

That means SQLite drifts from "control-plane plus queue" toward another data-plane database. Under
heavy chat load this creates unnecessary write pressure and database growth.

## Locked Storage Rules

### ClickHouse Owns Data-Plane History

ClickHouse tables are long-lived history:

- `chat_messages`
- `raw_kick_events`
- `raw_event_attempts`
- `channel_subscription_periods`

Search, export, analytics, profile pages, and subscription summaries continue to read from
ClickHouse-backed normalized tables. No frontend contract changes are expected.

### SQLite Owns Control Plane And Temporary Queues

SQLite tables are control-plane or temporary runtime state:

- `admin_users`
- `followed_channels`
- `sender_profiles` as a best-effort cache
- `retention_settings`
- `worker_heartbeats`
- `data_migration_runs`
- `kick_event_subscriptions`
- `raw_event_queue` as temporary pending/failed work only
- `kick_webhook_events` as a short-retention webhook inbox

Processed queue/inbox rows should not live forever in SQLite.

### No User-Visible Data Loss

The application serves historical chat from `chat_messages`. This work must not delete or rewrite
that table. If a queue row is pruned after a successful processed attempt, that does not remove the
visible message.

## Production Safety Rules

- Apply this work through normal migrations and code deploys; do not require a manual destructive
  SQL step.
- Keep migration statements idempotent.
- Do not add ClickHouse mutations against `chat_messages`.
- Existing processed rows in SQLite can be pruned only when they are no longer needed for retry.
- `raw_event_attempts` remains the source used by startup backfill to know whether a raw event was
  already processed.
- If a processed queue row is removed but its ClickHouse processed attempt exists, startup backfill
  must not re-enqueue it.
- If a ClickHouse attempt insert fails, do not delete the queue row as processed.
- Failed queue rows should remain available for admin inspection/retry.

## Target Behavior

### Raw Event Queue

`raw_event_queue` should contain only:

- pending rows
- claimed rows
- failed rows that need admin action or retry

Successful rows should be removed from the queue after:

1. the message batch was inserted or confirmed duplicate,
2. the processed attempt was written to ClickHouse, and
3. the worker is ready to acknowledge the queue item.

The repository method can keep the existing name `MarkProcessed`, but its implementation should
delete the queue row instead of updating it to `processed`.

### Raw Event Attempts

`raw_event_attempts` remains in ClickHouse. It is the audit/history table and also protects startup
backfill from re-enqueuing already processed raw events.

Malformed raw events that can never succeed should not be retried forever. Examples:

- missing message id
- malformed JSON payload
- unsupported/incomplete chat payload shape that cannot produce a valid chat message

These should be recorded as terminal ignored/invalid outcomes in ClickHouse attempt history and
removed from the active queue, or otherwise marked terminal without staying in a retry loop.

### Sender Profiles

`sender_profiles` is a cache, not the source of truth for visible messages. A sender profile write
must never block `chat_messages` insertion.

Required behavior:

- Build the `ChatMessage` from the sender data already present in the raw payload.
- Attempt to upsert the sender profile cache only when useful.
- If sender cache upsert fails, log it and continue processing the message.
- Add a TTL/throttle gate so repeated messages from the same sender do not cause a SQLite write per
  message.

Initial acceptable TTL:

- one cache write per sender every 10 minutes in the listener process.

The cache may be in-memory. It does not need to survive process restart.

### Webhook Inbox

`kick_webhook_events` should remain an idempotent receiver inbox, but processed and ignored rows do
not need to live forever in SQLite.

Target behavior:

- Keep pending and failed webhook events until processed/retried/admin inspected.
- Prune processed and ignored webhook events older than a short retention window.
- Default retention: 7 days.
- Expose the retention as config only if implementation complexity stays low.

Normalized subscription periods already live in ClickHouse, so pruning processed inbox rows must not
change public subscription counts.

### Admin Operations

Operations UI/API should distinguish:

- active queue state from SQLite (`pending`, `claimed`, `failed`)
- historical raw-event state from ClickHouse attempts
- storage size by table

If processed queue rows are no longer stored in SQLite, admin copy and metrics should not imply that
SQLite queue row counts represent all-time processed event history.

## Implementation Phases

### Phase 1 - Plan And Context

- Replace the stale active implementation plan with this storage hot-path plan.
- Update context files so future agents know issue #23 is active.
- Do not change runtime code in this phase.
- Commit as one docs feature.

### Phase 2 - Sender Profile Cache Becomes Best Effort

- Refactor listener message preparation so the sender snapshot from the raw payload is sufficient to
  build `ChatMessage`.
- Make `SenderProfileRepository.Upsert` failures non-fatal in listener processing.
- Log sender cache failures with raw event id and sender kick user id when available.
- Add tests proving a sender profile upsert error still produces a visible chat message and a
  processed raw event attempt.
- Commit as one listener safety feature.

### Phase 3 - Sender Profile Write Throttle

- Add a small in-memory TTL gate around sender profile cache writes.
- Default TTL: 10 minutes.
- Avoid introducing another dependency or persistent store for this gate.
- Ensure first observation of a sender still writes immediately.
- Ensure repeated messages from the same sender within the TTL do not upsert again.
- Add focused unit tests.
- Commit as one listener cache-throttle feature.

### Phase 4 - Delete Processed Queue Rows

- Change SQLite `RawEventQueueRepository.MarkProcessed` to remove the row.
- Keep the method idempotent: calling it for an already-deleted row should return nil.
- Update fake queue repositories and tests to expect missing rows after successful processing.
- Confirm `CountPending`, `OldestPendingAge`, and admin queue depth still only count active
  pending/claimed rows.
- Add or update tests proving startup backfill does not re-enqueue a raw event that already has a
  ClickHouse `processed` attempt.
- Commit as one raw-event queue pruning feature.

### Phase 5 - Terminal Invalid Raw Events

- Classify permanent normalization failures separately from retryable failures.
- Initial terminal invalid cases:
  - missing message id
  - malformed raw payload JSON
  - missing followed channel after channel lookup proves the channel is not known
- Store a terminal attempt status in ClickHouse, for example `ignored` or `invalid`.
- Remove terminal invalid items from `raw_event_queue` so they do not retry forever.
- Keep truly transient failures retryable.
- Update failed-event admin behavior so terminal invalid events are visible or intentionally excluded
  according to the chosen status.
- Add tests for terminal invalid versus retryable failures.
- Commit as one raw-event invalid-classification feature.

### Phase 6 - Webhook Inbox Retention

- Add a repository method for pruning processed/ignored webhook inbox rows older than the retention
  window.
- Add config for retention only if it stays simple; otherwise use a constant default of 7 days.
- Run pruning from the webhook processor loop or another existing API background path.
- Do not prune pending or failed webhook events.
- Add repository/service tests.
- Commit as one webhook inbox retention feature.

### Phase 7 - Admin Operations Clarification

- Adjust operations response/UI copy if needed to clarify:
  - SQLite queue depth is active work, not all-time history.
  - processed raw history comes from ClickHouse attempts.
  - failed queue rows may be retryable, while terminal invalid rows are not ordinary backlog.
- Ensure storage table rows still display row counts and sizes accurately after queue pruning.
- Add/update frontend and backend tests if response shape/copy changes.
- Commit as one admin operations feature.

### Phase 8 - Docs And Verification

- Update:
  - `docs/architecture.md`
  - `docs/project_plan.md` if storage wording changed
  - `docs/context/living_brain.md`
  - `docs/context/decisions.md`
  - `docs/context/change_log.md`
  - `docs/context/recent_changes.md`
- Run relevant validation:
  - `go test ./...`
  - `go vet ./...`
  - frontend tests/typecheck only if UI changed
  - `pnpm format:check`
- Commit final docs/verification updates.

## Test Plan

Backend unit tests:

- Sender profile upsert failure does not fail raw event processing.
- Sender profile cache write is throttled per sender.
- Processed queue rows are deleted.
- `MarkProcessed` remains idempotent.
- Pending/claimed queue counts are unchanged by processed-row deletion.
- Backfill skips events with processed ClickHouse attempts.
- Terminal invalid payloads stop retrying.
- Retryable failures still retry until max attempts.
- Webhook retention prunes only processed/ignored old rows.
- Webhook retention leaves pending/failed rows untouched.

Backend integration-style tests:

- Listener processes a batch, writes `chat_messages`, writes processed attempts, and leaves no
  processed queue rows.
- Startup after processing does not refill the queue with already-processed raw events.
- Failed event admin endpoints still show retryable failed items.

Frontend tests:

- Only required if admin operations response/copy changes.

Manual production checks after deploy:

- Back up Docker volumes before deploy.
- Restart with the new image.
- Verify `/health`.
- Verify `/search` can query existing historical messages.
- Verify admin Operations:
  - queue depth does not grow indefinitely under normal traffic
  - failed raw events are understandable
  - ClickHouse failures remain stable
  - listener heartbeat is fresh
- Watch `listener` logs during a high-traffic channel.

## Out Of Scope

- Replacing SQLite queue with RabbitMQ, NATS, Kafka, Redis, or ClickHouse queue tables.
- Removing `raw_kick_events` or `raw_event_attempts`.
- Changing public `/messages` response shape.
- Changing frontend search/profile behavior.
- Deleting historical `chat_messages`.
- Adding distributed multi-node listener coordination.
