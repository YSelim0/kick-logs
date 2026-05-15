# Current API Contract Inventory

This document captures the Python backend contract that the Go rewrite must preserve. It is scoped
to the backend API surface consumed by the current Next.js frontend.

Source snapshot:

- Branch: `feat/go-clickhouse-rewrite`
- Date: 2026-05-15
- Backend route source: `apps/api/src/kick_logs/presentation/http/routes/`
- Backend schema source: `apps/api/src/kick_logs/presentation/http/schemas/`
- Frontend type source: `apps/web/src/types/api.ts`
- Frontend client source: `apps/web/src/lib/api-client.ts`

## Compatibility Rules

Strict compatibility:

- endpoint path and method
- request JSON field names
- query parameter names
- response JSON field names and nesting
- public versus admin-only access boundaries
- auth cookie name, path, HttpOnly behavior, secure flag, same-site value, and max-age semantics
- message cursor format
- CSV export column order

Best-effort compatibility:

- exact FastAPI validation error body shape for fields the frontend does not inspect
- exact server-generated timestamps in examples
- exact ordering of object keys inside JSON responses

## Frontend Client Expectations

The frontend API client:

- prefixes every request with `API_BASE_URL`
- sends `credentials: "include"` by default
- sends JSON request bodies with `content-type: application/json`
- omits query parameters whose values are `undefined`, `null`, or an empty string
- parses JSON responses when possible
- throws `ApiClientError(status, body)` for non-2xx responses
- uses `body.detail` as the user-facing error message when it is a string

## Auth Cookie Contract

Configured defaults:

- cookie name: `kick_logs_session`
- path: `/`
- `HttpOnly`: true
- `Secure`: `false` by default, env-overridable
- `SameSite`: `lax` by default, env-overridable
- max age: `JWT_EXPIRES_MINUTES * 60`
- default expiry minutes: `10080`

Routes:

- `POST /auth/login` sets the cookie.
- `POST /auth/logout` deletes the cookie at path `/`.
- `GET /auth/me` reads the cookie.

Authentication errors:

- missing cookie: `401` with `{"detail":"Authentication required."}`
- invalid session: `401` with `{"detail":"Invalid session."}`
- invalid login: `401` with `{"detail":"Invalid credentials."}`
- missing super admin role: `403` with `{"detail":"Super admin role required."}`

## Endpoint Inventory

| Method | Path                                        | Access             | Success |
| ------ | ------------------------------------------- | ------------------ | ------- |
| GET    | `/health`                                   | public             | `200`   |
| POST   | `/auth/login`                               | public             | `200`   |
| POST   | `/auth/logout`                              | public             | `200`   |
| GET    | `/auth/me`                                  | admin cookie       | `200`   |
| GET    | `/messages`                                 | public             | `200`   |
| GET    | `/messages/export`                          | public             | `200`   |
| GET    | `/admin/channels`                           | admin cookie       | `200`   |
| POST   | `/admin/channels`                           | admin cookie       | `201`   |
| DELETE | `/admin/channels/{channel_id}`              | admin cookie       | `200`   |
| GET    | `/admin/users`                              | admin cookie       | `200`   |
| POST   | `/admin/users`                              | super admin cookie | `201`   |
| GET    | `/admin/operations/summary`                 | admin cookie       | `200`   |
| GET    | `/admin/data-management/summary`            | admin cookie       | `200`   |
| PUT    | `/admin/data-management/retention-settings` | admin cookie       | `200`   |
| POST   | `/admin/data-management/cleanup/preview`    | admin cookie       | `200`   |
| POST   | `/admin/data-management/cleanup/confirm`    | admin cookie       | `200`   |
| GET    | `/analytics/overview`                       | public             | `200`   |
| GET    | `/analytics/message-volume`                 | public             | `200`   |
| GET    | `/analytics/top-senders`                    | public             | `200`   |
| GET    | `/analytics/top-channels`                   | public             | `200`   |
| GET    | `/analytics/top-emotes`                     | public             | `200`   |
| GET    | `/users/{slug}/analytics`                   | public             | `200`   |
| GET    | `/channels/{slug}/analytics`                | public             | `200`   |

