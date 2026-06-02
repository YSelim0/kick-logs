# Issue 23 - JetStream Durable Ingestion Plan

## Summary

Issue 23 stays on branch `feat/issue-23-storage-hot-path-hardening`, but the implementation direction changes after the ingestion audit.

The first hardening work reduced SQLite growth and clarified operational data, but it did not fully satisfy the core product rule: once a Kick chat event reaches the application, the application must not silently drop it because an in-memory buffer is full, a retry budget is exhausted, or a planned reconnect cycle runs.

The long-term fix is to move the live chat ingestion hot path to NATS JetStream:

```text
Kick WebSocket listener -> NATS JetStream durable stream -> processor workers -> ClickHouse batch inserts
```

SQLite remains for control-plane data only. ClickHouse remains the long-lived analytics and search store. NATS JetStream is only a durable backlog for unprocessed raw chat events.

Public API contracts, frontend request/response shapes, and existing ClickHouse history must remain compatible.

## Core Principle

The application's most important rule is data capture continuity.

If a Kick chat event reaches the listener process, the system must either:

- durably persist it to the ingestion stream, or
- fail loudly with visible operational errors/backpressure.

The system must never intentionally discard a reached event silently.

The target delivery model is **at-least-once ingestion with idempotent writes**, not false exactly-once semantics. Duplicate delivery is acceptable internally if visible search/profile results remain deduplicated.

## Current Risk

The audit found these ingestion risks:

- `bufferedRawWriter.Submit` can drop the oldest event or the incoming event when the memory buffer is full.
- ClickHouse flush retries can eventually drop a whole batch after the retry limit.
- `RunOnce` can increment stored counters after a memory submit, before durable storage is guaranteed.
- The parser can discard incomplete `ChatMessageEvent` payloads before raw payload archival.
- Some old SQLite queue/claim abstractions still make the data path harder to reason about.

These are acceptable only as temporary implementation history. They are not acceptable as the final architecture.

## Target Runtime Shape

The production stack should contain these services:

- `nats`: NATS JetStream with file storage and a persistent Docker volume.
- `clickhouse`: long-lived event, message, analytics, and subscription data.
- `api`: HTTP API and admin/search/profile endpoints.
- `listener`: Kick Pusher capture service. It subscribes to Kick and publishes raw events to JetStream.
- `processor`: JetStream consumer service. It normalizes raw events and writes ClickHouse batches.
- `web`: Next.js frontend.

Listener and processor are intentionally separate. Listener priority is staying connected to Kick and acknowledging JetStream publishes. Processor priority is batching, retrying, and writing to ClickHouse efficiently.

## Storage Ownership

### NATS JetStream

JetStream owns temporary durable backlog for unprocessed raw chat events.

It must not become a long-term analytics database. Acked events should leave the backlog according to stream retention.

### ClickHouse

ClickHouse owns the durable data plane:

- `raw_kick_events`
- `chat_messages`
- raw event failure/terminal ignored audit data, if still needed
- analytics tables/views
- subscription period data

ClickHouse is the source for search, user profile pages, channel profile pages, analytics, and admin data size summaries.

### SQLite

SQLite owns control-plane state only:

- admin users
- followed channels
- sender profile cache
- retention settings
- worker/processor heartbeat rows
- webhook subscription registry and webhook inbox state
- data migration metadata

SQLite must not be used as the live chat raw-event queue after this plan is complete.

Existing `raw_event_queue` and `raw_event_claims` tables may remain for compatibility during migration, but they must not be active ingestion dependencies.

## JetStream Design

Initial stream design:

- Stream name: `KICK_RAW_EVENTS`
- Subject: `kick.raw.chat`
- Storage: file
- Retention: backlog/work-queue style retention for unacked events
- Consumer: durable pull consumer for processor workers
- Ack policy: explicit ACK
- Redelivery: enabled through ack wait / max delivery policy

JetStream must not be configured to silently discard old messages under normal pressure. If stream size limits are hit, that is a critical operational problem, not an acceptable data-loss policy.

Recommended raw event envelope:

```json
{
  "raw_event_id": "deterministic-or-generated-id",
  "kick_message_id": "message-id-if-present",
  "event_name": "App\\Events\\ChatMessageEvent",
  "pusher_channel": "chatrooms.123.v2",
  "followed_channel_id": 1,
  "channel_slug": "example",
  "kick_channel_id": 123,
  "kick_chatroom_id": 456,
  "received_at": "2026-06-02T12:00:00Z",
  "body": {}
}
```

Idempotency key order:

1. Kick message id, when present.
2. Deterministic hash of stable raw event fields when Kick message id is missing.
3. Generated id only for event types where no natural id exists and duplicate visibility is impossible or handled elsewhere.

## Listener Capture Behavior

The listener should:

