# Implementation Plan

## Active Feature: Frontend v2 Re-skin

Re-skin the entire web app to the v2 design captured in `docs/design/design.pen` and documented in
`docs/design/design.md`. Backend stays untouched — this is purely a frontend visual + structural
update against existing APIs.

### Scope

- New palette (Kick green accent on dark neutrals), Geist Sans + Geist Mono typography,
  Vercel-influenced minimal layout.
- Global header simplified: brand + active route pill + GitHub icon + Admin button.
  `Channels` / `Users` nav links removed until those index pages exist.
- All six routes (`/`, `/search`, `/users/[slug]`, `/channels/[slug]`, `/login`, `/admin`)
  re-styled against the v2 designs in `docs/design/design.pen`.

### Approach

1. **Design tokens.** Replace the legacy magenta palette with the new tokens in
   `apps/web/tailwind.config.{ts,js}` and `apps/web/src/app/globals.css`. Wire Geist Sans + Geist
   Mono via `next/font` and apply through `font-sans` / `font-mono` Tailwind classes. Drop the old
   `kick-*` color names where they leak into components.
2. **Shared chrome.** Update the top header component (and admin variant) to match the v2 layout.
   Remove `Channels` and `Users` nav links from public chrome.
3. **Search.** Re-style the form panel, status strip, result rows, and load-more loader. Remove
   sender avatar from result rows. Render sender username with `sender_color_snapshot` (fallback
   white). Keep `#channelSlug` rendered in accent green. Render emotes inline using
   `https://files.kick.com/emotes/{id}/fullsize` with the existing fallback chip. Drop the
   `AND filtreleme` suffix from the status strip. Replace the bottom load-more strip with a
   centered inline `● daha eski mesajlar yükleniyor…` indicator.
4. **Landing.** Apply the compact hero, 4-cell stats bar, and 2×2 analytics grid against existing
   analytics endpoints.
5. **User profile.** Re-style identity panel, stats bar, 3-column analytics grid (equal panel
   heights), latest messages list with the new sender-color + inline-emote rules.
6. **Channel profile.** Same shape as user profile, with channel identity, `LOGGING` accent pill,
   and `Kanalda ara` CTA.
7. **Admin.** Move from the existing two-column "main + right session panel" to the v2 sidebar
   layout (`Operations`, `Channels`, `Users`, `Data`, `Settings`). Re-style the operations status
   banner, metric cards, ingestion strip, channel table, and existing data management UI to match.
   Keep all current behaviors (failed-events modal, retention/cleanup confirmation flow,
   super-admin-only user management).
8. **Login.** Centered v2 card with Geist headings and the existing email/password form.

### Progress

- [x] Step 1 (design tokens) — v2 tokens + Geist wired (2026-05-21)
- [x] Step 2 (shared chrome) — `SiteHeader` component landed; consumed by landing
- [ ] Step 3 (search)
- [x] Step 4 (landing) — `mRzu8` re-skin shipped (2026-05-21)
- [ ] Step 5 (user profile)
- [ ] Step 6 (channel profile)
- [ ] Step 7 (admin)
- [ ] Step 8 (login)

### Out of Scope (Now)

- `/channels` and `/users` index/listing pages. Header links stay removed until these exist.
- New backend endpoints or schema changes.
- Mobile redesign beyond keeping inputs/buttons usable.
- Theme switcher (still dark-only).

### Risks

- Legacy `kick-*` color tokens are referenced in many components; removing them needs a sweep to
  avoid leftover magenta. Do a `kick-background` / `kick-primary` grep before merging.
- Sender color (`sender_color_snapshot`) values may be missing or invalid; need a safe parser
  with white fallback to avoid blown layouts.
- Geist font loading: load through `next/font/google` (or the official `geist` package) so SSR
  hydration matches and CSS variables are stable.
- Equal-height panels in the 3-column grid depend on chart height matching the list panel; tune
  with CSS grid `auto-rows: 1fr` rather than hardcoded pixel heights.

### Verification

- Run `pnpm lint`, `pnpm typecheck`, `pnpm test` in `apps/web`.
- Manual smoke test against the real backend in Docker for every route on desktop width.
- Confirm dark-only rendering (no flashes of light theme during hydration).
- Confirm failed-events modal still opens via the clickable `İnceleme gerekli` text on the new
  metric card.

## Completed Plans

- Issue #11: Worker hot-path optimizations (sender resolver removal + batch existence + batch
  raw-event fetch). PR #12 merged.
- Issue #9: Stabilize high-volume Kick chat ingestion with ClickHouse batching and backpressure.
  Archived under `docs/archive/issue_09/`.
- Go + ClickHouse rewrite (phases 1-9). Archived under `docs/archive/go_rewrite/`.
- Post-MVP features (features 1-8). Archived under `docs/archive/post_mvp/`.
- MVP (phases 1-10). Archived under `docs/archive/mvp/`.
