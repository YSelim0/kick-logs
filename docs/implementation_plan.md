# Active Channel Subscribers Implementation Plan

## Summary

This plan defines the next product feature: visitors should be able to open a channel profile page
at `/channels/{slug}` and view the active subscriber list for that channel.

The list is based only on subscription periods already captured by Kick Logs. It must not imply that
the app knows subscribers from before webhook tracking was enabled.

The feature builds on the existing subscription summary flow:

- `GET /channels/{slug}/subscription-summary`
- `SubscriptionPeriodRepository.ActiveSummary`
- ClickHouse table: `channel_subscription_periods`

The existing summary count stays in place. The new work adds a detailed subscriber list modal and a
user-friendly full-list download.

## Current Status

- Active subscriber counts already exist on channel profile pages.
- Backend active count logic counts distinct subscribers where `expires_at > now()`.
- Gifted active count counts distinct subscribers where `expires_at > now()` and `is_gift = 1`.
- Backend public subscriber list and export endpoints have been implemented.
- Channel profile frontend exposes the active and gifted subscriber counts as modal triggers.
- The modal supports paginated viewing plus JSON, CSV, and TXT download options.
- Streak/month count remains intentionally omitted because the stored webhook data does not contain a
  reliable source field.

## Product Goals

- Let visitors inspect which captured subscribers are currently active for a channel.
- Keep channel profile pages compact by opening the subscriber list in a modal.
- Make the active subscriber count itself discoverable and actionable.
- Let visitors download the full active subscriber list as JSON, CSV, or a readable TXT report.
- Keep UI copy understandable for normal visitors; do not mention webhook internals in public empty
  states.
- Preserve the existing dark, dense, professional visual system.

## Non-Goals

- No attempt to reconstruct subscribers from before Kick Logs started capturing subscription events.
- No admin-only subscriber management UI.
- No public account system.
- No XLSX/PDF export for this feature.
- No display of subscription streak/month count unless Kick payload data is explicitly available and
  persisted later.
- No database migration just to infer streak/month count.
- No new landing page or large marketing-style section.

## Data Source

Use ClickHouse table `channel_subscription_periods`.

Relevant stored fields already available:

- `followed_channel_id`
- `channel_slug`
- `channel_display_name`
- `subscriber_kick_user_id`
- `subscriber_username`
- `subscriber_slug`
- `subscriber_profile_image_url`
- `gifter_kick_user_id`
- `gifter_username`
- `gifter_slug`
- `gifter_profile_image_url`
- `is_gift`
- `started_at`
- `expires_at`
- `event_type`

Active subscriber definition:

- `expires_at > now()`

The existing summary query does not check `started_at <= now()`. Keep list behavior aligned with the
existing count unless a later bug fix intentionally changes both summary and list together.

## Streak And Month Count Decision

Do not show month/streak data in the UI or export for now.

Reasoning:

- Current Kick webhook payload handling does not persist an explicit streak/month field.
- Deriving a "registered month count" from captured periods can be misleading because the app may
  not have historical periods from before tracking began.
- Public UI and downloaded files should avoid presenting inferred values as subscription truth.

If Kick payloads later expose a reliable streak/month field, add it in a separate feature with a
clear migration/storage plan.

## Backend API

### Paginated Subscriber List

Endpoint:

```text
GET /channels/{slug}/subscribers?limit=50&offset=0&gift_only=false
```

Access:

- Public.

Purpose:

- Used by the channel profile modal.
- Returns only the requested page of active subscribers.

Query params:

- `limit`: default `50`, max `100`.
- `offset`: default `0`.
- `gift_only`: optional boolean. When true, return only active gifted subscribers.

Suggested response:

