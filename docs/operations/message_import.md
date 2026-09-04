# One-Off Message Import

`cmd/importmessages` backfills `chat_messages` in ClickHouse from a JSON message
export (the `{items, count, max_rows, truncated}` shape the app's own message
export/search paths produce). It is append-only:

- Existing rows are matched by `kick_message_id` and are **never** updated.
- Nothing is written to ClickHouse unless `-dry-run=false` is combined with the
  exact `-confirm=IMPORT-CHAT-MESSAGES` phrase.
- The JSON export is the source of truth for the fields it carries (message,
  channel, sender, emotes, `reply_metadata`, `message_created_at`). A CSV
  export of the same dataset can be cross-checked with `-verify-csv`, but it
  is only used as an informational sanity check, never as an import input.

## Prerequisites

Drop the export file(s) under `./import/` at the repo root on the target
machine (bind-mounted read-only into the tool's container; see `compose.yaml`
`import-messages` service, `tools` profile). This directory is gitignored
because export files can contain real chat data.

## Back up first

Even though this tool only inserts rows for ids that do not already exist, do
a ClickHouse backup before the first real run so the operation is trivially
reversible:

```bash
docker compose down
BACKUP_DIR="$HOME/kick-logs-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"
docker run --rm -v kick-logs_clickhouse_data:/volume -v "$BACKUP_DIR":/backup alpine \
  tar -czf /backup/clickhouse_data.tar.gz -C /volume .
docker compose up -d
```

See `docs/operations/backup_restore.md` for the full multi-volume procedure
and restore steps.

## Dry run (always do this first)

```bash
docker compose --profile tools run --rm import-messages \
  -input /import/export.json \
  -dry-run \
  -limit 20 \
  -verify-csv /import/export.csv
```

`-dry-run` defaults to `true`, so omitting it is also safe. The tool prints:

- `records_read` — how many rows were read (after `-limit`, if set).
- `would_insert` — new rows not yet present in ClickHouse.
- `already_in_clickhouse` — rows whose `kick_message_id` already exists;
  these are skipped and never touched.
- `duplicate_within_file` — rows whose `kick_message_id` repeats inside the
  input file itself; only the first occurrence is ever considered.
- `invalid_records` — rows that failed validation (missing
  `kick_message_id`, unparseable `message_created_at`, missing channel slug,
  or missing sender identity), grouped by reason with a sample row.

Re-run without `-limit` (or with a higher one) once the small sample looks
correct.

## Real import

Only after reviewing a dry-run report:

```bash
docker compose --profile tools run --rm import-messages \
  -input /import/export.json \
  -dry-run=false \
  -confirm=IMPORT-CHAT-MESSAGES
```

Omit `-limit` to import everything, or keep it to import in controlled
batches. The final log line reports `written`, plus the same
`skipped_already_existed` / `skipped_duplicate_in_file` / `skipped_invalid`
breakdown as the dry run.

## Field mapping notes

- Row id: `fnv64a(kick_message_id)` masked into an `int63`, identical to the
  live listener's `deterministicMessageID` (`internal/usecase/listener/normalizer.go`),
  so backfilled rows use the same id a live-ingested duplicate would have
  produced.
- `reply_metadata` is preserved byte-for-byte from the export's
  `reply_metadata` object; `reply_to_sender` / `reply_to_content` /
  `reply_to_message_id` are derived from it for the flat columns the search
  UI reads.
- The export has no raw Kick payload, so `raw_payload_json` is stored as
  `{}` for imported rows (existing live-ingested rows are unaffected).
- `channel_kick_id` (the numeric Kick channel id) is not present in this
  export shape and is left unset; `channel_id`, `channel_chatroom_id`,
  `sender_id`, and `sender_kick_id` are populated directly from the export.
