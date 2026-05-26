# Prediction Feature — Channel Search + Latest Prediction Analysis

## Summary

Add a first-class **Prediction** feature to `kick-logs`, ported from the `kick-prediction`
prototype but rebuilt to the project's architecture and design system.

Flow:

```text
Header "Prediction" button -> /prediction
/prediction search submit    -> /prediction/{channel_slug}
/prediction/{channel_slug}   -> latest Kick prediction analysis for that channel
```

`/prediction` is a search-first page only (no analytics on it). `/prediction/{slug}` fetches the
channel's latest prediction through a **new backend endpoint** and renders summary, charts, outcome
detail cards, and a top-users chart using the existing v2 design tokens.

## Decisions (locked with the owner)

- **Backend proxy, not client-side fetch.** Kick's prediction endpoint sends no CORS headers for the
  web origin and can return `Request blocked by security policy` to plain requests. The frontend
  calls a Kick Logs endpoint; the backend owns the Kick call, browser-like headers, normalization,
  and error mapping. This matches `infra/kick` (`channel_resolver.go`) and the brief's recommended
  shape.
- **Live fetch only.** Each `/prediction/{slug}` view fetches the latest prediction on demand. No new
  ClickHouse/SQLite tables, no migration, no write path. The endpoint is stateless.
- **recharts for charts.** A charting dependency is added (the repo currently has none; landing bars
  are hand-rolled CSS). recharts renders the donut, grouped bar, and horizontal bar charts. Chart
  colors are tokenized, not the prototype's hardcoded hues.
- **Turkish UI, v2 tokens.** Strings follow the existing Turkish UI; the prototype's gray/green
  palette is dropped in favor of `bg-page`/`bg-panel`/`bg-elevated`/`border-subtle`/accent green.

## Current State

- `/users` and `/channels` are search-first index pages — reuse their submit-only form pattern for
  `/prediction`.
- `infra/kick/channel_resolver.go` already fetches `https://kick.com/api/v2/channels/{slug}` with a
  browser `user-agent`; the prediction fetcher follows the same adapter shape with the extra
  browser-like headers the brief documents (`Referer`, `Accept-Language`, `sec-ch-ua`, fetch
  metadata).
- HTTP routes register through `internal/http/server.go`; use cases are wired in `cmd/api/main.go`
  and exposed via `routes.Dependencies`.
- `next.config.mjs` already allows `files.kick.com` and `kick.com` image hosts (no change needed for
  Kick avatars/images, though the prediction payload exposes no images).
- `SiteHeader` `ActiveRoute` is `"search" | "channels" | "users" | "admin" | null`; nav items live in
  `NAV_ITEMS`.
- Prettier: `.prettierrc.json` (100 col, LF, no trailing comma). CI: `code-style.yml` →
  `pnpm format:check`, `go-tests.yml` → `go test ./...`.

---

## Proposed Changes

### Backend — Prediction Fetch + Normalize + Route

#### [MODIFY] `apps/api-go/internal/domain/models.go` (or [NEW] `domain/prediction.go`)

- Add domain types:
  - `Prediction` — `ID`, `ChannelID`, `Title`, `DurationSeconds`, `State`, `WinningOutcomeID`,
    `CreatedAt`, `LockedAt` (nullable), `UpdatedAt`, plus derived `TotalPoints`, `TotalVotes`,
    `Outcomes`.
  - `PredictionOutcome` — `ID`, `Title`, `TotalVoteAmount`, `VoteCount`, `ReturnRate`, derived
    `PointShare`, `IsWinner`, `TopUsers`.
  - `PredictionTopUser` — `ID`, `Username`, `Amount`.
- Domain stays free of HTTP/Kick imports (architecture rule).

#### [NEW] `apps/api-go/internal/ports/prediction.go`

- `PredictionFetcher` interface: `LatestPrediction(ctx, slug string) (domain.Prediction, error)`.
- Sentinel errors for the use case to map: `ErrPredictionNotFound` (no prediction / `data.prediction`
  is null), `ErrPredictionBlocked` (Kick security policy / non-2xx), `ErrChannelNotFound` (404).

