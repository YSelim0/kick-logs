# Decisions

## 2026-05-09

- Use a monorepo in `kick-logs`.
- Use Python for backend and listener.
- Use `uv` for Python project/dependency management.
- Use FastAPI for the backend API.
- Use PostgreSQL for persistence.
- Use SQLAlchemy 2.x async ORM with asyncpg for PostgreSQL access.
- Use Alembic for database migrations.
- Use pragmatic clean architecture for backend code.
- Keep domain entities independent from SQLAlchemy, FastAPI, Pydantic, and external clients.
- Use one Python backend package shared by API and listener entrypoints.
- Run PostgreSQL in Docker.
- Use Docker Compose as the default local runtime.
- Use a development Docker stack with hot reload.
- Use Next.js for frontend.
- Use pnpm as frontend package manager.
- Use Tailwind, shadcn/ui, and lucide-react for frontend UI.
- Defer UI implementation until the backend API is working end-to-end.
- Use `docs/design/design.md` as the source of truth for UI/UX decisions.
- Use a fixed dark-only UI theme.
- Use UI palette `#26001B`, `#810034`, `#FF005C`, `#FFF600`, black, and white.
- Prefer `#FFF600` for primary buttons.
- Do not use blur, glow, colored lighting, or atmospheric background effects.
- Keep UI typography compact; do not use landing-page-scale text in app screens.
- Keep button/control corner radii modest for a serious professional feel.
- Use the provided search UI reference as structural guidance only; do not copy the green visual style exactly.
- Design the `/search` screen first and wait for approval before designing admin panel screens.
- Use the user-provided logo asset where a product mark is needed.
- Search results use one shared outer list container with stacked message rows, not per-message modal/card components.
- Sender avatars in search results should be fully circular.
- Emotes should render inline where they appear in message content.
- `/search` is public and does not require login.
- `/admin` is an authenticated backend management dashboard for operational tasks such as managing followed channels.
- Implement admin authentication in MVP; production hardening can be refined later.
- Use Kick web Pusher chat events, not official Kick webhooks, for MVP ingestion.
- Use commit message format `feat(scope): title`.
- Store messages indefinitely.
- Implement full admin login in MVP.
- Seed default super admin with `admin@kicklogs.local` / `admin123`, overridable by env.
- Super admin can create new admin users.
- Use one listener worker for all enabled channels.
- Add date range filters to search.
- Use optional `AND` search filters; sender is exact case-insensitive, while channel and content are case-insensitive contains.
- Store raw Kick payloads and all useful normalized fields.
- Enrich sender profile images through Kick web endpoints when possible.
- Parse `[emote:id:name]` tokens and render image fallback URLs.
- Use `/` for the public landing page, `/search` for public message search, and `/admin` for
  authenticated backend management.
- Allow multi-agent development for non-overlapping work scopes.
- The original MVP used a sequential phase implementation plan; that completed plan now lives in `docs/archive/`.
- Active implementation agents must use the current `docs/implementation_plan.md` and matching active task file.
- Do not add placeholder `web` or `listener` services in Phase 1; add each service only in its owning phase.

## 2026-05-10

- Public `/search` date inputs default to the last 7 days: `Başlangıç` is current local date/time minus 7 days and `Bitiş` is current local date/time. Users can clear date fields to omit date filters.
- MVP started search-first at `/search`; post-MVP work can use `/` for compact landing content.

## 2026-05-12

- Bare `/search` page load does not automatically fetch latest messages; the result area stays idle with `Arama yapmak için yukarıdaki formu kullanın.` until the user submits a search.
- Explicitly submitting the search form with empty filters still fetches latest messages.
- `/search` date inputs stay as local `datetime-local` values in the UI/URL, but API requests convert them to UTC ISO strings; `end` includes the full selected minute.
- `/search` reply rows show the replied-to sender and replied-to message content above the current message in muted gray text.
- Reply rendering uses `message_type === "reply"`, `reply_metadata.original_sender.username`, and `reply_metadata.original_message.content`; long reply previews expose the full original content through a `title` attribute.
- Repository sponsorship uses Buy Me a Coffee account `yavuzselim` through GitHub `FUNDING.yml` and README links.
- The completed MVP implementation plan is archived under `docs/archive/`; active work uses the post-MVP feature plan in `docs/implementation_plan.md`.
- Post-MVP development is split into feature-scoped task files under `docs/tasks/post_mvp_*.md`.
- The selected post-MVP roadmap prioritizes admin operations, search improvements, analytics, landing analytics, user/channel profiles, and admin data management.

