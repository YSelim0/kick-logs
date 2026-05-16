# Go Rewrite Phase 5: Messages, Search, And Export

## Scope

Rebuild public message search and export endpoints in Go on top of ClickHouse.

This phase owns query behavior, cursor pagination, response mapping, and CSV/JSON export.

## Out Of Scope

- Do not implement live listener ingestion yet.
- Do not implement analytics/profile endpoints except helper queries if unavoidable.
- Do not change frontend search UI behavior.
- Do not change public route access.

## Checklist

- [x] Implement message response structs matching the current JSON shape.
- [x] Implement query parsing for: - `sender` - `channel` - `q` - `start` - `end` - `cursor` - `limit` - `reply_only` - `emote_only`
- [x] Preserve sender case-insensitive exact matching.
- [x] Preserve channel matching behavior.
- [x] Preserve content search behavior.
- [x] Preserve date range behavior.
- [x] Preserve newest-first ordering.
- [x] Preserve cursor format `message_created_at|message_id`.
- [x] Implement `GET /messages`.
- [x] Implement `GET /messages/export?format=json`.
- [x] Implement `GET /messages/export?format=csv`.
- [x] Clamp export rows to configured max rows.
- [x] Decode JSON columns into current nested response fields.
- [x] Render empty result behavior with `items: []` and `next_cursor: null`.

## Tests And Checks

- [x] Search without filters returns newest rows.
- [x] Sender exact match does not return partial username matches.
- [x] Channel and content filters combine correctly.
- [x] Date range filters return only rows inside the range.
- [x] Reply-only filter returns only reply messages.
- [x] Emote-only filter returns only rows with parsed emotes.
- [x] Cursor pagination does not duplicate or skip rows in deterministic fixtures.
- [x] JSON export response matches current shape.
- [x] CSV export column order matches the contract inventory.
- [x] Public access requires no auth cookie.

## Acceptance Criteria

- [x] `/search` can use the Go API for historical message search.
- [x] Infinite scroll works through existing `next_cursor` behavior.
- [x] Export buttons work without frontend contract changes.
- [x] Query performance is acceptable on realistic local seed data.

## Commit Boundary

Commit message search/export parity before listener ingestion work.
