# Kick Webhook Operations

## Overview

`POST /webhooks/kick` receives Kick subscription events. Signature verification uses Ed25519
with the public key from `KICK_WEBHOOK_PUBLIC_KEY`. Missing or invalid key fails closed (503/401).

## Kick Developer Panel Setup

1. Go to the Kick Developer panel.
2. Under your app's webhook settings, copy the **Public Key** — set it as `KICK_WEBHOOK_PUBLIC_KEY`.
3. Set the **Callback URL** to the active environment's public webhook URL:
   - Local: `https://<your-cloudflare-tunnel>.trycloudflare.com/webhooks/kick`
   - Production: `https://kicklogs.net/webhooks/kick`
4. The callback URL is global to the app — only one environment can receive live events at a time.

## Local Development (Cloudflare Tunnel)

```text
Kick -> https://<tunnel>.trycloudflare.com/webhooks/kick -> cloudflared -> localhost:8000
```

Start a tunnel:

```powershell
cloudflared tunnel --url http://localhost:8000
```

Copy the printed `trycloudflare.com` URL and paste it as the callback URL in the Kick Developer
panel. Update `KICK_WEBHOOK_PUBLIC_KEY` with the public key from the same panel.

## Production

```text
Kick -> https://kicklogs.net/webhooks/kick -> Cloudflare -> VPS:8000
```

Production requirements:

- `/webhooks/kick` must bypass Cloudflare challenge, access, and bot-fight protections.
  Configure a Cloudflare Page Rule or WAF rule to skip those checks for this path.
- The origin (VPS) must remain Cloudflare-only via firewall — do not open port 80/443 to
  unknown IPs. The Cloudflare-to-origin path is already authenticated by the signed payload.
- Do not add IP allowlisting for Kick source IPs. The Ed25519 signature is the only valid
  authenticity signal.

## Cloudflare Bypass for `/webhooks/kick`

In the Cloudflare dashboard:

- **Security > WAF**: add a rule matching `http.request.uri.path eq "/webhooks/kick"` with
  action **Skip** → check all managed rules and rate limiting.
- **Page Rules** (legacy): `kicklogs.net/webhooks/kick` → Security Level: Essentially Off,
  Disable Performance features if they buffer the body.

## Rate Limiting

`POST /webhooks/kick` has no application-level rate-limit policy. The endpoint's security model
is: Ed25519 signature verification + idempotent inbox insert. Aggressive IP throttling would
interfere with Kick's delivery retries and is not appropriate for server-to-server webhook traffic.

## Subscription Sync

On API startup, `SyncAll` runs in the background if `KICK_CLIENT_ID` and `KICK_CLIENT_SECRET`
are configured. It resolves missing `broadcaster_user_id` values and creates any missing Kick
event subscriptions.

Sync is also triggered automatically when a followed channel is added or disabled via the admin
panel. A manual sync can be triggered via:

```http
POST /admin/webhooks/sync
Cookie: kick_logs_session=<session>
```

## Health Check

```http
GET /admin/webhooks/health
Cookie: kick_logs_session=<session>
```

Returns:

- `inbox_counts` — pending/processed/failed/ignored counts
- `latest_webhook_received_at` — last received event timestamp
- `channels[].subscriptions[].status` — per-channel, per-event-type sync status
- `missing_client_credentials` / `missing_webhook_public_key` — config warnings

## Partial Data Window

Kick webhooks only deliver events after event subscriptions are created. The first 30 days after
enabling the feature may show incomplete subscriber counts. The application serves whatever data
has been collected without a warning label (product decision, see implementation plan).
