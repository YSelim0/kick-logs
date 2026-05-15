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

- [ ] Implement Kick channel resolver equivalent to the current `https://kick.com/api/v2/channels`
      behavior.
- [ ] Implement Kick sender profile resolver when sender slug metadata is available.
- [ ] Implement Pusher websocket client.
- [ ] Subscribe to `chatrooms.{chatroom_id}.v2`.
- [ ] Subscribe to required channel-level streams when needed.
- [ ] Load enabled followed channels from SQLite.
- [ ] Resolve missing channel metadata before subscription.
- [ ] Periodically resync followed channels and reconnect when subscription set changes.
- [ ] Parse `App\Events\ChatMessageEvent`.
- [ ] Insert raw events into ClickHouse before normalization.
- [ ] Parse content, sender fields, badges, identity color, message type, timestamps, reply
      metadata, thread parent id, and raw payload.
- [ ] Parse emotes into response-compatible fields and ClickHouse helper arrays.
- [ ] Upsert sender profile cache in SQLite.
- [ ] Insert normalized messages into ClickHouse idempotently.
- [ ] Append raw-event processing attempt rows.
- [ ] Retry raw events that were stored but not normalized.
- [ ] Record listener heartbeat in SQLite.
- [ ] Reconnect with backoff after websocket failures.

## Tests And Checks

- [ ] Parser tests cover normal chat messages.
- [ ] Parser tests cover reply messages and reply metadata.
- [ ] Parser tests cover messages with emotes.
- [ ] Parser tests cover missing optional sender/channel fields.
- [ ] Ingestion tests prove raw event is written before message normalization.
- [ ] Ingestion tests prove duplicate `kick_message_id` does not create duplicate visible
      messages.
- [ ] Recovery tests process raw events that lack normalized messages.
- [ ] Listener tests prove enabled-channel resync changes subscriptions.
- [ ] Heartbeat tests update freshness state.

## Acceptance Criteria

- [ ] A followed live channel can be subscribed through the Go listener.
- [ ] Live messages become searchable through the Go API.
- [ ] Reply and emote rendering data remains compatible with the frontend.
- [ ] Admin operations can show listener freshness and raw-event processing health.

## Commit Boundary

Commit listener ingestion parity separately from analytics/profile work.
