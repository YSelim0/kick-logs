# Kick Logs UI Design Guide

This document is the source of truth for UI and UX decisions. Update it whenever a later
conversation changes search UI, admin UI, visual style, routing, or frontend behavior.

## Design Sources

- `docs/design/design.md`: human-readable UI rules and decisions.
- `docs/design/design.pen`: editable design artifact. Current approved screens are the v2 set
  (`Search Screen / v2`, `Landing / v2`, `Admin / v2`, `Login / v2`, `User Profile / v2`,
  `Channel Profile / v2`). Older variants in the same file are reference-only.

Do not commit screenshots or exported images unless explicitly requested.

## Routes

- `/search`: public primary application search screen. No login required.
- `/admin`: authenticated admin dashboard for backend operations.
- `/login`: login screen.
- `/`: public compact landing page with project positioning and live analytics.
- `/users`: public user search index. Search-first — empty prompt on load, results populate as
  the user types.
- `/users/[slug]`: public sender profile with identity, analytics, and latest messages.
- `/channels`: public channel search index. Search-first — empty prompt on load, results populate
  as the user types.
- `/channels/[slug]`: public channel profile with stored Kick metadata, analytics, and latest
  messages.
- `/prediction`: public channel search page for Kick predictions. Search-first — empty prompt on
  load. Submitting navigates to `/prediction/{slug}`; it does not render analytics itself.
- `/prediction/[slug]`: public latest-prediction analysis for the channel (summary, charts, outcome
  cards, top users), fetched live from Kick in the browser.

Landing must stay product-focused and must not turn into a marketing site.

### Index Page UX Rules (`/users`, `/channels`)

- Search-first: initial load shows an empty state with a centered icon and a prompt string.
- Explicit submit only: typing does NOT fire requests. User clicks the `Ara` button or presses
  Enter to send the query. Debounce-while-typing is intentionally avoided so high-traffic
  ingestion is not put under additional ClickHouse `LIKE` pressure on every keystroke.
- Submit button is disabled until the trimmed query is at least 2 characters long.
- Submit button text switches to `Aranıyor…` and is disabled while a request is in flight.
- Results are ordered by message count (backend default).
- Empty-results state quotes the last submitted query, not the current input value.
- Loading, empty-results, and error states are all handled.
- Each result row links to the corresponding profile page.

## Visual Direction

The UI follows a minimal, dense, Vercel-influenced dark operations-tool direction. No marketing
typography, no decorative cards, no blur/glow.

### Palette

Tokenized as Tailwind / CSS variables. Hex values:

| Token            | Hex         | Usage                                       |
| ---------------- | ----------- | ------------------------------------------- |
| `accent`         | `#00e701`   | Kick green. Primary buttons, key accents.   |
| `accent-hover`   | `#00c701`   | Hover state for primary buttons.            |
| `accent-muted`   | `#00e70133` | Translucent green for soft accent fills.    |
| `bg-page`        | `#0b0e0f`   | Page background.                            |
| `bg-panel`       | `#191b1f`   | Cards, panels, banners.                     |
| `bg-elevated`    | `#24272c`   | Inputs, table cells, badges, emote chips.   |
| `bg-deep`        | `#000000`   | Reserved for deepest surfaces if needed.    |
| `border-subtle`  | `#24272c`   | Default hairline borders.                   |
| `border-strong`  | `#474f54`   | Active/focused inputs, secondary buttons.   |
| `text-primary`   | `#ffffff`   | Primary text.                               |
| `text-secondary` | `#9ca3af`   | Secondary text, descriptions.               |
| `text-muted`     | `#474f54`   | Labels, timestamps, low-emphasis text.      |
| `text-on-accent` | `#0b0e0f`   | Text/icons on green buttons.                |
| `danger`         | `#ff4d4f`   | Error states, "İnceleme gerekli", failures. |
| `warning`        | `#facc15`   | Reserved for soft warnings (not buttons).   |

The legacy magenta palette (`#26001B`, `#810034`, `#FF005C`, `#FFF600`) is fully replaced. No
backward compatibility, no theme switcher. Dark-only.

### Typography

