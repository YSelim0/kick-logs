# Phase 4 Tasks: Channels, Message Search, Ingestion Use Cases

## Scope

Implement the backend product APIs except the live listener runtime. This phase owns admin channel management, Kick channel metadata resolution, message ingestion use cases, emote parsing, and public message search.

Do not implement websocket/Pusher listener loops, frontend screens, or Docker web service.

## Inputs

- Completed Phase 3 auth/admin user foundation.
- Kick channel resolver decision from `docs/project_plan.md`.
- Public search contract from `docs/architecture.md`.

## Tasks

- [ ] Kick channel resolver:
  - [ ] Add application port for resolving channel slug/nickname.
  - [ ] Implement resolver with Kick web endpoint `https://kick.com/api/v2/channels/{slug}`.
  - [ ] Extract Kick channel id, chatroom id, slug/display metadata, profile image/banner when available, and raw payload.
  - [ ] Fail gracefully when Kick endpoint changes or returns an error.
- [ ] Admin channel use cases:
  - [ ] List followed channels.
  - [ ] Add channel by slug/nickname; resolve metadata before persisting.
  - [ ] Re-enable existing disabled channel when re-added.
  - [ ] Disable or remove channel through `DELETE /admin/channels/{id}` according to MVP behavior.
- [ ] Admin channel routes:
  - [ ] `GET /admin/channels`
  - [ ] `POST /admin/channels`
  - [ ] `DELETE /admin/channels/{id}`
  - [ ] Require authenticated admin or super admin.
- [ ] Message ingestion use case:
  - [ ] Normalize Kick chat payload into sender + chat message records.
  - [ ] Deduplicate by Kick message id.
  - [ ] Store raw payload JSONB.
  - [ ] Store sender snapshots, badges, reply metadata, thread parent id, and message timestamp.
- [ ] Emote parser:
  - [ ] Parse `[emote:id:name]` tokens.
  - [ ] Store `id`, `name`, original token, and inferred image URL.
  - [ ] Keep original message content unchanged for search/display.
- [ ] Public search use case and route:
  - [ ] `GET /messages?sender=&channel=&q=&start=&end=&cursor=&limit=`
  - [ ] No authentication required.
  - [ ] Optional filters combine with `AND`.
  - [ ] Case-insensitive contains for sender, channel, and content.
  - [ ] Date filtering on `message_created_at`.
  - [ ] Newest-first ordering.
  - [ ] Cursor pagination based on `(message_created_at, id)`.
  - [ ] Return sender avatar/profile fields and parsed emotes for frontend rendering.
- [ ] Tests:
  - [ ] Channel resolver success/failure with mocked HTTP.
  - [ ] Admin channel add/list/delete auth checks.
  - [ ] Emote parser cases.
  - [ ] Ingest idempotency.
  - [ ] Search filter combinations and pagination.

## Acceptance Criteria

- [ ] Admins can manage followed channels through API.
- [ ] Public message search works without login.
- [ ] Ingest use case can persist normalized messages without listener runtime.
- [ ] Search tests cover all documented filter combinations.
- [ ] Docs/context are updated with implemented API details.

## Handoff

Phase 5 can call existing channel loading, event parsing, and ingestion use cases from the listener without duplicating persistence logic.
