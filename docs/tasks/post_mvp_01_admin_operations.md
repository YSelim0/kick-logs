# Post-MVP Feature 1 Tasks: Admin Operations Dashboard

## Goal

Add an authenticated operations dashboard so admins can see whether ingestion,
storage, and raw-event processing are healthy.

## Scope

This feature owns operational metrics only. Do not add public analytics,
search filters, data deletion, retention settings, or landing page work here.

## Backend Tasks

- [x] Add persistence for listener heartbeat/freshness if no existing DB state can
      reliably answer whether the listener is alive.
- [x] Make the listener update its heartbeat at a short fixed interval while running.
- [x] Add admin-only operations summary use case.
- [x] Include total counts for channels, enabled channels, senders, messages, and raw events.
- [x] Include raw event counts grouped by status.
- [x] Include database size and key table sizes for `chat_messages` and `raw_kick_events`.
- [x] Include latest message time, latest raw event receive time, latest processed raw event
      time, and oldest pending raw event time when available.
- [x] Expose the summary through an authenticated admin route.
- [x] Add backend tests for permissions, counts, raw status aggregation, database size shape,
      and listener freshness state.

## Frontend Tasks

- [ ] Add an operations dashboard section to `/admin`.
- [ ] Show compact cards for listener status, database size, message count, raw event count,
      failed raw events, pending raw events, and last ingest time.
- [ ] Add a manual refresh action.
- [ ] Use calm warning/error states for stale listener heartbeat and failed raw events.
- [ ] Keep channel management and user management visually separate from operations metrics.
- [ ] Add frontend tests for loading, success, stale listener, failed raw events, and API error
      states.

## Docs And Acceptance

- [ ] Update README/admin usage notes with the operations dashboard.
- [ ] Update context docs.
- [ ] Verify backend tests and frontend tests/typecheck/lint for the touched areas.
- [ ] Acceptance: an authenticated admin can open `/admin` and understand storage growth,
      raw event backlog, and listener freshness without reading Docker logs.
