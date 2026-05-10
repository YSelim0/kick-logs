# Phase 8 Tasks: Public Search UI

## Scope

Implement the public `/search` screen. This phase owns search form, URL/query state, public message fetching, infinite scroll, dense result rows, circular avatars, and inline emotes.

Do not implement admin dashboard workflows, login UI beyond shared layout needs, backend changes, or landing page content.

## Inputs

- Completed Phase 7 frontend foundation.
- Public `GET /messages` backend contract.
- `docs/design/design.md` and `docs/design/design.pen`.

## Tasks

- [x] Public route behavior:
  - [x] `/search` requires no login.
  - [x] Do not show admin-only controls.
  - [x] Keep `/search` usable when no auth cookie exists.
- [x] Search form:
  - [x] Fields: `Kullanıcı Adı`, `Kanal Adı`, `Aramak istediğiniz Kelime`, `Başlangıç`, `Bitiş`.
  - [x] Map fields to `sender`, `channel`, `q`, `start`, and `end`.
  - [x] Omit empty fields from query params.
  - [x] Preserve submitted filter state in URL or local route state consistently.
  - [x] Default `Başlangıç` to 7 days before current local date/time and `Bitiş` to current local date/time.
  - [x] Use yellow `#FFF600` primary `Ara` button.
- [x] Results fetching:
  - [x] Fetch newest-first public messages.
  - [x] Use backend cursor pagination.
  - [x] Implement infinite scroll loading.
  - [x] Handle empty, loading, and error states compactly.
- [x] Result rows:
  - [x] Render rows inside one shared outer list container.
  - [x] Do not create one card/modal component per message.
  - [x] Fixed metadata columns: circular avatar, sender, channel, timestamp.
  - [x] Flexible message column expands/wraps for long content.
  - [x] Render sender profile image when available.
  - [x] Use circular fallback avatar otherwise.
- [x] Emotes:
  - [x] Parse message content tokens or use backend parsed emotes.
  - [x] Render emotes inline at their message positions.
  - [x] Use `https://files.kick.com/emotes/{id}/fullsize`.
  - [x] Fall back to emote name/token on image failure.
- [x] Visual rules:
  - [x] Match dark-only palette.
  - [x] No blur, glow, colored lighting, or hero layout.
  - [x] Modest radii.
  - [x] No landing page content.
- [x] Tests/checks:
  - [x] Search form query mapping.
  - [x] Empty filters fetch latest messages.
  - [x] Infinite scroll appends rows.
  - [x] Emote fallback rendering.
  - [x] Public route does not redirect to login.

## Acceptance Criteria

- [x] Any visitor can use `/search` without auth.
- [x] All documented filter combinations produce correct query params.
- [x] Results render as dense rows with circular avatars and inline emotes.
- [x] Frontend typecheck/build passes.
- [x] No admin dashboard implementation is introduced.

## Handoff

Phase 9 can add authenticated admin workflows without changing public search behavior.