```json
{
  "channel_slug": "nuriben",
  "items": [
    {
      "subscriber_kick_user_id": 123456,
      "username": "example_user",
      "slug": "example-user",
      "profile_image_url": "https://...",
      "is_gift": true,
      "gifter_kick_user_id": 987654,
      "gifter_username": "gift_sender",
      "gifter_slug": "gift-sender",
      "gifter_profile_image_url": "https://...",
      "started_at": "2026-05-17T06:05:00Z",
      "expires_at": "2026-06-16T06:05:00Z"
    }
  ],
  "count": 248,
  "limit": 50,
  "offset": 0
}
```

### Full Export

Endpoint:

```text
GET /channels/{slug}/subscribers/export?gift_only=false&format=txt
```

Access:

- Public.

Purpose:

- Downloads the full active subscriber list.
- Supported formats:
  - `json`
  - `csv`
  - `txt`

Response headers:

- TXT:
  - `Content-Type: text/plain; charset=utf-8`
  - `Content-Disposition: attachment; filename="{channel_slug}-active-subscribers.txt"`
- CSV:
  - `Content-Type: text/csv; charset=utf-8`
  - `Content-Disposition: attachment; filename="{channel_slug}-active-subscribers.csv"`
- JSON:
  - `Content-Type: application/json; charset=utf-8`
  - `Content-Disposition: attachment; filename="{channel_slug}-active-subscribers.json"`

Suggested text output:

```text
Kick Logs Aktif Abone Listesi

Kanal: nuriben
Liste: Tüm aktif aboneler
Oluşturulma: 14.06.2026 01:15
Toplam: 248

1. ID: 123456
   Kullanıcı: example_user
   Başlangıç: 17.05.2026 06:05
   Bitiş: 16.06.2026 06:05
   Tür: Hediye
   Hediye eden: gift_sender
```

Gift-only TXT export should change `Liste` to `Hediye aktif aboneler`.

Empty export should still download a useful file/payload with total `0` and no item rows.

## Backend Query Design

The list must avoid duplicate subscribers.

Expected behavior:

- A subscriber appears once in the current active list.
- If multiple active periods exist for the same subscriber, choose the latest period by
  `started_at DESC`, then `expires_at DESC`.
- `gift_only=true` filters rows where the selected active period is gifted.
- Sort public list by `started_at DESC`, then `subscriber_username ASC`.
- Return a total count for the current filter.

Implementation approach:

- Add a domain model for active channel subscribers.
- Extend `SubscriptionPeriodRepository` port with:
  - paginated list method,
  - full export/list method or an unbounded method with a safe export cap if needed.
- Add ClickHouse repository methods using query-time aggregation/windowing.
- Keep the query readable first; optimize later only if production data requires it.

## Rate Limiting

Add a public rate-limit policy for subscriber export.

Recommended:

- `GET /channels/{slug}/subscribers/export`
- IP key
- similar to message export, e.g. 3 requests per 60 seconds with burst 2

Paginated list calls can reuse the existing profile/analytics public limits only if matching remains
clear. If not, add a dedicated `channel-subscribers` policy with a moderate limit.

## Frontend UX

Location:

- Channel profile page: `/channels/{slug}`.

Modal triggers:

- `AKTİF ABONE` stat cell opens the modal for all active subscribers.
- `HEDİYE ABONE` stat cell opens the same modal with `gift_only=true`.

Modal behavior:

- Title reflects the selected mode:
  - `Aktif aboneler`
  - `Hediye aktif aboneler`
- First open fetches `limit=50&offset=0`.
- `Daha fazla yükle` fetches the next page.
- The download button opens a compact export menu, matching the search page export pattern.
- Export menu options:
  - JSON
  - CSV
  - TXT
- Selecting an option downloads the full list for the same modal mode.
- Modal can be closed by close button, Escape, or backdrop click using the existing dialog pattern.

List row content:

- Subscriber avatar when available; otherwise a compact fallback initial.
- Subscriber username.
- Subscriber username links to the app user profile when slug is available.
- Gift badge when `is_gift=true`.
- Gifter username when available.
- Start date.
- Expiry date.

Do not show:

- webhook terminology,
- inferred month count,
- inferred streak count,
- internal event IDs.