#### [NEW] `apps/api-go/internal/infra/kick/prediction_resolver.go`

- `WebPredictionResolver` mirroring `WebChannelResolver`: `http.Client` with timeout, `baseURL`
  `https://kick.com`.
- `GET {baseURL}/api/v2/channels/{slug}/predictions/latest` with browser-like headers
  (`user-agent`, `accept`, `Referer` `https://kick.com/{slug}`, `Accept-Language`, `sec-ch-ua`,
  `sec-fetch-*`).
- Decode `{ "data": { "prediction": {...} }, "message": "..." }`. A null/absent `prediction`
  maps to `ErrPredictionNotFound`.
- Detect the blocked-body shape (`{"error":"Request blocked by security policy.",...}`) and non-2xx
  status → `ErrPredictionBlocked`; 404 → `ErrChannelNotFound`.
- Map raw Kick fields → domain (`total_vote_amount`, `vote_count`, `return_rate`, `top_users`).
- Treat the endpoint as undocumented/unstable: tolerate missing optional fields, no panics on empty
  `outcomes`.

#### [NEW] `apps/api-go/internal/usecase/predictions/service.go`

- `Service` holding a `PredictionFetcher`.
- `LatestPrediction(ctx, slug)`:
  - Trim/validate slug (reuse the profile slug rules: non-empty, length-bounded).
  - Call the fetcher.
  - Derive server-side: `TotalPoints = Σ outcome.TotalVoteAmount`, `TotalVotes = Σ outcome.VoteCount`,
    per-outcome `PointShare = TotalVoteAmount / TotalPoints` (guard divide-by-zero),
    `IsWinner = outcome.ID == WinningOutcomeID`.
  - Re-export the fetcher sentinel errors for the route layer.

#### [NEW] `apps/api-go/internal/http/routes/predictions.go`

- `RegisterPredictionRoutes(mux, deps)` → `GET /channels/{slug}/prediction` (public, no auth).
- Reuse `profileSlug` validation.
- Response schema struct (explicit Go struct, JSON `camelCase` matching the frontend model in the
  brief: `totalPoints`, `pointShare`, `isWinner`, `topUsers`, etc.).
- Error mapping:
  - `ErrPredictionNotFound` → `404` `"No active prediction found for this channel."`
  - `ErrChannelNotFound` → `404` `"Channel not found."`
  - `ErrPredictionBlocked` → `502` `"Kick prediction request was blocked. Try again later."`
  - invalid slug → `422`; nil service / other → `500`.

#### [MODIFY] `apps/api-go/internal/http/routes/dependencies.go`

- Add `Predictions *predictionsusecase.Service` to `Dependencies`.

#### [MODIFY] `apps/api-go/internal/http/server.go`

- Call `routes.RegisterPredictionRoutes(mux, deps)`.

#### [MODIFY] `apps/api-go/cmd/api/main.go`

- Build `predictionsusecase.NewService(kick.NewWebPredictionResolver())` and pass it in
  `routes.Dependencies{ Predictions: ... }`. Independent of ClickHouse (always wired).

#### [NEW] Backend unit tests

- `usecase/predictions` test: normalization (totals, point share, winner flag, divide-by-zero
  guard) using a fake fetcher.
- `http/routes` test: success JSON shape; 404 no-prediction; 404 channel-not-found; 502 blocked;
  422 invalid slug — using a fake fetcher injected through `Dependencies`.

---

### Frontend — Route Pages

#### [NEW] `apps/web/src/app/prediction/page.tsx`

- Route handler with SEO metadata rendering `PredictionSearchPage`. `activeRoute="prediction"`.

#### [NEW] `apps/web/src/app/prediction/[slug]/page.tsx`

- Route handler reading the `slug` param, rendering `PredictionAnalysisPage`.

---

### Frontend — Feature Components (`apps/web/src/features/prediction/`)

#### [NEW] `api.ts`

- `getPrediction(slug, client = apiClient)` → `client.get<PredictionResponse>(\`/channels/${slug}/prediction\`)`.

