# Kick Webhook Subscription Tracking

## Summary

Kick Logs will add a generic Kick webhook foundation and first use it to track
subscription events for followed channels. The first product goal is to calculate each followed
channel's **currently active subscriber count** from webhook-delivered subscription periods.

Initial normalized event scope:

- `channel.subscription.new`
- `channel.subscription.renewal`
- `channel.subscription.gifts`

Out of scope for this first implementation:

- KICKs analytics
- moderation events such as ban/timeout
- follow/livestream events
- frontend visualization as the first step

The infrastructure must still be generic enough to add those event families later without
redesigning the webhook receiver, raw inbox, event subscription registry, or processing flow.

## Product Rules

- Active subscriber count means channel-specific unique users whose subscription period has not
  expired.
- Active count is calculated dynamically from stored periods:
  `started_at <= now < expires_at`.
- Do not run a cleanup/cron just to expire users from active counts.
- Use Kick-provided `expires_at` as the source of truth.
- If `expires_at` is missing, fallback to `created_at + 30 days`.
- During the first 30 days after enabling webhooks, serve whatever data has been collected. Do not
  show warning/coverage labels for the initial partial window.
- Gift subscriptions create one normalized subscription period per giftee.
- Anonymous or missing gifter data must not block normalization.
- Disabling a followed channel should delete or deactivate the related Kick event subscriptions so
  unnecessary webhook traffic stops.
- Frontend work comes after the backend/data pipeline is complete.

## Runtime And Networking

- Add a public webhook endpoint: `POST /webhooks/kick`.
- Local development uses a Cloudflare tunnel URL, for example:
  `https://local-cloudflare-domain.com/webhooks/kick`.
- Production uses the real domain, for example:
  `https://kicklogs.net/webhooks/kick`.
- The Kick Developer panel webhook callback URL must point to the active environment's public
  `/webhooks/kick` URL.
- The Kick event subscription API does not receive the callback URL in the request body; it uses the
  app's configured webhook callback.
- Cloudflare challenge/access/bot protections must not apply to `/webhooks/kick`.
- Existing `/search` Cloudflare protection is acceptable as long as it is path-scoped and does not
  affect `/webhooks/kick`.
- App rate limiting should not treat webhook requests like user traffic. Webhook security is
  signature verification plus idempotency, not aggressive IP throttling.

## Configuration

Add env/config fields:

- `KICK_CLIENT_ID`
- `KICK_CLIENT_SECRET`
- `KICK_API_BASE_URL=https://api.kick.com`
- `KICK_OAUTH_TOKEN_URL=https://id.kick.com/oauth/token`
- `KICK_WEBHOOK_PUBLIC_KEY`
- `KICK_WEBHOOK_SYNC_ENABLED=true`
- `KICK_WEBHOOK_EVENTS=channel.subscription.new,channel.subscription.renewal,channel.subscription.gifts`
- `KICK_WEBHOOK_PROCESS_BATCH_SIZE=50`
- `KICK_WEBHOOK_PROCESS_MAX_ATTEMPTS=5`

Behavior:

- If Kick client credentials are missing:
  - API still starts.
  - Event subscription sync is disabled.
  - Chat logging and the rest of the product continue to work.
  - Log a clear warning.
- If the webhook public key is missing:
  - API still starts.
  - `POST /webhooks/kick` fails closed because signatures cannot be verified.
  - Log a clear warning.
- Add all new env fields to `.env.example` and Compose API env wiring.

## Data Model

SQLite remains the control-plane store:

- Kick app token cache metadata if needed.
- Kick event subscription registry.
- Webhook inbox status and retry state.
- Sync state per followed channel/event type.

ClickHouse remains the data-plane analytics store:

- Normalized subscription periods.
- Optional raw valid webhook archive if needed for audit/debug beyond the SQLite inbox.

Required logical entities:

### `followed_channels`

Add an explicit broadcaster identity field:

- `broadcaster_user_id`

Important rule:

- Do not assume existing `followed_channels.kick_channel_id` is the same as Kick
  `broadcaster_user_id`.
- Resolve and store `broadcaster_user_id` through the official Kick channel API before creating
  webhook event subscriptions.

### `kick_webhook_events`

SQLite inbox table:

- `message_id` from `Kick-Event-Message-Id` as the primary idempotency key
- `subscription_id`
- `event_type`
- `event_version`
- `raw_payload_json`
- `status`: `pending`, `processed`, `failed`, `ignored`
- `attempts`
- `received_at`
- `processed_at`
- `error_message`

### `kick_event_subscriptions`

SQLite registry table:

- followed channel id
- broadcaster user id
- event type
- event version
- method `webhook`
- Kick subscription id
- status
- latest sync error
- created/updated/synced timestamps

### `channel_subscription_periods`

ClickHouse normalized analytics table:

- deterministic id
- event message id
- event type
- followed channel id
- broadcaster user id
- channel slug/display-name snapshot
- subscriber/giftee Kick user id
- subscriber username/slug/profile snapshot when available
- gifter Kick user id and username/slug/profile snapshot when available
- `is_gift`
- `started_at`
- `expires_at`
- raw payload JSON
- ingested timestamp

Idempotency:

- `Kick-Event-Message-Id` is unique in the inbox.
- Normalized rows must also be idempotent.
- For gift events with multiple giftees, use a deterministic unique key such as:
  `message_id + giftee_user_id`.

## Kick API Integration

Add a Kick API client for official endpoints.

Responsibilities:

- Obtain and cache app access tokens using env client credentials.
- Resolve broadcaster user id for followed channels through official channel lookup.
- List existing event subscriptions.
- Create event subscriptions via `POST /public/v1/events/subscriptions`.
- Delete event subscriptions when a followed channel is disabled.

Event subscription sync:

- On API startup:
  - Load enabled followed channels.
  - Resolve missing broadcaster user ids.
  - Ensure required subscription events exist.
  - Store created/existing subscription ids.
- On admin channel add:
  - Resolve channel metadata.
  - Upsert followed channel.
  - Ensure subscription events for that channel.
- On admin channel disable:
  - Disable the local followed channel.
  - Delete known Kick event subscriptions for that channel.
  - Mark local subscription records deleted/disabled.
- Add an authenticated manual sync endpoint for recovery/debugging even though startup and channel
  changes sync automatically.

If Kick API fails during sync:

- Do not fail the whole API.
- Store latest sync error.
- Surface sync health through admin API.
- Retry on next startup, channel update, or explicit manual sync.

## Webhook Receiver

Receiver flow:

1. Accept only `POST /webhooks/kick`.
2. Read the raw request body once.
3. Validate required Kick headers:
   - `Kick-Event-Message-Id`
   - `Kick-Event-Message-Timestamp`
   - `Kick-Event-Type`
   - `Kick-Event-Version`
   - `Kick-Event-Signature`
4. Verify the signature against the raw body and timestamp using `KICK_WEBHOOK_PUBLIC_KEY`.
5. Insert the webhook inbox row idempotently.
6. Return 2xx quickly for duplicates and successfully stored events.
7. Return non-2xx for invalid signature, missing required headers, or malformed required envelope.

Security rules:

- Do not rely on source IP allowlisting for Kick webhook authenticity.
- Do not require admin auth on the webhook endpoint.
- Do not apply the normal user-facing rate limit policies to the webhook endpoint.
- Signature verification is fail-closed.

## Webhook Processor

Processor flow:

1. Pick pending webhook inbox rows.
2. Parse event by type.
3. For supported sub events, normalize into subscription periods.
4. Write normalized periods to ClickHouse.
5. Mark the inbox row processed.
6. On parse/DB failure, increment attempts and keep latest error.
7. Unsupported but valid events are marked ignored/raw-stored, not failed.

Normalization rules:

- `channel.subscription.new` creates one non-gift period.
- `channel.subscription.renewal` creates one non-gift renewal period.
- `channel.subscription.gifts` creates one gift period per giftee.
- Store subscriber/giftee snapshots from payload.
- Store gifter snapshot only when present.
- Store broadcaster/channel snapshot from payload and map to a followed channel by
  `broadcaster_user_id`.
- Unknown/unfollowed broadcaster events should not crash processing. Mark them ignored with a clear
  reason.

Runtime placement:

- Keep webhook processing in the API process for the first implementation unless the processing path
  proves heavy.
- Use a bounded background worker started by `cmd/api`.
- Keep processing idempotent so a future dedicated worker service can be split out cleanly.

## Public And Admin API

Backend APIs must be ready before frontend work starts.

Public API:

- Add a channel subscription summary endpoint for channel profile usage.
- Response includes at minimum:
  - channel slug
  - active subscriber count
  - active gifted subscriber count
  - latest subscription event timestamp if available

Admin API:

- Add webhook/subscription health endpoint.
- Include:
  - configured event types
  - missing credentials/config flags
  - followed channels with subscription sync status
  - latest webhook received timestamp
  - pending/failed/ignored/processed webhook inbox counts
  - latest sync error per channel/event type

Admin action:

- Add manual webhook subscription sync endpoint for recovery/debugging.

Frontend:

- Defer visual implementation until backend is complete.
- Later UI surfaces:
  - public channel profile active subscriber metric
  - admin channel list/sync health
  - admin operations webhook health panel

## Implementation Phases

### Phase 1 - Plan And Config Docs

- Replace the stale active implementation plan with this webhook subscription plan.
- Add GitHub issue from the same plan text.
- Do not change runtime code in this phase.

### Phase 2 - Storage And Domain Foundation

- Add domain models and port interfaces for:
  - webhook inbox
  - event subscription registry
  - subscription periods
  - Kick event subscription API client
- Add SQLite migrations and repositories for:
  - `followed_channels.broadcaster_user_id`
  - `kick_webhook_events`
  - `kick_event_subscriptions`
