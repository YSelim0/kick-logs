# Go Rewrite Phase 6: Listener And Ingestion

## Scope

Rebuild the Kick listener and ingestion pipeline in Go.

This phase owns websocket subscription, raw-event durability, event parsing, emote/reply parsing,
sender enrichment, dedupe, retry, and listener heartbeat behavior.

## Out Of Scope

- Do not implement analytics/profile APIs here.
- Do not implement PostgreSQL data migration here.
- Do not change frontend behavior.
- Do not remove Python listener until cutover.

## Checklist

- [x] Implement Kick channel resolver equivalent to the current `https://kick.com/api/v2/channels`
      behavior.
- [x] Implement Kick sender profile resolver when sender slug metadata is available.
- [x] Implement Pusher websocket client.
- [x] Subscribe to `chatrooms.{chatroom_id}.v2`.
- [x] Subscribe to required channel-level streams when needed.
- [x] Load enabled followed channels from SQLite.
- [x] Resolve missing channel metadata before subscription.
- [x] Periodically resync followed channels and reconnect when subscription set changes.
- [x] Parse `App\Events\ChatMessageEvent`.
- [x] Insert raw events into ClickHouse before normalization.
- [x] Parse content, sender fields, badges, identity color, message type, timestamps, reply
      metadata, thread parent id, and raw payload.
- [x] Parse emotes into response-compatible fields and ClickHouse helper arrays.
- [x] Upsert sender profile cache in SQLite.
- [x] Insert normalized messages into ClickHouse idempotently.
- [x] Append raw-event processing attempt rows.
- [x] Retry raw events that were stored but not normalized.
- [x] Record listener heartbeat in SQLite.
- [x] Reconnect with backoff after websocket failures.

## Tests And Checks

- [x] Parser tests cover normal chat messages.
- [x] Parser tests cover reply messages and reply metadata.
- [x] Parser tests cover messages with emotes.
- [x] Parser tests cover missing optional sender/channel fields.
- [x] Ingestion tests prove raw event is written before message normalization.
- [x] Ingestion tests prove duplicate `kick_message_id` does not create duplicate visible
      messages.
- [x] Recovery tests process raw events that lack normalized messages.
- [x] Listener tests prove enabled-channel resync changes subscriptions.
- [x] Heartbeat tests update freshness state.

## Acceptance Criteria

- [x] A followed live channel can be subscribed through the Go listener.
- [x] Live messages become searchable through the Go API.
- [x] Reply and emote rendering data remains compatible with the frontend.
- [x] Admin operations can show listener freshness and raw-event processing health.

## Commit Boundary

Commit listener ingestion parity separately from analytics/profile work.
