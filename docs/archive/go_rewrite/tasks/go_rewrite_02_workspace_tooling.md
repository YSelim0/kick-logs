# Go Rewrite Phase 2: Workspace And Tooling

## Scope

Create the Go backend workspace and minimal service skeleton without replacing the Python runtime.

This phase owns build tooling, baseline config, health endpoint, logging, and local commands.

## Out Of Scope

- Do not implement auth/admin business behavior beyond health/config scaffolding.
- Do not implement ClickHouse or SQLite schemas except placeholder wiring if needed.
- Do not remove Python services.
- Do not modify frontend feature behavior.

## Checklist

- [x] Add `apps/api-go/go.mod` and `go.sum`.
- [x] Add command entrypoints: - `cmd/api` - `cmd/listener` - `cmd/migrate`
- [x] Add internal package skeleton: - config - app bootstrap - HTTP server - middleware - domain - ports - use cases - infrastructure
- [x] Implement environment config loading with clear defaults for local development.
- [x] Implement structured logging for API and listener processes.
- [x] Implement `GET /health` in the Go API.
- [x] Add a Go Dockerfile or multi-stage build target.
- [x] Add Compose wiring for optional Go API service while Python remains available.
- [x] Add local commands in README or package scripts if needed: - test - lint or vet - build - run api - run listener - run migrations
- [x] Add `.gitignore` entries for Go build outputs if necessary.

## Tests And Checks

- [x] `go test ./...` passes.
- [x] `go vet ./...` passes or an equivalent baseline check is documented.
- [x] Go API health route returns the expected health response locally.
- [x] Docker build for the Go API succeeds.

## Acceptance Criteria

- [x] The Go workspace builds cleanly.
- [x] The Go API can run without replacing the Python API.
- [x] The health endpoint is available and contract-compatible.
- [x] No Python runtime behavior is removed.

## Commit Boundary

Commit this phase as the Go workspace and tooling bootstrap.