- resolve enabled followed channels,
- subscribe to Kick Pusher channels,
- avoid periodic reconnects,
- reconnect only on websocket failure or actual followed-channel set changes,
- publish every reached `ChatMessageEvent` raw payload to JetStream,
- wait for JetStream PubAck before counting the event as captured,
- surface publish failure as an operational failure,
- avoid memory-buffer drop policies entirely.

The listener should not require all normalized message fields before durable capture. Raw capture happens first; validation and terminal invalid classification happen in the processor.

Listener metrics should include:

- websocket connected status,
- subscribed channel count,
- last Kick event received time,
- last JetStream publish ack time,
- publish failure count,
- current stream/backlog health if cheaply available.

## Processor Behavior

The processor should:

- pull batches from the durable JetStream consumer,
- batch insert raw payloads into ClickHouse,
- normalize valid chat messages,
- batch insert normalized `chat_messages`,
- record terminal invalid/ignored events when the raw payload cannot become a visible message,
- ACK JetStream messages only after required durable writes succeed,
- avoid per-success attempt bloat,
- rely on redelivery for transient ClickHouse failures.

Terminal invalid events should be ACKed only after their terminal status is durable enough for diagnosis. Transient failures must remain unacked so JetStream can redeliver them.

## Idempotency Requirements

At-least-once delivery means duplicate processing can happen. The visible product must remain stable.

Required behavior:

- duplicate raw delivery must not create duplicate visible chat rows,
- `/search` results must not show duplicate messages,
- `/users/{slug}` and `/channels/{slug}` timelines must not show duplicate messages,
- analytics should count unique messages according to the chosen message identity,
- retries and processor restarts must be safe.

Implementation options:

- deterministic `raw_event_id`,
- ClickHouse `ReplacingMergeTree` or equivalent read-side dedupe,
- query-time `argMax`/latest row selection where needed,
- strict `kick_message_id`-based message identity for visible `chat_messages`.

The chosen approach must be tested with redelivery scenarios.

## Operations And Admin Visibility

Admin Operations should show the new ingestion health clearly:

- listener heartbeat,
- processor heartbeat,
- JetStream stream pending count,
- consumer pending count,
- ack-pending count,
- redelivery count,
- oldest pending event age,
- ClickHouse latest raw event time,
- ClickHouse latest visible message time,
- processor insert failure count.

Old SQLite raw queue tables should not be presented as the active ingestion queue once JetStream is live. If they remain visible, they must be marked as legacy/internal.

Any event-drop metric should be treated as critical. A non-zero value means a bug or infrastructure failure, not expected operation.

## Migration And Cutover

Historical ClickHouse data must not be deleted or rewritten as part of the first JetStream cutover.

Existing tables remain queryable:

- `chat_messages`
- `raw_kick_events`
- `raw_event_attempts`
- subscription-related tables

Cutover strategy:

1. Add NATS JetStream service and volume.
2. Add publisher/consumer ports and infrastructure implementations.
3. Add processor service while keeping public API unchanged.
4. Route listener raw capture through JetStream.
5. Verify processor writes ClickHouse batches and search/profile results stay unchanged.
6. Disable old memory buffered writer and SQLite raw queue usage in live ingestion.
7. Leave old queue tables in place until a later cleanup issue.

Short deployment downtime is acceptable if explicitly chosen, but normal runtime must not intentionally disconnect from Kick on a timer.

Old `raw_event_queue` rows should be handled one of these ways:

- drain them before enabling JetStream-only live ingestion,
- migrate them once into JetStream,
- or keep them as legacy rows if they are already terminal/processed and no longer needed.

The chosen path must be documented before production deploy.

## Earlier Issue 23 Work

The earlier branch work remains useful as preliminary hardening:

- sender profile writes became less critical to message ingestion,
- sender profile cache writes were throttled,
- processed/terminal queue cleanup was clarified,
- webhook inbox retention was added,
- admin copy around operational tables was improved.

That work does not replace the JetStream ingestion cutover. The final architecture must remove intentional chat-event drops from the live path.

## Implementation Phases

### Phase 1 - Plan And Context

- Replace this implementation plan with the JetStream durable ingestion plan.
- Update context docs only where they describe old intentional reconnect/drop behavior as acceptable.
- Keep this phase docs-only.

Exit criteria:

- The plan clearly states NATS JetStream as the target hot path.
- SQLite is documented as control-plane only.
- No runtime code changes are included in this phase unless explicitly requested.

### Phase 2 - NATS Foundation

- Add NATS Go dependency.
- Add `nats` service to `compose.yaml` with persistent JetStream volume.
- Add environment/config values for NATS URL, stream name, subject, consumer name, ack wait, batch size, and fetch timeout.
- Create application ports for raw event publishing, raw event consuming, and stream stats.
- Add infrastructure package for JetStream connection, stream setup, publisher, consumer, and stats.
- Keep listener behavior unchanged until the publisher is tested behind an interface.

Exit criteria:

- App can create/verify the stream and durable consumer.
- Unit tests cover config defaults and publisher/consumer construction with fakes where possible.
- Docker Compose can start NATS without breaking existing services.

