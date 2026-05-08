# Agent Instructions

Before making any code, docs, config, dependency, database, Docker, or commit changes in this repository, every agent must read:

1. `docs/project_plan.md`
2. `docs/context/living_brain.md`
3. `docs/context/decisions.md`
4. `docs/context/change_log.md`
5. `docs/context/recent_changes.md`

Keep these files current. When implementation decisions change, update the relevant context document in the same unit of work.
Use `recent_changes.md` for the latest short handoff summary after each meaningful change, and `change_log.md` for chronological history.

## Project Defaults

- Monorepo name: `kick-logs`
- Backend: Python, FastAPI, PostgreSQL
- Python tooling: `uv`
- Frontend: Next.js, pnpm, Tailwind, shadcn/ui, lucide-react
- Runtime: Docker Compose dev stack
- Commit format: `feat(scope): title`
- Do not push unless explicitly asked.
- Do not commit secrets, `.env`, virtual environments, logs, caches, or dependency folders.