Empty state:

```text
Bu kanal için henüz aktif abonelik kaydı yok.
```

## Download UX

The modal download action should follow the search page export pattern:

- A square/icon download button opens an export menu.
- The menu contains JSON, CSV, and TXT options.
- Clicking outside closes the menu.
- Selecting an option starts the download and closes the menu.

Behavior:

- Each option starts a browser download from the export endpoint with the selected `format`.
- The button should not require loading the full list into the modal first.
- Download uses the current modal mode:
  - all active subscribers,
  - gift-only active subscribers.

TXT should be the most human-readable report format. JSON and CSV are provided for users who want to
reuse the list elsewhere.

## Implementation Phases

### Phase 1 - Plan And Context

- Replace active implementation plan with this active subscriber list plan.
- Update context docs with the new feature decision.
- Do not implement runtime code in this phase unless explicitly requested.

Exit criteria:

- `docs/implementation_plan.md` contains only this active feature plan.
- Previous request-form implementation plan content is removed from the active plan.
- Context docs mention the no-streak/no-month-count decision.

### Phase 2 - Backend List And Export API

- Add domain response models for active channel subscribers.
- Extend `SubscriptionPeriodRepository` port.
- Implement ClickHouse paginated list query.
- Implement ClickHouse full export query or safe full-list method.
- Add public routes:
  - `GET /channels/{slug}/subscribers`
  - `GET /channels/{slug}/subscribers/export`
- Add schemas/mappers for JSON response.
- Add JSON, CSV, and TXT export formatters.
- Add focused backend tests for:
  - active subscribers only,
  - gift-only filtering,
  - duplicate subscriber collapse,
  - pagination,
  - JSON, CSV, and TXT export formats,
  - missing channel behavior.

Exit criteria:

- Public list endpoint returns stable JSON.
- Public export endpoint downloads JSON, CSV, and readable TXT formats.
- Existing subscription summary behavior is not broken.

### Phase 3 - Frontend Modal

- Add channel subscriber API helpers.
- Add TypeScript response types.
- Make `AKTİF ABONE` and `HEDİYE ABONE` stat cells clickable when a channel profile is ready.
- Add subscriber list modal using the existing dialog pattern.
- Add first-page loading, load-more, empty, and error states.
- Add download menu with JSON, CSV, and TXT options.
- Add frontend tests for:
  - clicking active count opens all-active modal,
  - clicking gifted count opens gift-only modal,
  - empty state copy,
  - load-more behavior,
  - export link/button target.

Exit criteria:

- Visitors can inspect active subscribers without leaving the channel page.
- Modal is responsive and does not render huge lists at once.
- Visitors can download the full active subscriber list in JSON, CSV, or TXT.

### Phase 4 - Docs And Verification

- Update design docs for clickable stats and subscriber modal.
- Update context docs with completed implementation details.
- Run backend tests.
- Run frontend tests.
- Run typecheck, lint, build, and formatting checks.

Exit criteria:

- CI-relevant checks pass.
- Documentation reflects the new feature.
- No unrelated files are included.

## Manual Test Plan

1. Start Docker Compose.
2. Visit `/channels/{slug}` for a channel with captured subscription periods.
3. Click `AKTİF ABONE`.
4. Confirm the modal lists active subscribers.
5. Click `Daha fazla yükle` when more than one page exists.
6. Close and reopen from `HEDİYE ABONE`.
7. Confirm only gifted active subscribers are shown.
8. Click the download button.
9. Confirm the export menu shows JSON, CSV, and TXT options.
10. Download each format and confirm the files include channel name, generated time, total, and
    subscriber rows where that format supports those fields.
11. Visit a channel with no active subscriptions and confirm the public empty state copy.

## Open Decisions

No blocking decisions remain.

Non-blocking implementation details:

- exact modal row density,
- exact file name pattern,
- exact export date formatting,
- whether list export should have a hard maximum row cap later.
