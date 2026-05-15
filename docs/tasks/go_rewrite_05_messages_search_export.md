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

- [ ] Implement message response structs matching the current JSON shape.
- [ ] Implement query parsing for: - `sender` - `channel` - `q` - `start` - `end` - `cursor` - `limit` - `reply_only` - `emote_only`
- [ ] Preserve sender case-insensitive exact matching.
- [ ] Preserve channel matching behavior.
- [ ] Preserve content search behavior.
- [ ] Preserve date range behavior.
- [ ] Preserve newest-first ordering.
- [ ] Preserve cursor format `message_created_at|message_id`.
- [ ] Implement `GET /messages`.
- [ ] Implement `GET /messages/export?format=json`.
- [ ] Implement `GET /messages/export?format=csv`.
- [ ] Clamp export rows to configured max rows.
- [ ] Decode JSON columns into current nested response fields.
- [ ] Render empty result behavior with `items: []` and `next_cursor: null`.

## Tests And Checks

- [ ] Search without filters returns newest rows.
- [ ] Sender exact match does not return partial username matches.
- [ ] Channel and content filters combine correctly.
- [ ] Date range filters return only rows inside the range.
- [ ] Reply-only filter returns only reply messages.
- [ ] Emote-only filter returns only rows with parsed emotes.
- [ ] Cursor pagination does not duplicate or skip rows in deterministic fixtures.
- [ ] JSON export response matches current shape.
- [ ] CSV export column order matches the contract inventory.
- [ ] Public access requires no auth cookie.

## Acceptance Criteria

- [ ] `/search` can use the Go API for historical message search.
- [ ] Infinite scroll works through existing `next_cursor` behavior.
- [ ] Export buttons work without frontend contract changes.
- [ ] Query performance is acceptable on realistic local seed data.

## Commit Boundary

Commit message search/export parity before listener ingestion work.
