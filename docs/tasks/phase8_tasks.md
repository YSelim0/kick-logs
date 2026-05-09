# Phase 8 Tasks: Public Search UI

## Scope

Implement the public `/search` screen. This phase owns search form, URL/query state, public message fetching, infinite scroll, dense result rows, circular avatars, and inline emotes.

Do not implement admin dashboard workflows, login UI beyond shared layout needs, backend changes, or landing page content.

## Inputs

- Completed Phase 7 frontend foundation.
- Public `GET /messages` backend contract.
- `docs/design/design.md` and `docs/design/design.pen`.

## Tasks

- [ ] Public route behavior:
  - [ ] `/search` requires no login.
  - [ ] Do not show admin-only controls.
  - [ ] Keep `/search` usable when no auth cookie exists.
- [ ] Search form:
  - [ ] Fields: `Kullanıcı Adı`, `Kanal Adı`, `Aramak istediğiniz Kelime`, `Başlangıç`, `Bitiş`.
  - [ ] Map fields to `sender`, `channel`, `q`, `start`, and `end`.
  - [ ] Omit empty fields from query params.
  - [ ] Preserve submitted filter state in URL or local route state consistently.
  - [ ] Use yellow `#FFF600` primary `Ara` button.
- [ ] Results fetching:
  - [ ] Fetch newest-first public messages.
  - [ ] Use backend cursor pagination.
  - [ ] Implement infinite scroll loading.
  - [ ] Handle empty, loading, and error states compactly.
- [ ] Result rows:
  - [ ] Render rows inside one shared outer list container.
  - [ ] Do not create one card/modal component per message.
  - [ ] Fixed metadata columns: circular avatar, sender, channel, timestamp.
  - [ ] Flexible message column expands/wraps for long content.
  - [ ] Render sender profile image when available.
  - [ ] Use circular fallback avatar otherwise.
- [ ] Emotes:
  - [ ] Parse message content tokens or use backend parsed emotes.
  - [ ] Render emotes inline at their message positions.
  - [ ] Use `https://files.kick.com/emotes/{id}/fullsize`.
  - [ ] Fall back to emote name/token on image failure.
- [ ] Visual rules:
  - [ ] Match dark-only palette.
  - [ ] No blur, glow, colored lighting, or hero layout.
  - [ ] Modest radii.
  - [ ] No landing page content.
- [ ] Tests/checks:
  - [ ] Search form query mapping.
  - [ ] Empty filters fetch latest messages.
  - [ ] Infinite scroll appends rows.
  - [ ] Emote fallback rendering.
  - [ ] Public route does not redirect to login.

## Acceptance Criteria

- [ ] Any visitor can use `/search` without auth.
- [ ] All documented filter combinations produce correct query params.
- [ ] Results render as dense rows with circular avatars and inline emotes.
- [ ] Frontend typecheck/build passes.
- [ ] No admin dashboard implementation is introduced.

## Handoff

Phase 9 can add authenticated admin workflows without changing public search behavior.