- `font-sans`: **Geist** (variable). All UI body, headings, buttons.
- `font-mono`: **Geist Mono**. Timestamps, IDs, metric values where alignment matters, breadcrumbs,
  status strips, badge text, table column labels.

Type scale (Tailwind):

| Use                       | Size  | Weight                          |
| ------------------------- | ----- | ------------------------------- |
| Page title                | 22-24 | 600                             |
| Section title             | 13-14 | 600                             |
| Body                      | 13    | 400                             |
| Body emphasis             | 13    | 500                             |
| Metric value (big)        | 24-28 | 600                             |
| Stat label / column label | 10-11 | 500 (mono, letter-spacing ~0.8) |
| Caption / sub             | 11-12 | 400                             |

### Radii & Spacing

- Controls (buttons, inputs, badges): 6px.
- Panels/cards: 8px.
- Pill (badge with dot): 999px.
- Spacing scale: 4, 8, 12, 16, 20, 24, 32, 40, 48.

### Style Rules

- Dark background only. No theme switcher.
- Use only tokens above. New colors require a design decision.
- Primary buttons fill `accent` with `text-on-accent` text.
- Secondary buttons: transparent fill + `border-strong` outline + `text-primary`.
- Outline-only style for accent (e.g. `Sadece emote` toggle when on): `border-subtle` → `accent`
  border + small accent checkbox; no green flood.
- Icons via `lucide-react`. Note: lucide renamed `more-horizontal` to `ellipsis` and `alert-triangle`
  to `triangle-alert`; use the new names.
- Tailwind for layout. shadcn/ui primitives where they fit (Button, Dialog already in tree).
- Forms compact and work-focused. No marketing hero in app shell.
- Hairline 1px borders (`border-subtle`) on most surfaces. No shadows.

## Global Header

- 56px high, `bg-page` with bottom `border-subtle`.
- Desktop: left side has logo square + `kick logs` wordmark + nav links; right side has GitHub icon + `Admin` outline button.
- Mobile: left side has logo square + `kick logs` wordmark; right side has GitHub icon + hamburger (Menu/X toggle). Nav links hidden.
- Hamburger opens a fixed dropdown panel below the header (z-40) with a dark backdrop. Panel lists all nav links (Search, Channels, Users, Prediction) plus Admin. Clicking any link closes the menu.
- `Channels`, `Users`, and `Prediction` nav links are active and point to `/channels`, `/users`,
  and `/prediction`.
- `ActiveRoute` type supports `"search" | "channels" | "users" | "prediction"` to highlight the
  current section. Profile pages (`/channels/[slug]`, `/users/[slug]`) use the parent route
  (`"channels"`, `"users"`) so the nav item stays highlighted when browsing within a section.
- Clicking the brand goes to `/`.
- Admin page uses a different chrome: brand + `/ admin` breadcrumb on the left, user email +
  `SUPER ADMIN` badge + `Çıkış` outline button on the right.

## Landing Page (`/`)

- Compact hero: pill badge (`● Self-hosted · Açık kaynak`), large title `Kick chat için kalıcı log.`
  (48/600, letterSpacing -1), one-line description, primary CTA `Arama başlat`, secondary `GitHub`.
- Stats bar: 4 cells in one row (`TOPLAM MESAJ`, `KANAL`, `KULLANICI`, `EMOTE`). Cells share a single
  rounded border, separated by 1px hairlines.
- Two analytics rows of 2 columns each: `Mesaj hacmi` bar chart (14 days, accent green bars), `Top
kanallar`, `Top kullanıcılar`, `Top emoteler`. Each as a panel with title + mono sub.
- Data sources: `/analytics/overview`, `/analytics/message-volume?bucket=day`,
  `/analytics/top-channels`, `/analytics/top-emotes`, `/analytics/top-senders`.

## Search Screen (`/search`)

Public, no auth.

### Layout

1. Global header.
2. Page area: `Search` title + mono status strip (`Tüm kanallar · Yeni → Eski`). The previous
   `AND filtreleme` suffix is removed.
3. Form panel (single `bg-panel` card):
   - Row 1: `KULLANICI ADI`, `KANAL ADI`, `İÇERİK` inputs with lucide icons.
   - Row 2: `BAŞLANGIÇ`, `BİTİŞ`, `HIZLI ARALIK` select.
   - Row 3: `Sadece yanıtlar` and `Sadece emote` toggle pills on the left; on the right an action
     group: export icon button (square), `Sıfırla` secondary, `Ara` primary (accent green).
