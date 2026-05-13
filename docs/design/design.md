# Kick Logs UI Design Guide

This document is the source of truth for UI and UX decisions. Update it whenever a later conversation changes search UI, admin UI, visual style, routing, or frontend behavior.

## Current Rule: Backend First

Do not build frontend UI until the backend API is working end-to-end.

Implementation order:

1. Finish backend API and database.
2. Finish listener ingestion.
3. Verify API behavior through backend tests and manual API calls.
4. Build frontend screens against the working API.

Frontend planning can continue, but no UI code should be scaffolded before the backend is functional.

## Design Sources

- `docs/design/design.md`: human-readable UI rules and decisions.
- `docs/design/design.pen`: editable design artifact. It currently contains the first `/search` screen draft only.

The search reference image from planning is structural guidance only. Keep the compact form order and dense search workflow, but do not copy the exact look, green palette, or spacing one-to-one.

Use the application logo asset provided by the user where the UI needs the product mark. In early design drafts, a simplified message mark can be used until the frontend asset path is finalized.

Do not commit screenshots or exported images unless explicitly requested.

## Routes

- `/search`: public primary application search screen. No login is required.
- `/admin`: authenticated admin dashboard for backend operations.
- `/login`: login screen.
- `/`: redirects to `/search` until a future landing page is intentionally designed.

Do not build a landing page before the application screens exist.

## Visual Direction

The UI should follow a dark, serious, professional operations-tool direction.

Palette:

- `#26001B`: primary dark background
- `#810034`: borders, secondary surfaces, low-emphasis accents
- `#FF005C`: accent icons and alert/highlight states
- `#FFF600`: primary action color, especially buttons
- `#000000`: deep surfaces and panels
- `#FFFFFF`: primary text

Style rules:

- Use a dark background.
- Use only the palette above unless a future decision explicitly extends it.
- Do not provide a theme switcher; the app is dark-only.
- Prioritize `#FFF600` for primary buttons.
- Use `#FF005C` for selected accent icons or highlights, not as the default button color.
- Keep forms compact and work-focused.
- Avoid marketing-style hero sections in the application shell.
- Avoid decorative card-heavy layouts.
- Avoid large display typography because the app screens are not landing pages.
- Avoid blur, glow, colored lighting, or atmospheric background effects.
- Keep control and button corner radii modest. Prefer 6px for controls and 8px maximum for panels/cards unless a later decision overrides this.
- Use clear spacing, predictable alignment, and dense but readable information.
- Use shadcn/ui primitives for reusable controls.
- Use lucide-react icons for field labels, buttons, and actions.
- Use Tailwind for layout and visual styling.

Current design scope:

- Design `/search` first.
- Do not design `/admin` screens until the `/search` screen is approved.
- Do not design landing-page-like screens yet.

## Search Screen

The current `/search` draft in `design.pen` should feel like a serious data/search tool, not a marketing page.

Access:

- `/search` is public and should be usable by anyone without authentication.
- Do not show admin-only controls on the public search screen.

Header/navigation:

- Keep the top region compact and functional.
- Show the Kick Logs brand lockup and active `/search` route clearly.
- Use restrained navigation with the search route as the active state.
- A small query-scope/status strip can summarize search behavior such as all-channel scope, newest-first ordering, and `AND` filter logic.
- Admin-account controls should not appear on `/search`; backend management belongs in `/admin`.
- Header controls should keep the dark palette, yellow active/action states, pink accent icons, and modest corner radii.

Search form fields:

- `Kullanıcı Adı`
- `Kanal Adı`
- `Aramak istediğiniz Kelime`
- `Başlangıç`
- `Bitiş`
- `Sadece yanıtlar` toggle for reply-only results
- `Sadece emote` toggle for emote-only results
- Compact `Hızlı aralık` select for `1 saat`, `24 saat`, `7 gün`, `30 gün`

Search button:

- Yellow primary button using `#FFF600`.
- Includes lucide search icon.
- Label: `Ara`

Field behavior:

- All fields are optional.
- Empty fields are not included in the API query.
- Search semantics must match backend `/messages` behavior.
- Date fields map to `start` and `end`.
- Date inputs are local `datetime-local` values in the UI and URL.
- Before calling the API, the frontend converts date filters to UTC ISO strings.
- The `Bitiş` value includes the full selected minute, so `02:43` includes messages through `02:43:59.999`.
- On first `/search` load and on reset, date fields default to the last 7 days:
  - `Başlangıç`: current local date/time minus 7 days.
  - `Bitiş`: current local date/time.
- The `Hızlı aralık` select updates only the date range and keeps the other filters intact.
- Date inputs and the quick range select sit on their own row so time filters do not compete
  with secondary result filters.
- Users can still clear or change the date fields; cleared date fields are omitted from the API query.
- `Sadece yanıtlar` maps to `reply_only=true`.
- `Sadece emote` maps to `emote_only=true`.
- `Sadece yanıtlar` and `Sadece emote` sit in the row below the date controls, directly to
  the left of the `İşlem` action group.
- Opening `/search` without URL query parameters must not call the backend automatically.
- Before the user submits a search, the results area shows an icon with `Arama yapmak için yukarıdaki formu kullanın.`
- An explicit search submit can still fetch latest messages when all filters are empty.
- Export is a single square download icon button. Clicking it opens compact `JSON indir` and
  `CSV indir` actions.
