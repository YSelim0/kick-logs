# Reverse Proxy, Real Client IP, and Origin Lockdown

This runbook covers exposing Kick Logs safely behind a reverse proxy and giving the API the real
client IP so its rate limiting (issue #20) works and cannot be trivially bypassed. It is generic:
nothing here is tied to a specific domain or host.

## Why this matters

The API resolves the client IP from a proxy header (`RATE_LIMIT_CLIENT_IP_HEADER`, default
`CF-Connecting-IP`) when `RATE_LIMIT_TRUST_PROXY=true`. That header is only trustworthy if a client
**cannot reach the app or the origin while bypassing the trusted proxy**. Two bypass paths must be
closed:

1. **Direct container access.** `docker compose` publishes the api/web/ClickHouse ports. If they bind
   to `0.0.0.0`, anyone can hit `http://<host-ip>:<port>/...` directly, skip the proxy, and forge the
   IP header — every IP-based limit collapses and ClickHouse-backed reads can be hammered.
2. **Direct origin access.** If the host's `443`/`80` are open to the whole internet, a client can hit
   the origin directly (skipping a CDN like Cloudflare) and forge the CDN's client-IP header.

## 1. Bind container ports to loopback (closes path 1)

`compose.yaml` binds published ports to `${API_BIND_HOST:-127.0.0.1}` / `WEB_BIND_HOST` / `CH_BIND_HOST`.
Keep the default `127.0.0.1` when a reverse proxy runs on the same host. Verify after `up`:

```bash
sudo ss -tlnp | grep -E ':3000|:8000|:8123|:9000'   # all should show 127.0.0.1, not 0.0.0.0
curl -sS --max-time 3 http://<host-ip>:8000/health   # must FAIL/refuse from outside
```

If you run without a local reverse proxy and need direct external access, set `API_BIND_HOST=0.0.0.0`
(and front it with a firewall yourself).

## 2. Reverse proxy: forward the real client IP

The proxy must pass the client IP header through to the app. Examples (nginx):

- **Behind a CDN (e.g. Cloudflare).** The CDN sets `CF-Connecting-IP`; nginx forwards request headers
  to the upstream by default, so the app receives it. Optionally make nginx's own logs show the real
  client with the `realip` module:

  ```nginx
  # http{} or server{} — pull current ranges from cloudflare.com/ips-v4 and ips-v6
  set_real_ip_from 173.245.48.0/20;   # ... (all Cloudflare ranges)
  real_ip_header CF-Connecting-IP;
  ```

  Keep `RATE_LIMIT_CLIENT_IP_HEADER=CF-Connecting-IP`.

- **Plain nginx, no CDN.** nginx already sets `X-Real-IP $remote_addr` (the real TCP peer — not
  spoofable once the app port is loopback-bound). Point the app at it:

  ```env
  RATE_LIMIT_CLIENT_IP_HEADER=X-Real-IP
  ```

When the app is **not** behind any trusted proxy, set `RATE_LIMIT_TRUST_PROXY=false` so it uses the
raw `RemoteAddr`.

## 3. Firewall the origin to the CDN (closes path 2)

Only needed when a CDN proxies traffic and you trust its client-IP header. Restrict `80`/`443` to the
CDN's published ranges so nobody can hit the origin directly with a forged header.

> **WARNING — do not lock yourself out.** Keep SSH (`22`) and any other admin access open *before*
> enabling a default-deny policy. Apply and verify over a session you can afford to lose, or use a
> console fallback.

ufw sketch (Cloudflare example; refresh ranges from `cloudflare.com/ips-v4` + `ips-v6`):

```bash
sudo ufw allow 22/tcp                      # keep SSH first!
for cidr in $(curl -s https://www.cloudflare.com/ips-v4) \
            $(curl -s https://www.cloudflare.com/ips-v6); do
  sudo ufw allow proto tcp from "$cidr" to any port 80,443
done
sudo ufw default deny incoming
sudo ufw enable
```

The CDN ranges change occasionally; re-run the loop (or schedule it) and reload the firewall when they
do. The same ranges should feed nginx `set_real_ip_from` in step 2.

## Verify end-to-end

```bash
curl -sS --max-time 3 http://<host-ip>:8000/health    # direct app  -> refused
curl -sI https://<your-domain>/api/health             # via proxy   -> 200
# spam a cheap endpoint quickly via the proxy -> eventually 429 + Retry-After
```
