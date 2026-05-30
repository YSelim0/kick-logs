# Client-Side Prediction Migration (Issue #19)

## Summary

Move Kick prediction fetching and normalization from the Go API proxy to the browser. The Go
endpoint `GET /channels/{slug}/prediction` is removed; the frontend calls Kick's public endpoints
directly and keeps the exact same normalized `Prediction` shape the UI already consumes. This is a
data-source migration only — no UI, layout, or behavior change.

## Why

- Removes live prediction polling load from the Go API (the page polls every 5s).
- Keeps prediction traffic between the visitor's browser and Kick.
- Avoids designing backend rate limits/cache only for prediction polling (relevant to #20, which now
  excludes prediction).
- Keeps the Go API focused on logged chat data, admin, analytics, and ingestion.

## Decision (supersedes the 2026-05-26 backend-proxy decision)

- Prediction is **client-side only**. The earlier decision to proxy through the Go API (to avoid CORS
  and Kick's "Request blocked by security policy") is reversed for this feature.
- Accepted risk: Kick's browser-accessible endpoints are undocumented; if Kick changes CORS, shape,
  or blocking behavior, the page fails with the existing clean error state and there is no backend
  fallback. Mitigation: all direct-Kick logic stays isolated behind `getPrediction(slug)`, so a
  backend proxy or feature flag can be restored later without touching the UI.

## Current State

- Frontend contract: `getPrediction(slug): Promise<Prediction>` in
  `apps/web/src/features/prediction/api.ts`, consumed only by
  `apps/web/src/features/prediction/prediction-analysis-page.tsx`.
- The page maps errors by `ApiClientError.status`: `404` -> not-found state, anything else -> error
  state. The new browser client must throw `ApiClientError` with the same status semantics so the
  page and its tests stay unchanged.
- `Prediction` / `PredictionOutcome` / `PredictionTopUser` types live in `apps/web/src/types/api.ts`
  (camelCase, derived `totalPoints`/`totalVotes`/`pointShare`/`isWinner`).
- Go normalization to port (from `usecase/predictions` + `infra/kick/prediction_resolver.go`):
  - `totalPoints = Σ outcome.total_vote_amount`, `totalVotes = Σ outcome.vote_count`.
  - `pointShare = total_vote_amount / totalPoints` (0 when `totalPoints == 0`).
  - `isWinner = winning_outcome_id != "" && outcome.id == winning_outcome_id`.
  - slug normalized to trimmed lowercase, max length 160.
- Kick endpoints:
  - `GET https://kick.com/api/v2/channels/{slug}` — channel existence.
  - `GET https://kick.com/api/v2/channels/{slug}/predictions/latest` — latest prediction.
  - Error mapping: channel `404` and null/absent `prediction` -> `404` (not-found state);
    non-2xx, non-empty top-level `error`, network, or malformed body -> non-404 (error state).

## Changes

### Commit 1 — `docs(plan): plan client-side prediction migration`

- Replace `docs/implementation_plan.md` with this plan (old prediction-backend plan removed).

### Commit 2 — `feat(web): fetch predictions directly from the browser`

- [NEW] `apps/web/src/features/prediction/kick-prediction-client.ts`
  - `fetchKickPrediction(slug, fetchImpl = fetch, signal?)`: normalize slug, validate channel via
    `channels/{slug}`, fetch `predictions/latest`, normalize snake_case -> `Prediction`, derive
    totals/share/winner, map failures to `ApiClientError` (404 vs non-404).
- [MODIFY] `apps/web/src/features/prediction/api.ts`
  - `getPrediction(slug)` calls `fetchKickPrediction(slug)`; same name, same `Promise<Prediction>`
    result. Drop the old `apiClient` dependency.
- [NEW] `apps/web/src/features/prediction/kick-prediction-client.test.ts`
  - snake_case normalization; totals/share/winner derivation; zero-points divide guard; null
    prediction -> 404; channel 404 -> 404; blocked/non-2xx -> non-404; tolerate `payload.prediction`
    and `payload.data.prediction` containers.
- `prediction-analysis-page.tsx` and its test: unchanged (contract preserved).

### Commit 3 — `feat(api): remove prediction proxy endpoint`

- [DELETE] `apps/api-go/internal/domain/prediction.go`
- [DELETE] `apps/api-go/internal/ports/prediction.go`
- [DELETE] `apps/api-go/internal/infra/kick/prediction_resolver.go`
- [DELETE] `apps/api-go/internal/usecase/predictions/` (`doc.go`, `service.go`, `service_test.go`)
- [DELETE] `apps/api-go/internal/http/routes/predictions.go`
- [DELETE] `apps/api-go/internal/http/prediction_routes_test.go`
- [MODIFY] `apps/api-go/cmd/api/main.go` — drop `predictionsusecase` import, the
  `predictionService` construction, and the `Predictions:` field. Keep `kick` import
  (`NewWebChannelResolver`).
- [MODIFY] `apps/api-go/internal/http/server.go` — remove `routes.RegisterPredictionRoutes`.
- [MODIFY] `apps/api-go/internal/http/routes/dependencies.go` — drop `predictionsusecase` import and
  the `Predictions` field.

### Commit 4 — `docs: document client-side prediction`

- [MODIFY] `docs/architecture.md` — remove `GET /channels/{slug}/prediction` from the API surface and
  the prediction-proxy paragraph; note prediction is client-side only.
- [MODIFY] `docs/design/design.md` — prediction page fetches Kick directly from the browser.
- [MODIFY] `docs/context/living_brain.md` — rewrite the Prediction section; drop the endpoint from the
  API contract list.
- [MODIFY] `docs/context/decisions.md` — dated entry reversing the backend-proxy decision.
- [MODIFY] `docs/context/change_log.md` + `docs/context/recent_changes.md` — chronological entry +
  handoff summary.

## Verification (per relevant commit)

```powershell
# Frontend (commit 2)
pnpm --filter @kick-logs/web typecheck
pnpm --filter @kick-logs/web lint
pnpm --filter @kick-logs/web test
pnpm --filter @kick-logs/web build
pnpm format:check

# Go (commit 3)
cd apps/api-go
gofmt -l .
go vet ./...
go test ./...
```

Manual (post-deploy): the browser Network tab shows direct calls to
`https://kick.com/api/v2/channels/{slug}` and `.../predictions/latest`, and no longer shows
`GET /channels/{slug}/prediction`; `/prediction/{slug}` still renders and refreshes every 5s with
loading/not-found/error states intact.

## Completed Plans

- Prediction feature (backend proxy). Superseded by this client-side migration.
- Channels & Users index search pages. (Historical.)
- Frontend v2 re-skin (all six routes). Archived under `docs/archive/redesign/`.
- Issue #11: Worker hot-path optimizations. PR #12 merged.
- Issue #9: High-volume ingestion batching/backpressure. Archived under `docs/archive/issue_09/`.
- Go + ClickHouse rewrite (phases 1-9). Archived under `docs/archive/go_rewrite/`.
- Post-MVP features (1-8). Archived under `docs/archive/post_mvp/`.
- MVP (phases 1-10). Archived under `docs/archive/mvp/`.
