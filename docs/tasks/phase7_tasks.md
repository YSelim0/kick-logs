# Phase 7 Tasks: Frontend Foundation

## Scope

Scaffold the frontend project and shared API foundation after backend acceptance. This phase owns `apps/web`, pnpm workspace configuration, styling/tooling setup, route shells, and API client/types.

Do not implement final `/search` UI, admin dashboard workflows, or new backend behavior in this phase.

## Inputs

- Accepted backend from Phase 6.
- API contracts from `docs/project_plan.md` and `docs/architecture.md`.
- UI rules from `docs/design/design.md`.

## Tasks

- [ ] Workspace setup:
  - [ ] Add root `package.json` and `pnpm-workspace.yaml`.
  - [ ] Scaffold `apps/web` with Next.js App Router and TypeScript.
  - [ ] Add scripts for dev, build, lint, and typecheck.
- [ ] Styling/tooling:
  - [ ] Configure Tailwind.
  - [ ] Install lucide-react.
  - [ ] Add shadcn/ui base setup.
  - [ ] Store palette tokens from `docs/design/design.md`.
  - [ ] Keep app dark-only; no theme switcher.
- [ ] Route shells:
  - [ ] `/` reserved placeholder only.
  - [ ] `/search` placeholder route noting public route behavior.
  - [ ] `/login` placeholder route.
  - [ ] `/admin` placeholder route with no final dashboard UI yet.
- [ ] Shared API layer:
  - [ ] `lib/api-client.ts` with base URL and credential handling.
  - [ ] Shared API response/error types.
  - [ ] Typed functions for health/auth/messages/admin endpoints, but no feature UI wiring yet.
- [ ] Docker:
  - [ ] Add `web` service to Compose only after `apps/web` can run.
  - [ ] Ensure `web` depends on API URL config, not on internal hardcoded localhost.
- [ ] Tests/checks:
  - [ ] Typecheck passes.
  - [ ] Build or lint passes according to configured scripts.
  - [ ] API client health call can be tested or mocked.

## Acceptance Criteria

- [ ] `pnpm install` and frontend scripts are documented.
- [ ] `apps/web` compiles.
- [ ] Routes exist but final search/admin UI is not implemented.
- [ ] Shared API client is ready for Phase 8 and Phase 9.
- [ ] No backend files are changed except docs/config wiring required for web service.

## Handoff

Phase 8 can implement `/search` using the shared API client, design docs, and backend message search contract.