### Phase 3 - Raw Event Envelope

- Introduce a stable raw event envelope type.
- Preserve raw Pusher payload before strict message normalization.
- Make incomplete `ChatMessageEvent` payloads captureable.
- Move invalid/ignored classification to processor logic.
- Define deterministic identity rules for raw events and visible chat messages.

Exit criteria:

- Tests prove incomplete chat payloads are not discarded before durable capture.
- Message identity fallback rules are covered by tests.
- Existing valid message parsing behavior remains compatible.

### Phase 4 - Listener Publishes To JetStream

- Replace live listener memory submission with JetStream publish.
- Wait for PubAck before incrementing capture counters.
- Treat publish failure as an error that triggers visible reconnect/backoff behavior.
- Keep channel resync logic based on actual channel set changes, not periodic forced reconnect.
- Remove or disable memory buffer drop metrics in the listener path.

Exit criteria:

- Listener tests prove events are counted only after publish ack.
- Publish failure does not silently drop events.
- No configured buffer-full policy can discard a reached chat event.

### Phase 5 - Processor Service

- Add `cmd/processor`.
- Pull JetStream messages in batches.
- Insert raw events into ClickHouse in batches.
- Normalize valid messages and insert `chat_messages` in batches.
- Write terminal invalid/ignored records when needed.
- ACK only after required durable writes succeed.
- Do not ACK transient ClickHouse failures.

Exit criteria:

- Processor tests cover success, transient failure/redelivery, and terminal invalid handling.
- Batch insert behavior remains available under load.
- Processor can run independently from listener.

### Phase 6 - ClickHouse Idempotency

- Review `raw_kick_events` and `chat_messages` table engines/keys.
- Add deterministic dedupe behavior for visible messages.
- Ensure retries/redeliveries do not create duplicate search/profile rows.
- Keep existing API response shapes unchanged.

Exit criteria:

- Tests prove duplicate redelivery does not create duplicate visible messages.
- Search, user profile, channel profile, and analytics queries remain compatible.
- Existing production data can continue to be queried.

### Phase 7 - Remove Old Hot Path Usage

- Stop writing live chat events to SQLite `raw_event_queue`.
- Stop using `raw_event_claims` for live chat ingestion.
- Remove or deprecate `bufferedRawWriter` drop behavior from active listener code.
- Update loadgen and internal wiring to exercise JetStream capture + processor.

Exit criteria:

- Live chat ingestion no longer depends on SQLite queue tables.
- No active code path intentionally drops chat events after they reach the listener.
- Legacy tables are either unused or clearly marked for later cleanup.

### Phase 8 - Admin Operations Update

Status: implemented in this branch.

- Add JetStream and processor health to admin Operations.
- Show backlog size, ack-pending count, redelivery count, oldest pending age, and processor heartbeat.
- Remove misleading active-queue wording for old SQLite raw queue tables.
- Keep failed/terminal raw event visibility where useful.

Exit criteria:

- Admin users can understand whether ingestion is capturing, backlogged, redelivering, or failing.
- The UI does not imply SQLite is still the live chat queue.

### Phase 9 - Documentation And Verification

Status: documentation updated; verification passed in this branch.

- Update architecture docs, context docs, README operational notes, and deployment notes.
- Document NATS volume backup/restore expectations.
- Document production cutover and rollback steps.
- Run backend tests, frontend checks, formatting, and Docker Compose smoke tests.

Exit criteria:

- CI-relevant checks pass.
- Compose smoke proves listener -> NATS -> processor -> ClickHouse works.
- Load test proves burst traffic increases backlog instead of dropping events, then drains.

## Test Plan

Required unit tests:

- incomplete `ChatMessageEvent` is captured before validation,
- listener counts an event only after PubAck,
- publish failure is not treated as successful capture,
- processor ACKs only after raw and visible writes succeed,
- transient ClickHouse failure causes redelivery,
- terminal invalid event becomes durable diagnostic state and then ACKs,
- duplicate redelivery does not duplicate visible chat messages.

Required integration or smoke tests:

- NATS JetStream starts through Docker Compose,
- stream and durable consumer are created,
- listener can publish a sample raw event,
- processor consumes and writes ClickHouse rows,
- `/search` can read the inserted message,
- burst/loadgen shows zero intentional drops and eventual backlog drain.

Required production checks:

- backup current volumes before cutover,
- deploy NATS volume,
- verify listener heartbeat,
- verify processor heartbeat,
- verify JetStream backlog metrics,
- verify latest raw event time advances,
- verify latest visible message time advances,
- verify existing historical search still works.

## Out Of Scope

- Changing public API response bodies.
- Redesigning frontend search/profile pages.
- Deleting historical ClickHouse data.
- Removing old SQLite queue tables immediately.
- Building multi-node leader election for listeners.
- Reworking webhook subscription architecture.
- Storing long-term raw history in NATS.
