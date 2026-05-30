# Rate Limiting for Public and Admin API Endpoints (Issue #20)

## Summary

Add GCRA-based rate limiting to all public and admin API endpoints. The limiter uses
`github.com/throttled/throttled/v2` with an in-memory LRU store, sits behind a `ports.RateLimiter`
interface so a future Redis backend requires no middleware or policy changes, and resolves real client
IPs through the Cloudflare `CF-Connecting-IP` header under configurable trusted-proxy mode.

## Why

- `POST /auth/login` has no brute-force protection.
- Public ClickHouse-backed reads (`/messages`, `/messages/export`, `/analytics/*`, profile analytics)
  can be hammered to drive expensive queries on the memory-constrained 4 GB VPS.
- Admin write endpoints are auth-gated but unthrottled.
- Behind Cloudflare + nginx, the API sees the proxy IP as `RemoteAddr`; without real-client-IP
  resolution any IP-based limit collapses to a single key or can be spoofed.

## Design Decisions

- **Algorithm:** GCRA (token-bucket semantics) via `throttled/throttled/v2`. GCRA gives a sustained
  `MaxRate` plus a separate `MaxBurst` and has no fixed-window boundary spike. A plain fixed-window
  counter was rejected: it has no separate burst control and double-counts across the window edge.
- **Port-based architecture:** `RateLimiter` interface in `ports/`, implementation in
  `infra/ratelimit/`. Middleware depends only on the interface.