4. Results header: `Sonuçlar` + mono `1,284 mesaj` count; right side `son eşleşme` mono caption.
5. Results list (single `bg-panel` card, hairline-separated rows).
6. Inline load-more loader at the bottom: small accent dot + mono `daha eski mesajlar yükleniyor…`,
   centered, no border, sits inside the page area below the results card.

### Form Field Behavior

- All fields optional. Empty fields are not included in the API query.
- Date inputs are local `datetime-local`; converted to UTC ISO before calling the API.
- `Bitiş` includes the full selected minute (`02:43` → `02:43:59.999`).
- On first load and on reset, dates default to the last 7 days.
- `Hızlı aralık` updates only the date range.
- `Sadece yanıtlar` → `reply_only=true`; `Sadece emote` → `emote_only=true`.
- Opening `/search` with no query params must not auto-call the backend.
- Before submit, results area shows an icon + `Arama yapmak için yukarıdaki formu kullanın.`
- Export icon opens a small `JSON indir` / `CSV indir` menu; closes on choice or outside click.
  CSV/JSON use the last submitted filters, not unsent edits.

### Backend Mapping

```text
Kullanıcı Adı       -> sender
Kanal Adı           -> channel
Aramak istediğiniz Kelime -> q
Başlangıç           -> start
Bitiş               -> end
Sadece yanıtlar     -> reply_only
Sadece emote        -> emote_only
```

## Message Result Rows

Shared outer list container. Each row a flex row inside the same panel; rows separated by hairline,
no card-per-row treatment.

### Columns

- `meta` column (~140px): sender username on top, channel slug `#name` below. **No avatar.**
- `message` column (fill): reply preview chip (if reply) above, message content below.
- `timestamp` column (mono, muted, right-aligned).

### Sender Username Color

- Render with the sender's chosen Kick color (`sender_color_snapshot`). Fall back to
  `text-primary` when unset.
- Channel label `#channelSlug` always renders in `accent` green and links to `/channels/[slug]`.
- Sender username links to `/users/[slug]` (lowercase slug, `_` → `-`).

### Emote Rendering

- Messages may contain tokens like `[emote:37226:KEKW]`.
- Render emote image inline at the token's position using
  `https://files.kick.com/emotes/{id}/fullsize`. Size to match line height (~20px).
- Fall back to a small `bg-elevated` chip with the emote name in mono accent text when the image
  fails to load.
- Never relegate emotes to a footer chip.

### Reply Context

- Reply rows show the replied-to sender + replied-to message content above the current message in
  a muted `bg-elevated` mini-chip.
- Reply preview sender name links to `/users/[slug]` when available; otherwise derive a slug from
  the username.
- Long reply text exposes the full original content via a `title` attribute.

### Links and Highlighting

- URLs in message content render as anchors with `rel="noopener noreferrer"` and `target="_blank"`.
- Matched `q` text renders as restrained inline highlight without breaking emote placement.

## User Profile (`/users/[slug]`)

- Breadcrumb (mono): `users / yavuz`.
- Identity panel (`bg-panel`, horizontal): circular avatar (real image when available), username
  (22/600), `@slug` (mono muted), mono meta row (`ilk mesaj`, `son aktivite`), right-aligned primary
  CTA `Mesajlarda ara` linking to `/search?sender={slug}`.
- 4-cell stats bar: `MESAJ`, `KANAL`, `EMOTE`, `İLK MESAJ`.
- 3-column analytics grid with **equal panel heights**: `Mesaj hacmi` (chart height tuned to match
  list-panel height), `Top kanallar`, `Top emoteler`.
- `Son mesajlar` panel: list of channel-label (green) + message + mono timestamp. Same emote and
  user-color rules as search rows.
- Unknown sender: calm not-found state with link back to `/search`.

## Channel Profile (`/channels/[slug]`)

- Breadcrumb (mono): `channels / exampleChannel`.
- Identity panel: rounded-square channel image, display name (22/600), `LOGGING` accent pill, mono
  meta row (`ilk log`, `son aktivite`), CTA `Kanalda ara` linking to `/search?channel={slug}`.
