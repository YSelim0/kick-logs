# Phase 7 Tasks: Frontend Foundation

## Scope

Scaffold the frontend project and shared API foundation after backend acceptance. This phase owns `apps/web`, pnpm workspace configuration, styling/tooling setup, route shells, and API client/types.

Do not implement final `/search` UI, admin dashboard workflows, or new backend behavior in this phase.

## Inputs

- Accepted backend from Phase 6.
- API contracts from `docs/project_plan.md` and `docs/architecture.md`.
- UI rules from `docs/design/design.md`.

## Tasks

- [x] Workspace setup:
  - [x] Add root `package.json` and `pnpm-workspace.yaml`.
  - [x] Scaffold `apps/web` with Next.js App Router and TypeScript.
  - [x] Add scripts for dev, build, lint, and typecheck.
- [x] Styling/tooling:
  - [x] Configure Tailwind.
  - [x] Install lucide-react.
  - [x] Add shadcn/ui base setup.
  - [x] Store palette tokens from `docs/design/design.md`.
  - [x] Keep app dark-only; no theme switcher.
- [x] Route shells:
  - [x] `/` reserved placeholder only.
  - [x] `/search` placeholder route noting public route behavior.
  - [x] `/login` placeholder route.
  - [x] `/admin` placeholder route with no final dashboard UI yet.
- [x] Shared API layer:
  - [x] `lib/api-client.ts` with base URL and credential handling.
  - [x] Shared API response/error types.
  - [x] Typed functions for health/auth/messages/admin endpoints, but no feature UI wiring yet.
- [x] Docker:
  - [x] Add `web` service to Compose only after `apps/web` can run.
  - [x] Ensure `web` depends on API URL config, not on internal hardcoded localhost.
- [x] Tests/checks:
  - [x] Typecheck passes.
  - [x] Build or lint passes according to configured scripts.
  - [x] API client health call can be tested or mocked.

## Acceptance Criteria

- [x] `pnpm install` and frontend scripts are documented.
- [x] `apps/web` compiles.
- [x] Routes exist but final search/admin UI is not implemented.
- [x] Shared API client is ready for Phase 8 and Phase 9.
- [x] No backend files are changed except docs/config wiring required for web service.

## Handoff

Phase 8 can implement `/search` using the shared API client, design docs, and backend message search contract.