#### [NEW] types (in `apps/web/src/types/api.ts` or `features/prediction/types.ts`)

- `PredictionResponse`, `PredictionOutcome`, `PredictionTopUser` matching the backend camelCase
  schema. Centralized per the design "reuse response types" rule.

#### [NEW] `prediction-search-page.tsx`

- `"use client"`. Search-first, same submit-only contract as `/channels` and `/users`:
  - `<form onSubmit>` with one channel-name input + a `Göster` submit button.
  - Trim + lowercase the slug; submit navigates with `router.push(\`/prediction/${slug}\`)`. It does
    **not** fetch on this page.
  - Submit disabled until trimmed input ≥ 2 chars.
  - Idle prompt centered icon + "Kanal adı girin ve tahmin verisini görün." (token styling).
- Uses `SiteHeader activeRoute="prediction"`.

#### [NEW] `prediction-analysis-page.tsx`

- `"use client"`. On mount, fetch `getPrediction(slug)`.
- After the first successful load, poll `getPrediction(slug)` every 5 seconds as a background
  refresh while the page remains open, including after `LOCKED`, `CANCELED`/`CANCELLED`, or
  `RESOLVED`. Keep existing content mounted during the refresh to avoid chart/list flash.
- Header strip: breadcrumb-style `kick.com/{slug}` (mono; `kick.com/` in accent, slug in
  `text-primary`) + a refresh button that re-runs the fetch (per prototype detail header, design-system
  styled).
- States:
  - Loading: centered muted message.
  - Error (502/network): `danger`-toned panel with primary + helper line.
  - Not found (404): calm not-found panel with a link back to `/prediction` (reuse profile
    not-found tone).