- 4-cell stats bar: `MESAJ`, `KULLANICI`, `EMOTE`, `İLK LOG`.
- 3-column analytics grid with **equal panel heights**: `Mesaj hacmi`, `Top kullanıcılar`,
  `Top emoteler`.
- `Son mesajlar` panel: username (rendered in sender color) + message + mono timestamp. Same emote
  and user-color rules as search rows.
- Unknown channel: calm not-found state.

## Prediction (`/prediction`, `/prediction/[slug]`)

Public, no auth. Visualizes a channel's latest Kick prediction game. Data is fetched live from
Kick's public web endpoints in the browser; nothing is persisted and the Go API is not in this
request path.

### Search page (`/prediction`)

- Search-first, submit-only (same contract as `/users` and `/channels`): typing fires no request.
- One channel-name input + a `Göster` primary button. Disabled until the trimmed query is ≥ 2
  characters.
- Submit trims and lowercases the input, then navigates to `/prediction/{slug}`. No analytics render
  on this page.
- Idle prompt: centered icon + `Tahmin verisi için kanal seçin`.

### Analysis page (`/prediction/[slug]`)

- Header strip: back link to `/prediction`, a `kick.com/{slug}` label (`kick.com/` in accent, slug in
  `text-primary`, mono), and a refresh icon button that re-runs the fetch.
- States: loading (mono accent-dot line), not-found (calm `warning` panel + `Başka kanal dene`),
  error (`danger` panel + `Tekrar dene`).
- Once a prediction is loaded, the analysis page refreshes every 5 seconds while the page stays
  open, including after `LOCKED`, `CANCELED`, `CANCELLED`, or `RESOLVED`. Refreshes happen in the
  background without returning the page to the loading state, so charts and outcome cards update in
  place without a visible flash. The client validates the channel once per slug/browser fetch
  context, then poll refreshes call only the latest-prediction endpoint.
- Summary card (`bg-panel`): prediction title + state pill; 4-cell metric row (`TOPLAM PUAN`,
  `TOPLAM OY`, `SÜRE`, `OLUŞTURULMA`). A muted lock timestamp line shows when `lockedAt` is set.
- Outcome detail cards appear directly below the summary card, before charts: colored dot + title,
  `KAZANAN` accent badge + accent border for the winner, three stats (`PUAN` + share %, `OY`,
  `GETİRİ` as `x` multiplier), an outcome-colored progress bar filled by that outcome's
  point-share, and a ranked top-users list. On tablet/desktop, outcome cards split the full row into
  two equal columns; do not switch them to a three-column grid for two-outcome predictions. When a
  winner exists, losing outcome cards render slightly lower opacity. Top-user rows get a subtle
  translucent white hover background.
- Chart row (two panels, stack on narrow): `Puan dağılımı` donut (point share per outcome) and
  `Oy ve getiri` grouped bar (vote count + return rate per outcome).
- `Top kullanıcılar` panel: horizontal bar chart of the highest bettors across outcomes, bar color
  by owning outcome, with an outcome legend.

### State Pills

The base palette has no blue/purple/cyan, so prediction states map onto existing tokens (never a new
hue):

- `RESOLVED` → accent green, `Sonuçlandı`.
- `LOCKED` → `warning` yellow, `Kilitli`.
- `CANCELED` / `CANCELLED` → `warning` yellow, `İptal`.
- `ACTIVE` → neutral (`bg-elevated` + `text-secondary`), `Aktif`.
- unknown → neutral, raw state text.

### Chart Palette

Multi-series prediction charts use `recharts` (the only charting dependency in the repo; landing bars
remain hand-rolled). Categorical colors come from a fixed sequence rooted in the app palette, cycled
by outcome index. The first two colors are intentionally high-contrast for common two-outcome
predictions:

```text
#22C55E · #C084FC · #FFFFFF · #FF005C · #474f54 · #26001B
```

In `Oy ve getiri`, vote count uses `#22C55E` and return rate uses `#C084FC`, with separate left/right
Y axes so large vote counts do not hide the return-rate bars. Axes, grid, and tooltip chrome use
`text-secondary` / `border-subtle` / `bg-elevated`. Charts must stay readable without color alone:
every chart ships labels or a legend, and legends must sit outside fixed-height chart containers so
they do not overflow panel borders.

