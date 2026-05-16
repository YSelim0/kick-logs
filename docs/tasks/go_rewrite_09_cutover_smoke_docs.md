# Go Rewrite Phase 9: Cutover, Smoke, And Docs

## Scope

Switch the default local runtime to the Go API/listener and ClickHouse/SQLite storage after parity
and migration are verified.

This phase owns Docker Compose cutover, smoke testing, README updates, and cleanup decisions.

## Out Of Scope

- Do not remove Python source until the user explicitly accepts parity and cutover.
- Do not delete PostgreSQL volumes.
- Do not introduce new product features.
- Do not redesign the frontend.

## Checklist

- [x] Update Compose so default `api` uses the Go API.
- [x] Update Compose so default `listener` uses the Go listener.
- [x] Keep a documented way to run the old Python/PostgreSQL reference runtime until final removal.
- [x] Update environment examples for ClickHouse and SQLite settings.
- [x] Update README quick start.
- [x] Update README migration instructions.
- [x] Update README backup/restore notes for ClickHouse and SQLite.
- [x] Update admin/data-management docs for ClickHouse mutation behavior.
- [x] Update architecture docs to reflect Go + ClickHouse + SQLite.
- [x] Update context docs with final cutover state.
- [x] Decide whether Python backend stays archived in-repo or is removed in a later cleanup commit.
- [x] Decide when PostgreSQL service is removed from Compose defaults.

## Smoke Test Checklist

- [x] `docker compose up --build -d` starts ClickHouse, Go API, Go listener, and web.
- [x] `GET /health` succeeds.
- [x] Default super admin can log in.
- [x] Admin can list channels.
- [x] Admin can add a channel.
- [x] Admin can disable a channel.
- [x] Go listener records heartbeat.
- [x] Fixture or live message ingestion writes searchable messages.
- [x] `/search` can query by sender exact match.
- [x] `/search` can query by channel and content.
- [x] Reply messages render reply metadata.
- [x] Emote messages render emote image metadata.
- [x] JSON export works.
- [x] CSV export works.
- [x] Landing analytics load.
- [x] User profile analytics load.
- [x] Channel profile analytics load.
- [x] Admin operations dashboard loads.
- [x] Admin data-management summary loads.
- [x] Cleanup preview works.
- [x] Public routes remain unauthenticated.
- [x] Admin routes reject unauthenticated requests.

## Acceptance Criteria

- [x] The default self-hosted startup path uses Go and ClickHouse.
- [x] Existing frontend workflows work against the Go API.
- [x] Migration and rollback notes are documented.
- [x] Python/PostgreSQL cleanup is left as an explicit final decision, not an accidental deletion.

## Commit Boundary

Commit cutover and docs after the full smoke checklist passes.