- **In-memory store now, Redis later:** `throttled` ships both `memstore` (in-memory LRU) and Redis
  stores behind the same store interface. Start with `memstore`; a future horizontal-scale move
  swaps the store via config only. Redis is intentionally NOT added now (single instance; avoids new
  memory/failure pressure on the 4 GB box, see #15).
- **Real client IP:** `CF-Connecting-IP` header when `RATE_LIMIT_TRUST_PROXY=true`, else
  `RemoteAddr`. Trusted only because the origin is locked to Cloudflare IP ranges (infra
  prerequisite).
- **Global middleware + path match:** Single middleware with an ordered policy table. Route files are
  not modified for rate limiting. The first matching policy wins.
- **Login dual-key:** Middleware applies IP-only check. Login handler applies IP+email check after
  body parse. Two separate rate limit calls — both must pass.
- **Admin keying:** Middleware extracts user ID from JWT cookie via `TokenService.GetUserID()` (no DB
  hit, just token parse). Falls back to IP if cookie is missing or token is invalid.
- **Prediction excluded:** Moved client-side per #19. No longer hits the Go API.
- **Memory budget:** `RATE_LIMIT_STORE_MAX_KEYS=65536` default. Each GCRA entry is ~150 bytes.
  Total: ~10 MB, well within the 384 MB api container budget from #15.

## Policy Table

| Endpoint | Key | MaxRate (sustained) | MaxBurst |
| --- | --- | --- | --- |
| `/health` | — | unlimited | — |
| `POST /auth/login` | IP | 20 / 10 min | 5 |
| `POST /auth/login` | IP + email (handler) | 8 / 10 min | 3 |
| `GET /messages` | IP | 20 / min | 10 |
| `GET /messages/export` | IP | 3 / min | 2 |
| `GET /analytics/*`, profile analytics | IP | 60 / min | 15 |
| `GET /admin/*` | user ID (IP fallback) | 120 / min | 30 |
| `POST/PUT/DELETE /admin/*` | user ID (IP fallback) | 30 / min | 10 |
| `POST /admin/data-management/cleanup/confirm` | user ID | 3 / min | 1 |

Numbers are a starting point and may be tuned after observing real traffic.

## Prerequisites (infra, not code — outside this plan's scope)

- VPS firewall restricts origin 80/443 to Cloudflare IP ranges.
- nginx forwards `CF-Connecting-IP` to the API via `real_ip` module or header passthrough.

---

## Phase 1 — Port Interface and GCRA Infra Adapter

### Goal

Define the `RateLimiter` port interface, implement it with `throttled/v2` GCRA + `memstore`, and
verify with unit tests. No middleware, no wiring yet.

### Tasks

- [ ] Add `github.com/throttled/throttled/v2` dependency to `apps/api-go/go.mod` via
      `go get github.com/throttled/throttled/v2@latest` and `go mod tidy`.
- [ ] Create `apps/api-go/internal/ports/ratelimit.go`:
  - [ ] Define `RateLimitResult` struct with `Limited bool` and `RetryAfter int` (seconds) fields.
  - [ ] Define `RateLimiter` interface with a single method:
        `RateLimit(key string, perPeriod int, periodSeconds int, maxBurst int) (RateLimitResult, error)`.
  - [ ] Design note: rate params are per-call so one limiter instance with one `memstore` serves all
        policies. The infra adapter lazily creates GCRA limiters per unique config tuple.
- [ ] Create `apps/api-go/internal/infra/ratelimit/gcra.go`:
  - [ ] Define `GCRARateLimiter` struct holding a `*memstore.MemStore` and a `sync.Map` that caches
        `*throttled.GCRARateLimiterCtx` instances keyed by a `policyKey` struct
        `{perPeriod, periodSeconds, maxBurst}`.
  - [ ] Implement `NewGCRA(maxKeys int) *GCRARateLimiter` constructor that creates the `memstore`
        with the given LRU capacity.
  - [ ] Implement `RateLimit(key, perPeriod, periodSeconds, maxBurst)` method:
    - Build `policyKey` from the three rate params.
    - Look up or lazily create + store a `throttled.GCRARateLimiterCtx` in `sync.Map` for that key.
    - Create the GCRA rate quota: `throttled.RateQuota{MaxRate: throttled.PerDuration(perPeriod, period), MaxBurst: maxBurst}`.
    - Call `limiter.RateLimitCtx(ctx, key, 1)` and map the result to `ports.RateLimitResult`.
    - Map `throttled`'s `RetryAfter` duration to integer seconds (round up).
  - [ ] Verify that different `policyKey` tuples produce independent GCRA limiter instances but
        share the same underlying `memstore`.
- [ ] Create `apps/api-go/internal/infra/ratelimit/gcra_test.go`:
  - [ ] `TestGCRA_AllowsBurst`: send `maxBurst+1` requests with the same key. All must return
        `Limited: false`.
  - [ ] `TestGCRA_BlocksAfterBurst`: send `maxBurst+2` requests. The last must return
        `Limited: true` with `RetryAfter > 0`.
  - [ ] `TestGCRA_IndependentKeys`: two different keys with same rate config must have independent
        limits. Exhaust key A, key B must still be allowed.
  - [ ] `TestGCRA_IndependentConfigs`: same key string but different rate params must be tracked
        independently (different GCRA limiter instances).
- [ ] Run `go vet ./...` and `go test ./...` in `apps/api-go` — all green.

### Commit

`feat(api): add rate limiter port and GCRA infra adapter`

---

## Phase 2 — Configuration

### Goal

Add rate limiting env-configurable settings to the application config. No middleware yet — just
config loading and validation.

### Tasks

- [ ] Modify `apps/api-go/internal/config/config.go`:
  - [ ] Add three fields to `Config` struct:
    - `RateLimitEnabled bool` — master switch, default `true`.
    - `RateLimitStoreMaxKeys int` — LRU capacity for `memstore`, default `65536`. Controls memory
      budget: 65536 keys × ~150 bytes = ~10 MB.
    - `RateLimitTrustProxy bool` — whether to read `CF-Connecting-IP` for real client IP, default
      `true`.
  - [ ] Add env var parsing in `Load()`:
    - `RATE_LIMIT_ENABLED` → `envBool("RATE_LIMIT_ENABLED", true)`
    - `RATE_LIMIT_STORE_MAX_KEYS` → `envInt("RATE_LIMIT_STORE_MAX_KEYS", 65536)`
    - `RATE_LIMIT_TRUST_PROXY` → `envBool("RATE_LIMIT_TRUST_PROXY", true)`
  - [ ] Wire the three parsed values into the `Config` return struct.
- [ ] Verify existing config tests still pass: `go test ./internal/config/...`.
- [ ] Run `go vet ./...` in `apps/api-go` — green.

### Commit

`feat(api): add rate limit configuration`

---

## Phase 3 — Client IP Resolver and Rate Limit Middleware

### Goal

Implement the client IP resolver and the rate limit middleware with the full policy table. This is
the core middleware that matches requests to policies, extracts keys, checks the limiter, and
returns 429 with `Retry-After` when limited.

### Tasks

#### Client IP Resolver

- [ ] Create or add to `apps/api-go/internal/http/middleware/ratelimit.go`:
  - [ ] Implement `ClientIP(r *http.Request, trustProxy bool) string`:
    - If `trustProxy` is true, read `CF-Connecting-IP` header first. If non-empty, return it
      (trimmed).
    - Otherwise, extract IP from `r.RemoteAddr` (strip port using `net.SplitHostPort`; if that
      fails, return `RemoteAddr` as-is).
    - This function must be exported because the login handler in `routes/auth.go` also calls it.

#### Policy Table

- [ ] Define types in `apps/api-go/internal/http/middleware/ratelimit.go`:
  - [ ] `RateLimitPolicy` struct:
    ```go
    type RateLimitPolicy struct {
        Name          string
        Match         func(method, path string) bool
        Key           func(r *http.Request, ts ports.TokenService, cfg config.Config) string
        PerPeriod     int
        PeriodSeconds int
        MaxBurst      int
    }
    ```
  - [ ] Helper match functions (unexported):
    - `exactMatch(method, path string) func(string, string) bool` — matches exact method + exact
      path.
    - `prefixMatch(method, prefix string) func(string, string) bool` — matches exact method + path
      prefix.
  - [ ] Key extraction functions (built inside `DefaultPolicies`):
    - `ip` key: `"ip:" + ClientIP(r, trustProxy)`.
    - `adminKey`: try `r.Cookie(cfg.JWTCookieName)` → `ts.GetUserID(cookie.Value)` →
      `"admin:uid:<id>"`. If any step fails, fall back to `"admin:ip:" + ClientIP(r, trustProxy)`.
      No DB hit — `GetUserID` is a pure JWT parse.
- [ ] Implement `DefaultPolicies(trustProxy bool) []RateLimitPolicy` returning the ordered policy
      slice. **Order matters — most specific first:**
  1. `cleanup-confirm`: `POST /admin/data-management/cleanup/confirm` → `adminKey`, 3/60s burst 1.
  2. `login-ip`: `POST /auth/login` → `ip`, 20/600s burst 5.
  3. `messages-export`: `GET /messages/export` → `ip`, 3/60s burst 2.
  4. `messages`: `GET /messages` → `ip`, 20/60s burst 10.
  5. `analytics`: `GET /analytics/*` → `ip`, 60/60s burst 15. Uses `prefixMatch("GET", "/analytics/")`.
  6. `profile-analytics`: `GET /users/*/analytics` and `GET /channels/*/analytics` → `ip`, 60/60s
     burst 15. Match: method GET, path has prefix `/users/` or `/channels/` and suffix `/analytics`.
  7. `admin-write`: `POST|PUT|DELETE /admin/*` → `adminKey`, 30/60s burst 10. Match:
     `method ∈ {POST, PUT, DELETE}` and `strings.HasPrefix(path, "/admin/")`.
  8. `admin-read`: `GET /admin/*` → `adminKey`, 120/60s burst 30.
  - [ ] `/health` has no matching policy → always passes through (implicit unlimited).
  - [ ] `OPTIONS` requests pass through because CORS middleware returns 204 before reaching rate
        limit middleware.

#### Middleware Function

- [ ] Implement `RateLimit(limiter ports.RateLimiter, policies []RateLimitPolicy, ts ports.TokenService, cfg config.Config, logger *slog.Logger) func(http.Handler) http.Handler`:
  - [ ] Return a standard middleware closure matching the existing pattern
        (`func(http.Handler) http.Handler`).
  - [ ] On each request:
    1. Iterate `policies`, call `policy.Match(r.Method, r.URL.Path)`.
    2. If no policy matches → call `next.ServeHTTP(w, r)` and return.
    3. Extract key via `policy.Key(r, ts, cfg)`.
    4. Call `limiter.RateLimit(key, policy.PerPeriod, policy.PeriodSeconds, policy.MaxBurst)`.
    5. If error → log warning, pass through (fail-open — a limiter error should not block requests).
    6. If `result.Limited` → set `Retry-After` header (integer seconds), write 429 JSON response
       `{"detail":"Too many requests."}`, return.
    7. If not limited → call `next.ServeHTTP(w, r)`.
  - [ ] 429 JSON response format must match existing `errorResponse` struct: `{"detail":"..."}`.
        Write it directly in middleware (don't import `routes` package to avoid circular dependency).

#### Tests

- [ ] Create `apps/api-go/internal/http/middleware/ratelimit_test.go`:
  - [ ] `TestClientIP_CFConnectingIP`: `trustProxy=true`, `CF-Connecting-IP` header set → returns
        header value.
  - [ ] `TestClientIP_RemoteAddr`: `trustProxy=false`, `CF-Connecting-IP` header set → ignores
        header, returns `RemoteAddr` (port stripped).
  - [ ] `TestClientIP_FallbackToRemoteAddr`: `trustProxy=true`, no `CF-Connecting-IP` header →
        falls back to `RemoteAddr`.
  - [ ] `TestClientIP_RemoteAddrPortStrip`: verify port is stripped from `192.168.1.1:12345`.
  - [ ] `TestRateLimit_Returns429WithRetryAfter`: create limiter with burst 0, send 2 requests.
        Second returns 429 with `Retry-After` header containing a positive integer and JSON body
        `{"detail":"Too many requests."}`.
  - [ ] `TestRateLimit_BurstTolerance`: create limiter with burst 2. First 3 requests (burst+1)
        pass, 4th returns 429.
  - [ ] `TestRateLimit_HealthUnlimited`: send 100 requests to `/health`. All pass (no matching
        policy).
  - [ ] `TestRateLimit_AdminUserIDKeying`: mock `TokenService`, set JWT cookie on request. Two
        different user IDs get independent limits. Exhaust user 1's limit, user 2 still passes.
  - [ ] `TestRateLimit_AdminIPFallback`: no cookie → admin requests keyed by IP. Verify the key
        format starts with `admin:ip:`.
  - [ ] `TestRateLimit_CleanupConfirmTighterThanAdminWrite`: verify `POST /admin/data-management/cleanup/confirm`
        matches `cleanup-confirm` policy (burst 1) not generic `admin-write` (burst 10). Exhaust
        limit with 2 requests; a generic `POST /admin/channels` with same user should still pass.
  - [ ] `TestRateLimit_FailOpen`: pass a limiter that always returns an error. Request should pass
        through (not blocked).
  - [ ] Use `httptest.NewRecorder` and `httptest.NewRequest` pattern consistent with existing tests.
  - [ ] Mock `TokenService` with a simple struct implementing `GetUserID` returning a configurable
        `(int64, bool)`.
- [ ] Run `go vet ./...` and `go test ./...` — all green.

### Commit

`feat(api): add rate limit middleware with client IP resolver`

---

## Phase 4 — Wiring and Login Email-Based Check

### Goal

Wire the limiter into the application: create the `GCRARateLimiter` in `main.go`, pass it through
`Dependencies`, insert the middleware into the handler chain, and add the login email-based rate
limit check in the login handler.

### Tasks

#### Dependencies Struct

- [ ] Modify `apps/api-go/internal/http/routes/dependencies.go`:
  - [ ] Add `RateLimiter ports.RateLimiter` field to `Dependencies`.
  - [ ] Add `TokenService ports.TokenService` field to `Dependencies`.
  - [ ] Add import for `ports` package (it may already be imported — check before adding).

#### Server Wiring

- [ ] Modify `apps/api-go/internal/http/server.go`:
  - [ ] Import `ports` package.
  - [ ] In `NewRouter`, extract `tokenService` from deps when available.
  - [ ] Insert rate limit middleware into the chain. Target position:
    ```
    CORS → RequestLogger → Recover → RateLimit → mux
    ```
    Rate limit runs after Recover (so panics in rate limit code are caught) and before mux (so
    limited requests never reach handlers).
  - [ ] Only apply when `cfg.RateLimitEnabled` is true and `deps.RateLimiter` is not nil:
    ```go
    if cfg.RateLimitEnabled && len(dependencySets) > 0 && dependencySets[0].RateLimiter != nil {
        handler = middleware.RateLimit(
            dependencySets[0].RateLimiter,
            middleware.DefaultPolicies(cfg.RateLimitTrustProxy),
            dependencySets[0].TokenService,
            cfg,
            logger,
        )(handler)
    }
    ```
  - [ ] When deps are absent or rate limiting is disabled, the chain is unchanged.

#### Login Email-Based Check

- [ ] Modify `apps/api-go/internal/http/routes/auth.go`:
  - [ ] Add import for `middleware` package and `strconv`.
  - [ ] In `login()` function, after `decodeJSON` succeeds and before calling `deps.Auth.Login()`,
        add the email-based rate limit check:
    ```go
    if deps.RateLimiter != nil && payload.Email != "" {
        clientIP := middleware.ClientIP(request, deps.Config.RateLimitTrustProxy)
        email := strings.ToLower(strings.TrimSpace(payload.Email))
        key := "login:email:" + clientIP + ":" + email
        result, _ := deps.RateLimiter.RateLimit(key, 8, 600, 3)
        if result.Limited {
            response.Header().Set("Retry-After", strconv.Itoa(result.RetryAfter))
            writeError(response, http.StatusTooManyRequests, "Too many requests.")
            return
        }
    }
    ```
  - [ ] This runs AFTER the middleware's IP-only login check. Both must pass. The middleware blocks
        IP-level brute force (20/10min); the handler blocks per-email+IP brute force (8/10min).
  - [ ] An attacker cannot lock out a victim by email alone because the key includes the attacker's
        IP. Different IPs get independent email limits.

#### Main Bootstrap

- [ ] Modify `apps/api-go/cmd/api/main.go`:
  - [ ] Add import for `ratelimitinfra "github.com/YSelim0/kick-logs/apps/api-go/internal/infra/ratelimit"`.
  - [ ] Add import for `"github.com/YSelim0/kick-logs/apps/api-go/internal/ports"`.
  - [ ] Capture `tokenService` in a variable before passing to `authusecase.NewService`:
    ```go
    tokenService := authinfra.NewJWTTokenService(cfg)
    authService := authusecase.NewService(
        adminRepo,
        authinfra.NewBcryptPasswordHasher(),
        tokenService,
    )
    ```
  - [ ] Create rate limiter after auth service:
    ```go
    var rateLimiter ports.RateLimiter
    if cfg.RateLimitEnabled {
        rateLimiter = ratelimitinfra.NewGCRA(cfg.RateLimitStoreMaxKeys)
        logger.Info("rate limiter enabled", "max_keys", cfg.RateLimitStoreMaxKeys, "trust_proxy", cfg.RateLimitTrustProxy)
    }
    ```
  - [ ] Pass both to `routes.Dependencies`:
    ```go
    server := app.NewAPIServer(cfg, logger, routes.Dependencies{
        // ... existing fields ...
        RateLimiter:  rateLimiter,
        TokenService: tokenService,
    })
    ```

#### Existing Test Fixes

- [ ] Modify `apps/api-go/internal/http/admin_routes_test.go`:
  - [ ] In `newAdminTestRouter`, capture `tokenService` from `authinfra.NewJWTTokenService(cfg)`
        before passing to `authusecase.NewService`.
  - [ ] Add `TokenService: tokenService` to `routes.Dependencies` in the test helper.
  - [ ] `RateLimiter` stays `nil` in existing tests → rate limiting middleware is skipped, all
        existing tests pass without modification.
- [ ] Check all other test files that create `routes.Dependencies` (search for
      `routes.Dependencies{`) and add the new fields if needed.

#### Integration Tests

- [ ] Add rate limit integration test (in `apps/api-go/internal/http/ratelimit_routes_test.go` or
      in existing test files):
  - [ ] `TestLoginRateLimit_IPBlocked`: create router with real `GCRARateLimiter` (small burst,
        e.g., maxBurst=1 for IP, maxBurst=0 for email). Send `maxBurst+2` login requests from same
        IP. Verify the last returns 429 with `Retry-After` header.
  - [ ] `TestLoginRateLimit_EmailKeyIndependent`: exhaust email-based limit for
        `IP1+email@test.com`. Same IP with `other@test.com` must still be allowed.
  - [ ] `TestLoginRateLimit_DifferentIPSameEmail`: exhaust email-based limit for
        `IP1+email@test.com`. `IP2+email@test.com` (different IP) must still be allowed (attacker
        cannot lock victim by email).
  - [ ] `TestAdminRateLimit_UserIDKeyed`: login as admin (get session cookie), send admin GET
        requests until limit. Login as different admin → independent limit.
  - [ ] `TestHealthEndpoint_NeverLimited`: send 50 requests to `/health`, all return 200.
- [ ] Run `go vet ./...` and `go test ./...` — all green.

### Commit

`feat(api): wire rate limiter and add login email-based check`

---

## Phase 5 — Documentation

### Goal

Document the rate limiting design, policy, configuration, and infrastructure prerequisites.

### Tasks

- [ ] Modify `docs/context/decisions.md`:
  - [ ] Add a `## 2026-05-31 (issue #20 — rate limiting)` section documenting:
    - GCRA via `throttled/v2` with `memstore`, Redis-swappable later.
    - Global middleware + path match strategy.
    - Login dual-key: IP in middleware, IP+email in handler.
    - Admin user ID from JWT cookie, IP fallback.
    - `CF-Connecting-IP` trusted-proxy mode.
    - Policy table reference.
    - Memory budget (~10 MB for 65536 keys).
    - Prediction endpoint excluded per #19.
- [ ] Modify `docs/context/living_brain.md`:
  - [ ] Add a `## Rate Limiting` section under the API contract or as a new top-level section:
    - Architecture: port in `ports/ratelimit.go`, infra in `infra/ratelimit/gcra.go`, middleware
      in `http/middleware/ratelimit.go`.
    - Config env vars: `RATE_LIMIT_ENABLED`, `RATE_LIMIT_STORE_MAX_KEYS`, `RATE_LIMIT_TRUST_PROXY`.
    - Policy table summary.
    - Login dual-key mechanism.
    - Admin user ID keying.
    - Client IP resolution chain.
  - [ ] Add `RATE_LIMIT_*` env vars to the config listing if one exists.
- [ ] Modify `docs/context/recent_changes.md`:
  - [ ] Move current "Latest" to "Previously Latest".
  - [ ] Add new "Latest" section summarizing the rate limiting implementation:
    - Port interface, GCRA infra adapter, middleware, policy table, login dual-key, admin user ID
      keying, client IP resolver, config, wiring, tests.
    - Verification results.
- [ ] Modify `docs/context/change_log.md` (if it exists):
  - [ ] Add chronological entry for the rate limiting work.
- [ ] Run `pnpm format:check` — green.

### Commit

`docs: document rate limiting design and policy`

---

## Verification (all phases complete)

```powershell
# Go checks
cd apps/api-go
go vet ./...
go test ./...

# Formatting
cd ../..
pnpm format:check
```

### Manual Smoke Test (optional, post-deploy)

- Start server locally.
- Burst `POST /auth/login` with wrong credentials → 429 after 6 requests (burst 5 + 1 sustained).
- Verify `Retry-After` header is a positive integer on 429 responses.
- Verify `/health` is never rate limited regardless of request volume.
- Verify `GET /messages` returns 429 after 31 rapid requests (burst 10 + 20 sustained over 1 min).
- Set `RATE_LIMIT_ENABLED=false`, restart → no rate limiting.

---

## Completed Plans

- Client-side prediction migration (issue #19). Completed.
- Channels & Users index search pages. Completed.
- Frontend v2 re-skin (all six routes). Archived under `docs/archive/redesign/`.
- Issue #11: Worker hot-path optimizations. PR #12 merged.
- Issue #9: High-volume ingestion batching/backpressure. Archived under `docs/archive/issue_09/`.
- Go + ClickHouse rewrite (phases 1-9). Archived under `docs/archive/go_rewrite/`.
- Post-MVP features (1-8). Archived under `docs/archive/post_mvp/`.
- MVP (phases 1-10). Archived under `docs/archive/mvp/`.
