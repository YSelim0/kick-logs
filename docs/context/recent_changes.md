# Recent Changes

This file is the short handoff summary of the latest project changes. Keep it concise and update it after each meaningful change so the next agent can quickly see what just happened.

## Latest

- Added `docs/design/design.md` as the UI/UX source of truth.
- Tracked `docs/design/design.pen` as the editable design artifact.
- Locked the backend-first rule: no UI implementation until the backend API and listener work end-to-end.
- Documented search UI fields, route decisions, result card requirements, admin UI requirements, and emote/avatar rendering rules.
- `AGENTS.md` now requires UI/frontend agents to read `docs/design/design.md`.
- Documented that multi-agent development is allowed for non-overlapping work scopes.
- Updated UI direction to fixed dark theme with palette `#26001B`, `#810034`, `#FF005C`, `#FFF600`, black, and white.
- Added the first `/search` screen design to `docs/design/design.pen`.
- The search reference image is structural guidance only; the final design should not copy the green visual style exactly.
- The user-provided logo should be used where a product mark is needed.
- Refined search results to use one shared outer list container with stacked message rows instead of per-message modal/card components.
- Search result avatars are circular, and emotes render inline inside message content.
- Clarified that `/search` is public and does not require login.
- Clarified that `/admin` is the authenticated backend management dashboard for tasks such as managing followed channels.
- Admin panel design is intentionally deferred until the search screen is approved.

## Commit Context

- Last committed docs unit:
  - `c7681ba feat(docs): add architecture plan`
