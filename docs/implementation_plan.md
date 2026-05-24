# Channels & Users Index Pages — Search Screens

## Summary

Visiting `/channels` or `/users` currently shows nothing — no index `page.tsx` files exist.
This change adds a **search-first** index page to both routes.

On page load, an empty search screen is shown with a prompt like "Search to find a user" /
"Search to find a channel". As the user types in the search input, results populate below.
A new `q=` text search parameter is added to the backend.

## Current State

- `/users/[slug]` and `/channels/[slug]` profile pages already work.
- Backend has `GET /analytics/top-senders` and `GET /analytics/top-channels` endpoints with
  `limit` (1–100) support, but no text search.
- `SiteHeader` (`site-header.tsx`) intentionally omits `Channels` / `Users` nav links
  "until those index pages exist" — we are now adding them.
- Prettier config: `.prettierrc.json` (100 col, LF, no trailing comma).
- CI: `code-style.yml` → `pnpm format:check`, `go-tests.yml` → `go test ./...`.

## Design Decisions

- Both pages are search-first: empty screen with a search prompt on initial load, results
  populate as the user types.
- Results are ordered by message count (backend default).
- `Channels` and `Users` nav links are added to `SiteHeader`.
- A `q=` parameter is added to the backend for server-side contains search.

---

## Proposed Changes

### Backend — `q=` Search Parameter

#### [MODIFY] [models.go](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/apps/api-go/internal/domain/models.go)

- Add `Query string` field to `AnalyticsFilter` struct.

#### [MODIFY] [analytics.go](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/apps/api-go/internal/infra/clickhouse/analytics.go)

- `analyticsWhere` adds a WHERE clause when `filter.Query` is set:
  - For `TopSenders`: `sender_username_lower LIKE '%…%' OR sender_slug_lower LIKE '%…%'`.
  - For `TopChannels`: `channel_slug_lower LIKE '%…%' OR channel_display_name_lower LIKE '%…%'`.
- The text search applies in the WHERE clause before GROUP BY, filtering on the denormalized
  `chat_messages` snapshot columns. This is consistent with how existing sender/channel scope
  filters work.

#### [MODIFY] [analytics.go](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/apps/api-go/internal/http/routes/analytics.go)

- `parseAnalyticsFilter` reads `request.URL.Query().Get("q")` and assigns it to
  `filter.Query`.
- `q` is only effective for `top-senders` and `top-channels` endpoints. Other analytics
  endpoints (overview, message-volume, top-emotes) ignore it.

#### [NEW] Backend unit tests

- Route tests verifying that the `q=` parameter correctly filters TopSenders and TopChannels
  results.

---

### Frontend — Route Pages

#### [NEW] [page.tsx](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/apps/web/src/app/channels/page.tsx)

- Next.js route handler rendering the `ChannelsIndexPage` component.

#### [NEW] [page.tsx](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/apps/web/src/app/users/page.tsx)

- Next.js route handler rendering the `UsersIndexPage` component.

---

### Frontend — Feature Components

#### [NEW] [channels-index-page.tsx](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/apps/web/src/features/channel-profile/channels-index-page.tsx)

- `"use client"` component.
- On initial load the result list is empty. A centered icon + "Search to find a channel"
  message is shown.
- As the user types in the search input, `GET /analytics/top-channels?q=…&limit=20` is called
  (debounced, ~300ms).
- Results rendered as rows/cards: channel image (rounded square), display name, slug,
  message count, last activity.
- Each row links to the `/channels/[slug]` profile page.
- Loading, empty (no search results), and error states.
- v2 design tokens: `bg-page`, `bg-panel`, `bg-elevated`, `border-subtle`, accent green.

#### [NEW] [users-index-page.tsx](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/apps/web/src/features/user-profile/users-index-page.tsx)

- `"use client"` component.
- On initial load the result list is empty. A centered icon + "Search to find a user" message
  is shown.
- As the user types in the search input, `GET /analytics/top-senders?q=…&limit=20` is called
  (debounced, ~300ms).
- Results rendered as rows: profile image (circular), username, slug, message count.
- Each row links to the `/users/[slug]` profile page (`_` → `-` slug conversion).
- Loading, empty, and error states.
- v2 design tokens.

---

### Frontend — Feature Tests

#### [NEW] [channels-index-page.test.tsx](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/apps/web/src/features/channel-profile/channels-index-page.test.tsx)

- Empty initial state (prompt message).
- Channel list renders after search input.
- Empty search result state.
- Profile link correctness.

#### [NEW] [users-index-page.test.tsx](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/apps/web/src/features/user-profile/users-index-page.test.tsx)

- Empty initial state (prompt message).
- User list renders after search input.
- Empty search result state.
- Profile link correctness (`_` → `-` slug conversion).

---

### Frontend — Site Header Update

#### [MODIFY] [site-header.tsx](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/apps/web/src/components/site-header.tsx)

- `ActiveRoute` type supports `"channels" | "users"` options.
- `Channels` and `Users` nav links added to the header.

---

### Frontend — Analytics API Update

#### [MODIFY] [api.ts](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/apps/web/src/features/analytics/api.ts)

- `AnalyticsQueryParams` type gets a `q?: string` field.
- `buildAnalyticsQuery` passes the `q` parameter through.

---

### Docs — Design & Context Updates

#### [MODIFY] [design.md](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/docs/design/design.md)

- `/channels` and `/users` index page design rules added.
- Header nav updated (`Channels` / `Users` links now active).

#### [MODIFY] [living_brain.md](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/docs/context/living_brain.md)

- New pages and backend `q=` parameter documented.

#### [MODIFY] [recent_changes.md](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/docs/context/recent_changes.md)

- Handoff summary.

#### [MODIFY] [change_log.md](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/docs/context/change_log.md)

- Chronological entry.

#### [MODIFY] [decisions.md](file:///c:/Users/yavuz/Desktop/codes/web/Projects/kick-project/kick-logs/docs/context/decisions.md)

- Design decisions: search-first UX, backend `q=` parameter, header nav.

---

## Verification Plan

### Automated Tests

```powershell
# Backend checks
cd apps/api-go
go test ./...
go vet ./...

# Frontend checks
pnpm --filter @kick-logs/web typecheck
pnpm --filter @kick-logs/web lint
pnpm --filter @kick-logs/web test
pnpm --filter @kick-logs/web build
pnpm format:check
```

### Manual Verification

- `http://localhost:3000/channels` → empty search screen, results appear as user types.
- `http://localhost:3000/users` → empty search screen, results appear as user types.
- Clicking a channel/user row navigates to the profile page.
- `Channels` and `Users` nav links visible in the header.
- Responsive behavior (mobile/desktop).

## Completed Plans

- Frontend v2 re-skin (all six routes). Archived under `docs/archive/redesign/`.
- Issue #11: Worker hot-path optimizations (sender resolver removal + batch existence + batch
  raw-event fetch). PR #12 merged.
- Issue #9: Stabilize high-volume Kick chat ingestion with ClickHouse batching and backpressure.
  Archived under `docs/archive/issue_09/`.
- Go + ClickHouse rewrite (phases 1-9). Archived under `docs/archive/go_rewrite/`.
- Post-MVP features (features 1-8). Archived under `docs/archive/post_mvp/`.
- MVP (phases 1-10). Archived under `docs/archive/mvp/`.