- Content blocks (the brief's three conceptual groups):
  1. **Summary card** (`bg-panel`): title + state pill; 4-cell metric row (total points, total
     votes, duration, created). Lock timestamp line when `lockedAt` set.
  2. **Distribution + comparison charts** (two panels, responsive stack):
     `PredictionDistributionChart` (donut) and `PredictionVoteReturnChart` (grouped bar).
  3. **Outcome detail cards** + **top-users chart** (`PredictionTopUsersChart`, horizontal bar).
- State-pill mapping (no blue/purple — those are not in the palette):
  - `RESOLVED` → accent green pill `Sonuçlandı`.
  - `LOCKED` → `warning` pill `Kilitli`.
  - `CANCELED` / `CANCELLED` → `warning` pill `İptal`.
  - `ACTIVE` → neutral pill (`bg-elevated` + `text-secondary`) `Aktif`.
  - unknown → `bg-elevated` + `text-muted`, raw state text.
- Number formatting helper (compact `21.8K`, `2.72x` multiplier, percent share) lives in a small
  `format.ts` in the feature folder.

#### [NEW] Chart components (recharts)

- `prediction-distribution-chart.tsx` — donut/pie of point share per outcome; slice label = outcome
  name + percent; tooltip = formatted points.
- `prediction-vote-return-chart.tsx` — grouped bar: vote count series + return-rate series per
  outcome.
- `prediction-top-users-chart.tsx` — horizontal bar of top users across outcomes, bar color by
  owning outcome, legend mapping color → outcome.
- All three consume a shared **tokenized categorical palette** (see design.md update), wrap charts in
  recharts `ResponsiveContainer`, and use `bg-panel`/`text-secondary`/`border-subtle` for axes,
  grid, and tooltip chrome. Understandable without color alone (labels/legends always present).

---

### Frontend — Feature Tests

#### [NEW] `prediction-search-page.test.tsx`

- Idle prompt renders.
- Submit disabled under 2 chars.
- Submitting a slug calls `router.push("/prediction/{slug}")` with trimmed/lowercased value (mock
  `next/navigation`).

#### [NEW] `prediction-analysis-page.test.tsx`

- Loading → resolved summary (title, state pill, totals) renders on a mocked `getPrediction`.
- 404 → not-found panel with back link.
- 502/error → error panel.
- Outcome cards render winner badge `KAZANAN` for the winning outcome.
- recharts is mocked/shimmed in jsdom (ResponsiveContainer has no layout in jsdom) — assert on the
  surrounding labels/data, not SVG geometry.

---

### Frontend — Site Header + Dependency

#### [MODIFY] `apps/web/src/components/site-header.tsx`

- Extend `ActiveRoute` with `"prediction"`.
- Add `{ route: "prediction", label: "Prediction", href: "/prediction" }` to `NAV_ITEMS`.

#### [MODIFY] `apps/web/package.json`

- Add `recharts` to dependencies; install with pnpm so the lockfile updates.

---

### Docs — Design & Context Updates

#### [MODIFY] `docs/design/design.md`

- Add `/prediction` and `/prediction/{slug}` to Routes + Index Page UX rules (search-first,
  submit-navigates).
- Add a **Prediction page** section: summary card, chart row, outcome cards, top-users chart,
  state-pill mapping, refresh control.
- Add a **Chart palette** decision: a tokenized categorical sequence for multi-series charts
  (rooted in `accent` + neutral tints + `warning`/`danger`, since the base palette has no
  blue/purple/cyan). recharts is now an allowed dependency.
- Header nav: `Prediction` link active; `ActiveRoute` includes `"prediction"`.

#### [MODIFY] `docs/architecture.md`

- Add `GET /channels/{slug}/prediction` to the public API surface and note the `infra/kick`
  prediction adapter + `usecase/predictions`.

#### [MODIFY] `docs/context/living_brain.md`

- Document the prediction feature: backend-owned live fetch, new endpoint, recharts, no persistence.

#### [MODIFY] `docs/context/decisions.md`

- Backend proxy over client fetch (CORS + security policy), live-fetch-only (no storage), recharts
  adoption, tokenized chart palette, state-pill mapping without blue.

#### [MODIFY] `docs/context/change_log.md` and `docs/context/recent_changes.md`

- Chronological entry + handoff summary.

---

## What Is Explicitly NOT Carried Over

- The prototype's standalone Next app, default CRA README, direct browser→Kick fetch, hardcoded
  chart hues, and the all-in-one client page. Responsibilities are split: kick adapter → use case →
  HTTP route → frontend api wrapper → UI components.

## Commit Plan (feature-sized units)

1. `feat(api): add kick prediction fetch endpoint` — domain types, port, kick adapter, use case,
   route, wiring, backend tests.
2. `feat(web): add prediction search and analysis pages` — recharts dep, routes, feature components,
   charts, header nav, frontend tests.
3. `docs: document prediction feature` — design + context docs.

(Split may adjust if a unit grows; each unit commits only after its checks pass.)

## Verification Plan

### Automated

```powershell
# Backend
cd apps/api-go
go test ./...
go vet ./...

# Frontend
pnpm --filter @kick-logs/web typecheck
pnpm --filter @kick-logs/web lint
pnpm --filter @kick-logs/web test
pnpm --filter @kick-logs/web build
pnpm format:check
```

### Manual

- `GET http://localhost:8000/channels/{slug}/prediction` returns normalized JSON for a channel with
  a recent prediction; returns 404 for a channel with none; 502 when Kick blocks.
- `http://localhost:3000/prediction` → search-first screen; submitting a slug navigates to
  `/prediction/{slug}`.
- `/prediction/{slug}` renders summary, charts, outcome cards, and top-users chart; loading, error,
  and not-found states behave.
- `Prediction` nav link visible and active in the header.
- Responsive (mobile/desktop) chart + card layout.

## Completed Plans

- Channels & Users index search pages. (Now historical.)
- Frontend v2 re-skin (all six routes). Archived under `docs/archive/redesign/`.
- Issue #11: Worker hot-path optimizations. PR #12 merged.
- Issue #9: High-volume ingestion batching/backpressure. Archived under `docs/archive/issue_09/`.
- Go + ClickHouse rewrite (phases 1-9). Archived under `docs/archive/go_rewrite/`.
- Post-MVP features (1-8). Archived under `docs/archive/post_mvp/`.
- MVP (phases 1-10). Archived under `docs/archive/mvp/`.