## Request Bodies

### `POST /auth/login`

```json
{
  "email": "admin@kicklogs.local",
  "password": "admin123"
}
```

Validation:

- `email`: string, min length 3, max length 320
- `password`: string, min length 1, max length 256

### `POST /admin/channels`

```json
{
  "slug": "hype"
}
```

Validation:

- `slug`: string, min length 1, max length 120

Failure:

- unresolved Kick channel: `422` with `{"detail":"Kick channel could not be resolved."}`

### `POST /admin/users`

```json
{
  "email": "moderator@kicklogs.local",
  "password": "strongpass123"
}
```

Validation:

- `email`: string, min length 3, max length 320
- `password`: string, min length 8, max length 256

Failures:

- duplicate email: `409` with `{"detail":"User email already exists."}`
- missing super admin role: `403` with `{"detail":"Super admin role required."}`

### `PUT /admin/data-management/retention-settings`

```json
{
  "message_retention_days": 90,
  "raw_event_retention_days": 30
}
```

Validation:

- `message_retention_days`: `30`, `90`, or `null`
- `raw_event_retention_days`: `30`, `90`, or `null`

### `POST /admin/data-management/cleanup/preview`

```json
{
  "target": "channel",
  "channel_slug": "hype",
  "sender": null
}
```

`target` values:

- `old_messages`
- `old_raw_events`
- `channel`
- `sender`

Target-specific rules:

- `channel` requires non-empty `channel_slug`
- `sender` requires non-empty `sender`
- `old_messages` uses stored `message_retention_days`
- `old_raw_events` uses stored `raw_event_retention_days`

### `POST /admin/data-management/cleanup/confirm`

```json
{
  "target": "channel",
  "channel_slug": "hype",
  "sender": null,
  "confirmation_text": "DELETE CHANNEL hype"
}
```

Validation:

- same fields as cleanup preview
- `confirmation_text`: string, min length 1, max length 240

Failure:

- wrong confirmation: `400`

## Query Parameters

### `GET /messages`

| Parameter    | Type                 | Default | Rule                                                                           |
| ------------ | -------------------- | ------- | ------------------------------------------------------------------------------ | ----------- |
| `sender`     | string or null       | null    | max 160, trimmed, case-insensitive exact match against username/slug snapshots |
| `channel`    | string or null       | null    | max 160, trimmed, case-insensitive contains against channel slug/display name  |
| `q`          | string or null       | null    | max 500, trimmed, case-insensitive contains against message content            |
| `start`      | ISO datetime or null | null    | inclusive `message_created_at >= start`                                        |
| `end`        | ISO datetime or null | null    | inclusive `message_created_at <= end`                                          |
| `reply_only` | boolean              | false   | restricts `message_type == "reply"`                                            |
| `emote_only` | boolean              | false   | restricts rows with at least one emote                                         |
| `cursor`     | string or null       | null    | max 200, format `message_created_at                                            | message_id` |
| `limit`      | integer              | 50      | 1 to 100                                                                       |

Invalid cursor:

```json
{
  "detail": "Invalid cursor."
}
```

Invalid date range:

```json
{
  "detail": "Search start datetime must be before end datetime."
}
```

Cursor behavior:

- split from the right on `|`
- timestamp parsed with `datetime.fromisoformat`
- `Z` is accepted by replacing it with `+00:00`
- naive timestamps are treated as UTC
- message id must be positive
- pagination returns rows where:
  - `message_created_at < cursor.message_created_at`, or
  - same timestamp and `id < cursor.message_id`

Ordering:

- `message_created_at DESC`
- `id DESC`

### `GET /messages/export`

Same filters as `/messages`, except:

| Parameter | Type            | Default | Rule                                        |
| --------- | --------------- | ------- | ------------------------------------------- |
| `format`  | `json` or `csv` | `json`  | query alias is exactly `format`             |
| `limit`   | integer or null | env max | min 1, clamped to `MESSAGE_EXPORT_MAX_ROWS` |

Default `MESSAGE_EXPORT_MAX_ROWS`: `1000`.

