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
- Use optional `AND` search filters with case-insensitive contains matching.
- Store raw Kick payloads and all useful normalized fields.
- Enrich sender profile images through Kick web endpoints when possible.
- Parse `[emote:id:name]` tokens and render image fallback URLs.
- Use `/search`, `/admin`, and reserve `/` for later landing content.
- Allow multi-agent development for non-overlapping work scopes.
- Use `docs/implementation_plan.md` as the sequential MVP implementation plan.
- Use `docs/tasks/phaseN_tasks.md` files as phase-scoped task contracts; agents must not cross into later phase scope without explicit direction.
- Do not add placeholder `web` or `listener` services in Phase 1; add each service only in its owning phase.

## 2026-05-10

- Public `/search` date inputs default to the last 7 days: `Başlangıç` is current local date/time minus 7 days and `Bitiş` is current local date/time. Users can clear date fields to omit date filters.
- MVP root route `/` redirects to `/search`; future landing content can replace this deliberately after the application screens are stable.
