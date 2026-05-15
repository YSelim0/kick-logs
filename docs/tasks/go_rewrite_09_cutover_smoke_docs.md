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

- [ ] Update Compose so default `api` uses the Go API.
- [ ] Update Compose so default `listener` uses the Go listener.
- [ ] Keep a documented way to run the old Python/PostgreSQL reference runtime until final removal.
- [ ] Update environment examples for ClickHouse and SQLite settings.
- [ ] Update README quick start.
- [ ] Update README migration instructions.
- [ ] Update README backup/restore notes for ClickHouse and SQLite.
- [ ] Update admin/data-management docs for ClickHouse mutation behavior.
- [ ] Update architecture docs to reflect Go + ClickHouse + SQLite.
- [ ] Update context docs with final cutover state.
- [ ] Decide whether Python backend stays archived in-repo or is removed in a later cleanup commit.
- [ ] Decide when PostgreSQL service is removed from Compose defaults.

## Smoke Test Checklist

- [ ] `docker compose up --build -d` starts ClickHouse, Go API, Go listener, and web.
- [ ] `GET /health` succeeds.
- [ ] Default super admin can log in.
- [ ] Admin can list channels.
- [ ] Admin can add a channel.
- [ ] Admin can disable a channel.
- [ ] Go listener records heartbeat.
- [ ] Fixture or live message ingestion writes searchable messages.
- [ ] `/search` can query by sender exact match.
- [ ] `/search` can query by channel and content.
- [ ] Reply messages render reply metadata.
- [ ] Emote messages render emote image metadata.
- [ ] JSON export works.
- [ ] CSV export works.
- [ ] Landing analytics load.
- [ ] User profile analytics load.
- [ ] Channel profile analytics load.
- [ ] Admin operations dashboard loads.
- [ ] Admin data-management summary loads.
- [ ] Cleanup preview works.
- [ ] Public routes remain unauthenticated.
- [ ] Admin routes reject unauthenticated requests.

## Acceptance Criteria

- [ ] The default self-hosted startup path uses Go and ClickHouse.
- [ ] Existing frontend workflows work against the Go API.
- [ ] Migration and rollback notes are documented.
- [ ] Python/PostgreSQL cleanup is left as an explicit final decision, not an accidental deletion.

## Commit Boundary

Commit cutover and docs after the full smoke checklist passes.
