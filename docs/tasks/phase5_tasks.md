# Phase 5 Tasks: Kick Listener Worker

## Scope

Implement the live Kick chat ingestion worker. This phase owns worker entrypoint, Pusher websocket client, event parsing, reconnect policy, sender enrichment, and listener Docker service.

Do not change public search UI, admin UI, auth contracts, or database schema except for narrowly required listener metadata additions documented before implementation.

## Inputs

- Completed Phase 4 channel, message, and ingestion use cases.
- Kick listener method from `docs/context/living_brain.md`.
- Listener runtime shape from `docs/architecture.md`.

## Tasks

- [ ] Listener service composition:
  - [ ] Add `presentation/worker/main.py`.
  - [ ] Add `presentation/worker/listener_service.py`.
  - [ ] Wire settings, unit of work, channel resolver, event parser, sender resolver, and ingestion use case.
- [ ] Load enabled channels:
  - [ ] Query enabled channels from DB.
  - [ ] Resolve missing Kick channel id/chatroom id before subscribing.
  - [ ] Skip disabled or unresolvable channels with structured logs.
- [ ] Pusher client:
  - [ ] Connect to `wss://ws-us2.pusher.com/app/32cbd69e4b950bf97679?protocol=7&client=js&version=8.4.0-rc2&flash=false`.
  - [ ] Subscribe to `chatrooms.{chatroom_id}.v2`.
  - [ ] Subscribe to `channel.{channel_id}` only if required by implemented behavior.
  - [ ] Handle `App\Events\ChatMessageEvent`.
- [ ] Event parser:
  - [ ] Parse JSON payloads safely.
  - [ ] Extract message id, chatroom id, content, type, created_at, sender fields, identity color, badges, metadata, reply fields, and thread parent id.
  - [ ] Reject malformed events without crashing the worker.
- [ ] Sender profile enrichment:
  - [ ] Add sender profile resolver by sender slug.
  - [ ] Cache/store profile image URL when available.
  - [ ] Continue ingestion if enrichment fails.
- [ ] Reconnect policy:
  - [ ] Backoff after websocket failures.
  - [ ] Re-subscribe enabled channels after reconnect.
  - [ ] Log connection, subscription, parse, and ingest events.
- [ ] Docker Compose:
  - [ ] Add `listener` service using same backend image/source as `api`.
  - [ ] Ensure listener depends on `postgres` and uses backend env.
- [ ] Tests:
  - [ ] Event parser with representative Kick payloads.
  - [ ] Listener service with fake Pusher client and fake repositories.
  - [ ] Reconnect policy unit tests.
  - [ ] Sender enrichment fallback.

## Acceptance Criteria

- [ ] Listener can ingest mocked Kick chat events into DB through existing use case.
- [ ] Listener Docker service starts without breaking API.
- [ ] Malformed events and transient network errors do not crash permanently.
- [ ] No frontend work is introduced.
- [ ] Docs/context are updated with implemented listener behavior.

## Handoff

Phase 6 can verify the complete backend system: API, database, admin channel management, listener ingestion, and public search.