## 2026-05-13

- Public `/messages` sender filtering uses case-insensitive exact matching against sender username/slug snapshots; channel and content filters remain case-insensitive contains matching.
- Post-MVP Feature 1 stores listener heartbeat state in PostgreSQL instead of inferring
  liveness from message timestamps, because quiet channels can be healthy but produce no
  messages.
- Admin operations metrics are exposed through `GET /admin/operations/summary` and remain
  authenticated admin-only.
- Post-MVP Feature 2 will render URLs found inside message content as safe clickable links in
  `/search` result rows. Link rendering must not break inline emotes or matched-text
  highlighting.
- `/search` date presets update only the date fields and keep other filters intact.
- `/search` CSV/JSON export actions use the last submitted filters, not unsent form edits.
- `/search` keeps secondary controls compact: quick date ranges are a select, exports sit
  behind one square download icon, and reply/emote filters use explicit `Sadece ...` labels.
- `/search` export menu must close on outside click.
- `/search` keeps date controls on their own row; result-type filters sit to the left of the
  `İşlem` action group so the date row does not feel cramped.
- Analytics endpoints are public read-only contracts under `/analytics/*` for future landing,
  user profile, and channel profile screens.
- Analytics `sender` scope uses case-insensitive exact sender username/slug matching;
  analytics `channel` scope uses case-insensitive exact channel slug/display-name matching.

## 2026-05-14

- Public `/` is a compact landing page, not a redirect. It explains the self-hosted project and
  loads public analytics from Feature 3 endpoints.
- Landing message volume uses a recent day-bucket range, while overview/top-list cards summarize
  current stored data.
- Landing navigation links to `/search`, `/admin`, GitHub, and Buy Me a Coffee support.
- Landing design must stay dark, compact, product-focused, and avoid oversized hero treatment.
- Header brand/logo areas in `/search` and `/admin` navigate to `/`.
- Public user profiles live at `/users/[slug]` and use `GET /users/{slug}/analytics`.
- Search result sender names and avatars link to public user profiles when sender slug exists.
- `/search` reply preview sender names also link to `/users/[slug]`; when Kick reply metadata has
  no slug, the frontend derives a lowercase username fallback.
- `/users/[slug]` top identity blocks use the same rounded bordered panel treatment as the rest of
  the profile sections.
- Public sender profile URLs follow Kick's profile slug behavior: chat usernames can display with
  underscores, but profile routes convert `_` to `-`; backend profile/search lookups accept both
  forms so existing underscore-stored data keeps working.

## 2026-05-16

- Go rewrite work starts in parallel under `apps/api-go`; Python remains the default runtime until
  explicit cutover.
- Phase 2 Go workspace uses the Go standard library for the initial HTTP server, routing,
  middleware, config, and logging to avoid unnecessary early dependencies.
- The optional Go API Compose service is named `api-go`, gated behind profile `go-rewrite`, and
  maps to host port `GO_API_PORT` or `8001` by default.
- Go local build outputs and caches stay untracked and outside Docker build context under
  `apps/api-go/bin/`, `apps/api-go/.gocache/`, `apps/api-go/.gomodcache/`, and
  `apps/api-go/.cache/`.
- The Go rewrite uses SQLite for control-plane state and ClickHouse for message/raw-event data.
- SQLite stores admin users, followed channels, sender profile cache, retention settings, worker
  heartbeats, and migration bookkeeping.
- ClickHouse stores denormalized chat messages, raw Kick events, and raw-event processing attempts.
- Go rewrite migrations are run through `cmd/migrate`; Compose exposes that binary as the
  `migrate-go` service behind profile `go-rewrite`.
- Go rewrite default super-admin seeding happens in SQLite migration startup and stores a bcrypt
  hash, not the plain password.
- Go rewrite auth preserves the Python cookie contract and uses HS256 JWTs with `sub`, `iat`, and
  `exp` claims.
- Go rewrite API startup may apply SQLite and ClickHouse migrations for local developer ergonomics;
  `migrate-go` remains the explicit migration command for Compose setup.
- Go rewrite admin channel deletion remains disable-only to preserve historical chat data.
- Go rewrite public message search/export reads denormalized ClickHouse `chat_messages` directly;
  the hot search path must not join back to SQLite.