CSV response:

- media type: `text/csv; charset=utf-8`
- header: `Content-Disposition: attachment; filename="kick-logs-export.csv"`
- formula-safe values: values starting with `=`, `+`, `-`, or `@` are prefixed with `'`

CSV columns, in order:

1. `message_created_at`
2. `kick_message_id`
3. `channel_slug`
4. `channel_display_name`
5. `sender_username`
6. `sender_slug`
7. `message_type`
8. `content`
9. `emotes`
10. `reply_to_sender`
11. `reply_to_content`
12. `thread_parent_id`

### `GET /analytics/*`

Common query parameters:

| Parameter | Type                 | Default | Rule                                                                         |
| --------- | -------------------- | ------- | ---------------------------------------------------------------------------- |
| `start`   | ISO datetime or null | null    | inclusive                                                                    |
| `end`     | ISO datetime or null | null    | inclusive                                                                    |
| `channel` | string or null       | null    | max 160, exact case-insensitive match against channel slug/display name      |
| `sender`  | string or null       | null    | max 160, exact case-insensitive match against sender username/slug snapshots |
| `limit`   | integer              | 10      | only top-list routes, 1 to 100                                               |
| `bucket`  | `hour` or `day`      | `day`   | only message-volume                                                          |

Invalid date range:

```json
{
  "detail": "Analytics start datetime must be before end datetime."
}
```

### Profile Routes

`GET /users/{slug}/analytics`

- `slug`: path string, min length 1, max length 160
- unknown sender: `404` with `{"detail":"Sender profile not found."}`
- frontend may link display usernames with `_` to profile slugs with `-`

`GET /channels/{slug}/analytics`

- `slug`: path string, min length 1, max length 160
- unknown channel: `404` with `{"detail":"Channel profile not found."}`

## Response Shapes

The canonical frontend response types live in `apps/web/src/types/api.ts`. Representative JSON
fixtures live in `docs/contracts/fixtures/`.

Fixture map:

- `health.json`
- `auth_login.json`
- `admin_users.json`
- `admin_channels.json`
- `message_search.json`
- `message_export.json`
- `analytics_overview.json`
- `message_volume.json`
- `top_senders.json`
- `top_channels.json`
- `top_emotes.json`
- `operations_summary.json`
- `data_management_summary.json`
- `data_cleanup_preview.json`
- `user_profile.json`
- `channel_profile.json`
- `errors.json`

## Message Reply And Emote Contract

Message rows include both a stable snapshot and nested sender/channel objects.

Required reply fields:

- `message_type` is `"reply"` for reply rows
- `reply_metadata` is an object
- reply sender is read from `reply_metadata.original_sender.username`
- reply content is read from `reply_metadata.original_message.content`
- `thread_parent_id` is the parent Kick message id when available

Required emote fields:

- response `emotes` is an array
- each emote has:
  - `id`
  - `name`
  - `token`
  - `image_url`
- export CSV serializes the message emotes list as JSON text

## Admin Data Cleanup Confirmation Text

The confirmation text is generated from the cleanup target:

- `old_messages`: `DELETE OLD MESSAGES`
- `old_raw_events`: `DELETE OLD RAW EVENTS`
- `channel`: `DELETE CHANNEL {channel_slug}`
- `sender`: `DELETE SENDER {sender}`

When retention is `null` for old-message or old-raw-event cleanup, preview returns:

- `can_execute: false`
- `reason: "Retention is set to keep forever."`

## Contract Test Guidance For Go

The Go rewrite should add contract tests that compare real handler responses against these fixtures
where practical.

High-priority strict fixtures:

- `auth_login.json`
- `message_search.json`
- `message_export.json`
- `operations_summary.json`
- `data_management_summary.json`
- `user_profile.json`
- `channel_profile.json`

High-priority behavior checks:

- auth cookie is set/deleted with correct attributes
- public routes do not require auth
- admin routes reject missing or invalid cookies
- super admin-only routes reject normal admin users
- sender search is exact, not contains
- channel/content search behavior is preserved
- cursor pagination is stable
- CSV export column order is unchanged
