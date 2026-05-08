# Living Brain

This file is the active project memory. Keep it updated whenever project behavior, architecture, implementation details, or working assumptions change.

## Current State

- Repository `kick-logs` has been initialized locally.
- Commit convention skill exists under `.agents/skills/commit-message-conventions`.
- The first local commit was created:
  - `679d936 feat(repo): add commit convention skill`
- The project implementation has not been scaffolded yet.

## Kick Chat Ingestion Method

The MVP listener should implement this self-contained Kick web chat ingestion flow:

- Use `curl_cffi` with browser impersonation to resolve `https://kick.com/api/v2/channels/{slug}`.
- Read Kick `channel_id` from response `id`.
- Read Kick `chatroom_id` from response `chatroom.id`.
- Connect to:
  - `wss://ws-us2.pusher.com/app/32cbd69e4b950bf97679?protocol=7&client=js&version=8.4.0-rc2&flash=false`
- Subscribe to:
  - `chatrooms.{chatroom_id}.v2`
  - `channel.{channel_id}`
- Handle event:
  - `App\Events\ChatMessageEvent`
- Extract sender username from `payload.sender.username`.
- Extract message content from `payload.content`.

## Product Direction

Build an MVP monorepo with:

- Python backend
- Next.js frontend
- PostgreSQL persistence
- Docker Compose local runtime
- Admin channel management
- Searchable historical Kick chat logs

## Locked Product Decisions

- Store messages indefinitely.
- Use a full login system with `super_admin` and `admin` roles.
- Seed default super admin:
  - email: `admin@kicklogs.local`
  - password: `admin123`
- Allow env override for default super admin credentials.
- Use `/search` for the app search screen, `/admin` for admin, and reserve `/` for a future landing page.
- Search filters are optional and combined with `AND`:
  - sender nickname
  - channel nickname/slug
  - message content
  - start datetime
  - end datetime
- Use case-insensitive contains matching for sender, channel, and message content.
- Use one listener worker/container to subscribe to all enabled channels.
- Store all useful available data, including normalized fields, parsed emotes, sender badges, profile image when enriched, reply metadata, and raw payload JSONB.
- Render emotes with `https://files.kick.com/emotes/{id}/fullsize` and fall back to the emote name/token if the image fails.

## Operational Rules

- Every agent must read `AGENTS.md` and context files before making changes.
- Keep documentation and context current with implementation changes.
- Update `docs/context/recent_changes.md` with a short latest-change handoff after each meaningful change.
- Commit after each completed unit of work when requested.
- User will manually push commits.