- The export menu closes after choosing an export format or clicking outside the menu.
- CSV and JSON export actions use the last submitted search filters, not unsent edits.

Backend query mapping:

```text
Kullanıcı Adı -> sender
Kanal Adı -> channel
Aramak istediğiniz Kelime -> q
Başlangıç -> start
Bitiş -> end
Sadece yanıtlar -> reply_only
Sadece emote -> emote_only
```

Examples:

- Only `Kullanıcı Adı=yavuz`: search all channels and all content for sender username/slug exactly matching `yavuz`.
- `Kullanıcı Adı=yavuz` and content `selam`: search all channels for sender username/slug exactly matching `yavuz` and message content containing `selam`.
- `Kanal Adı=exampleChannel` and content `hello`: search that channel for messages containing `hello`.
- Only content `hello`: search all channels and all users for messages containing `hello`.
- `Sadece yanıtlar` enabled: only reply messages.
- `Sadece emote` enabled: only messages with parsed emotes.
- Empty all filters: show latest messages across all channels.

Results:

- Infinite scroll.
- Newest-first ordering.
- More results load as the user scrolls down.
- Loading and empty states should be visually calm and compact.
- The load-more/loading hint should be compact and inline with the results list, not a full-width bordered strip.
- The user-friendly search retouch should keep the results list dominant while adding a compact summary panel for scope, status, last match, and active filters.
- The current user-friendly desktop retouch is represented in `design.pen` as `Search Screen / Desktop (User Friendly ReTouch Current)`.

## Message Result Rows

Search results should use one shared outer list container. Do not render every message as its own modal-like card because the screen can contain many messages and should support efficient infinite scrolling.

Each message row should show:

- sender avatar
- sender nickname
- channel nickname/slug
- timestamp
- message content
- emote rendering with fallback
- clickable links when message content contains URLs
- highlighted matched `q` text
- reply context above the current message when the Kick payload is a reply

Sender avatar:

- Use enriched sender profile image when available.
- Render sender avatars as fully circular images.
- Use a stable circular fallback avatar when no profile image exists.

Emotes:

- Messages may contain tokens like `[emote:37226:KEKW]`.
- Render emote image with `https://files.kick.com/emotes/{id}/fullsize`.
- If image loading fails, fall back to the emote name or original token.
- Render emotes inline at the position where they appear in the message content, not as a separate footer chip.

Result row layout:

- Rows should be stacked vertically inside the shared results container.
- Fixed metadata columns can hold avatar, sender, channel, and timestamp.
- The message content column should take the remaining horizontal space.
- If the content is long, the message column should absorb the extra width/wrapping behavior instead of turning the row into a separate card.
- Reply rows render the replied-to sender and replied-to message content above the current message in muted gray text.
- Reply preview data comes from `reply_metadata.original_sender.username` and `reply_metadata.original_message.content` when `message_type` is `reply`.
- Long reply preview text should expose the full replied-to content through a `title` attribute.
- URLs inside message content should render as clickable anchors without breaking inline emote
  placement or matched-text highlighting.
- Message links should open in a new tab with `rel="noopener noreferrer"` and use restrained
  styling that fits the dark results row.
- Matched `q` text should render as a restrained inline highlight in both plain text and link
  text, without replacing or moving emote images.

## Admin UI

Admin route:

```text
/admin
```

Admin purpose:

- `/admin` manages backend operational state.
- Primary admin workflow is managing followed Kick channels, such as adding/removing channels to ingest.
- Super admin can also manage admin users.

Admin requirements:

- Login required.
- Super admin and admin can manage followed channels.
- Super admin can create new admin users.
- Channel add form accepts Kick channel slug/nickname.
- System resolves channel metadata through backend API.
- Channel list shows enabled followed channels.
- Remove action disables or removes a followed channel according to backend behavior.

Implemented admin layout:

- `/login` uses a compact dark email/password form with the app logo and a restrained link back to public search.
- `/admin` uses a guarded operations layout with:
  - operations dashboard at the top of the main work column.
  - channel management in the main work column.
  - super-admin-only user management below channel management.
  - current session summary in a right-side panel.
- Regular `admin` users do not see the user management panel.
- Channel and user management are visually separate sections; neither uses hero/landing-page treatment.
- Operations dashboard shows compact cards for listener freshness, database size, message count,
  raw event count, failed raw events, pending raw events, and last ingest time.
- Operations dashboard includes a manual refresh action and calm warning/error states for stale
  listener heartbeat, failed raw events, and API failures.

Default super admin credentials for local MVP:

```text
email: admin@kicklogs.local
password: admin123
```

## Component Rules

- `components/ui` contains shadcn/ui primitives only.
- Feature-specific components live under their feature folders.
- Search components should be grouped under the search feature.
- Admin components should be grouped under admin-related features.
- API calls should use the shared API client layer.
- Keep API response types centralized and reused.

## Responsive Behavior

- Search form should remain usable on mobile and desktop.
- Inputs should not overflow.
- Buttons should keep stable height and not resize due to icon/text changes.
- Result rows should preserve readable message text and avatar alignment.

## Update Policy

When the user makes a UI decision in chat:

1. Update this file in the same unit of work.
2. Update `docs/context/recent_changes.md` with a short handoff summary.
3. Update `docs/context/decisions.md` if the decision is durable.
4. Update `docs/context/change_log.md` for chronological history.