## Admin (`/admin`)

Login required.

### Layout

- Top chrome: brand + `/ admin` mono breadcrumb (left), `email`, `SUPER ADMIN` badge, `Çıkış` button
  (right).
- Two-column body: left sidebar (220px) with vertical nav, right main column.

### Sidebar Nav

- `Operations`, `Channels`, `Users`, `Data`, `Settings`. Active item: `bg-panel` background with
  hairline, accent icon, `text-primary` label.
- Regular `admin` (non-super) users do not see `Users`.

### Operations Section

- Header: `Operations` (22/600) + sub `Listener sağlığı, ingestion durumu, depolama özeti`. Right:
  `Yenile` outline button.
- Status banner panel: accent dot, two-line status (line 1 listener freshness, line 2 failed-events
  count in `danger` when > 0). Right: UTC timestamp mono.
- 4-cell metric row: `Mesaj`, `Raw event`, `Başarısız raw`, `DB boyutu`. Each cell has a top mono
  label + lucide icon (icon color follows severity), big metric value, sub-caption (caption color
  follows severity).
- `Başarısız raw` card's sub-caption `İnceleme gerekli` is clickable and opens the existing failed
  events modal. The retry/clear modal stays as-is.
- Ingestion panel: header `Ingestion` + mono sub `queue, breaker, flush`; right pill
  `BREAKER CLOSED` / `OPEN` with accent or danger dot. Below: 6-cell mono metric strip
  (`Queue depth`, `Write queue`, `Drop count`, `Flush count`, `Son flush`, `CH failures`).

### Channels Section

- Panel header `Takip edilen kanallar` + mono `N aktif`. Right: inline add form (input with `+`
  icon + accent `Ekle` button).
- Table: `KANAL`, `DURUM`, `MESAJ`, `SON AKTİVİTE`, action ellipsis. Channel row image is a small
  rounded square placeholder until real images land.

### Data Management

- Lives as its own section (route slot under `Data`). Shows database/table sizes, retention
  settings, dry-run cleanup preview, explicit confirmation input, success/error states.
- Destructive cleanup must always show a dry-run preview first and require typing the exact
  confirmation text returned by the backend.

### Users Section (super admin only)

- Visually separate from channel management. No hero/landing treatment.

### Default Super Admin Credentials (local MVP)

```text
email: admin@kicklogs.local
password: admin123
```

## Login (`/login`)

- Centered card on `bg-page`. Card: `bg-panel`, 8-10px radius, hairline border, 32px padding.
- Header: brand square + `kick logs` (18/600), centered sub `Yönetim panelinize giriş yapın`.
- Fields: `E-POSTA`, `ŞİFRE` with lucide icons. Filled state uses `border-strong` instead of
  `border-subtle`.
- Submit: full-width `Giriş yap` primary accent button.
- Footer: muted `← Public arama sayfasına dön` link.

## Component Rules

- `components/ui` contains shadcn/ui primitives only.
- Feature-specific components live under their feature folders.
- Search components under the search feature. Admin under admin feature.
- API calls use the shared API client layer.
- Centralize API response types and reuse them.

## Responsive Behavior

- Search form remains usable on mobile and desktop.
- Inputs do not overflow.
- Buttons keep stable height regardless of icon/text changes.
- Result rows preserve readable message text alignment without avatars.

## Implementation Order

The frontend is being re-styled, not rebuilt. The backend API is already in place, so the v2
work is purely a re-skin against existing endpoints. Order:

1. Apply tokens (Tailwind config + globals.css).
2. Update header/chrome.
3. Re-style `/search` form + result rows (no avatar, sender color, inline emotes, loader).
4. Re-style `/`, `/users/[slug]`, `/channels/[slug]`.
5. Re-style `/admin` (sidebar layout + sections).
6. Re-style `/login`.

## Update Policy

When the user makes a UI decision in chat:

1. Update this file in the same unit of work.
2. Update `docs/context/recent_changes.md` with a short handoff summary.
3. Update `docs/context/decisions.md` if the decision is durable.
4. Update `docs/context/change_log.md` for chronological history.
