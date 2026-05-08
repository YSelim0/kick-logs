# Decisions

## 2026-05-09

- Use a monorepo in `kick-logs`.
- Use Python for backend and listener.
- Use `uv` for Python project/dependency management.
- Use FastAPI for the backend API.
- Use PostgreSQL for persistence.
- Run PostgreSQL in Docker.
- Use Docker Compose as the default local runtime.
- Use a development Docker stack with hot reload.
- Use Next.js for frontend.
- Use pnpm as frontend package manager.
- Use Tailwind, shadcn/ui, and lucide-react for frontend UI.
- Do not implement production authentication in MVP unless requested later.
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