- Add ClickHouse migration and repository for:
  - `channel_subscription_periods`
- Add repository tests for idempotent inserts, pending/failure transitions, subscription registry
  upserts, and active summary query.
- Commit as one backend storage feature.

### Phase 3 - Kick API Client And Subscription Sync

- Add config/env loading for Kick client credentials, token URL, API base URL, webhook event list,
  and sync enable flag.
- Implement app access token acquisition/cache.
- Implement official Kick channel lookup to resolve `broadcaster_user_id`.
- Implement list/create/delete event subscription methods.
- Add sync service:
  - startup sync
  - channel add sync
  - channel disable cleanup
  - manual admin sync
- Sync failures must be stored and visible; they must not take the API down.
- Add unit tests with fake Kick API client.
- Commit as one Kick subscription sync feature.

### Phase 4 - Webhook Receiver

- Add signature verifier.
- Add `POST /webhooks/kick`.
- Insert valid webhook events into the SQLite inbox idempotently.
- Return 2xx for duplicate message ids.
- Fail closed on invalid/missing signature config or invalid signatures.
- Exclude webhook endpoint from normal user-facing rate-limit policies.
- Add route/middleware tests for valid, duplicate, missing header, invalid signature, and malformed
  body cases.
- Commit as one webhook receiver feature.

### Phase 5 - Webhook Processor And Normalization

- Add parser/normalizer for supported sub events.
- Normalize:
  - `channel.subscription.new`
  - `channel.subscription.renewal`
  - `channel.subscription.gifts`
- Create one period per giftee for gift events.
- Use `expires_at` first, fallback to `created_at + 30 days`.
- Ignore valid but unsupported/unfollowed events with a clear reason.
- Start bounded background processing in API.
- Add tests for normalization and retry/failure behavior.
- Commit as one processor feature.

### Phase 6 - Backend Query APIs

- Add public channel subscription summary endpoint.
- Add admin webhook/subscription health endpoint.
- Add admin manual sync endpoint.
- Add tests for public summary, admin auth, health response shape, and manual sync.
- Commit as one API feature.

### Phase 7 - Docs And Smoke

- Update project docs/context for the completed backend webhook pipeline.
- Add operation notes for:
  - local Cloudflare tunnel callback
  - production callback URL
  - Cloudflare challenge bypass for `/webhooks/kick`
  - Kick callback URL is configured in the Kick Developer panel
- Run relevant backend checks:
  - `go test ./...`
  - `go vet ./...`
  - targeted ClickHouse integration test if Docker is available
  - `pnpm format:check`
- Commit docs/smoke updates.

### Phase 8 - Frontend Visualization

- Start only after backend is complete and verified.
- Add public channel profile active subscriber metric.
- Add admin webhook/subscription health surfaces.
- Follow `docs/design/design.md`.
- Commit frontend UI as separate feature-sized commits.

## Test Plan

Unit tests:

- Signature verification accepts valid raw body/signature.
- Signature verification rejects tampered body.
- Missing required webhook headers returns non-2xx.
- Duplicate message id is idempotent.
- New subscription event normalizes one period.
- Renewal event normalizes one period.
- Gift subscription event normalizes one period per giftee.
- Anonymous/missing gifter does not fail.
- Missing `expires_at` falls back to `created_at + 30 days`.
- Active count excludes expired periods.
- Active count dedupes overlapping periods for the same user/channel.
- Unsupported valid events are ignored, not failed.

Integration tests:

- Webhook receiver stores raw inbox row and returns quickly.
- Processor writes normalized ClickHouse rows and marks inbox processed.
- Processor retry path preserves raw payload and latest error.
- Channel add triggers subscription ensure when config is present.
- Channel disable triggers subscription deletion.
- Startup sync reconciles existing followed channels.
- Public summary endpoint reflects current active counts.
- Admin health endpoint reports inbox counts and subscription sync state.

Manual checks:

- Start app locally.
- Expose `/webhooks/kick` through Cloudflare tunnel.
- Configure Kick Developer panel callback URL to the tunnel URL.
- Add followed channel.
- Verify Kick event subscriptions are created.
- Trigger or receive real subscription/gift-sub events.
- Confirm normalized periods and active count update.

## Operations Notes

Local development:

```text
Kick -> https://local-cloudflare-domain.com/webhooks/kick -> cloudflared tunnel -> local API
```

Production:

```text
Kick -> https://kicklogs.net/webhooks/kick -> Cloudflare/proxy -> VPS API
```

Production requirements:

- `/webhooks/kick` must bypass Cloudflare challenge/access/bot protections.
- The origin can remain Cloudflare-only.
- Do not open direct VPS firewall rules for unknown Kick IPs.
- The Kick Developer panel callback URL must be changed to whichever environment is being tested.

Important limitation:

- Kick webhooks only provide events after event subscriptions are created.
- The first 30 days after enabling the feature may be incomplete, but the application will still
  serve the collected data without a warning label by product decision.
