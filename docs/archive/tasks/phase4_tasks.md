# Phase 4 Tasks: Channels, Message Search, Ingestion Use Cases

## Scope

Implement the backend product APIs except the live listener runtime. This phase owns admin channel management, Kick channel metadata resolution, message ingestion use cases, emote parsing, and public message search.

Do not implement websocket/Pusher listener loops, frontend screens, or Docker web service.

## Inputs

- Completed Phase 3 auth/admin user foundation.
- Kick channel resolver decision from `docs/project_plan.md`.
- Public search contract from `docs/architecture.md`.

## Tasks

- [x] Kick channel resolver:
  - [x] Add application port for resolving channel slug/nickname.
  - [x] Implement resolver with Kick web endpoint `https://kick.com/api/v2/channels/{slug}`.
  - [x] Extract Kick channel id, chatroom id, slug/display metadata, profile image/banner when available, and raw payload.
  - [x] Fail gracefully when Kick endpoint changes or returns an error.
- [x] Admin channel use cases:
  - [x] List followed channels.
  - [x] Add channel by slug/nickname; resolve metadata before persisting.
  - [x] Re-enable existing disabled channel when re-added.
  - [x] Disable or remove channel through `DELETE /admin/channels/{id}` according to MVP behavior.
- [x] Admin channel routes:
  - [x] `GET /admin/channels`
  - [x] `POST /admin/channels`
  - [x] `DELETE /admin/channels/{id}`
  - [x] Require authenticated admin or super admin.
- [x] Message ingestion use case:
  - [x] Normalize Kick chat payload into sender + chat message records.
  - [x] Deduplicate by Kick message id.
  - [x] Store raw payload JSONB.
  - [x] Store sender snapshots, badges, reply metadata, thread parent id, and message timestamp.
- [x] Emote parser:
  - [x] Parse `[emote:id:name]` tokens.
  - [x] Store `id`, `name`, original token, and inferred image URL.
  - [x] Keep original message content unchanged for search/display.
- [x] Public search use case and route:
  - [x] `GET /messages?sender=&channel=&q=&start=&end=&cursor=&limit=`
  - [x] No authentication required.
  - [x] Optional filters combine with `AND`.
  - [x] Case-insensitive contains for sender, channel, and content.
  - [x] Date filtering on `message_created_at`.
  - [x] Newest-first ordering.
  - [x] Cursor pagination based on `(message_created_at, id)`.
  - [x] Return sender avatar/profile fields and parsed emotes for frontend rendering.
- [x] Tests:
  - [x] Channel resolver success/failure with mocked HTTP.
  - [x] Admin channel add/list/delete auth checks.
  - [x] Emote parser cases.
  - [x] Ingest idempotency.
  - [x] Search filter combinations and pagination.

## Acceptance Criteria

- [x] Admins can manage followed channels through API.
- [x] Public message search works without login.
- [x] Ingest use case can persist normalized messages without listener runtime.
- [x] Search tests cover all documented filter combinations.
- [x] Docs/context are updated with implemented API details.

## Handoff

Phase 5 can call existing channel loading, event parsing, and ingestion use cases from the listener without duplicating persistence logic.
