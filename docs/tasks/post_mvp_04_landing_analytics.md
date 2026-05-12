# Post-MVP Feature 4 Tasks: Landing Page With Analytics

## Goal

Replace the root redirect with a public landing page that explains the
self-hosted project and shows useful live analytics.

## Scope

This feature owns `/` only. Do not build user profile pages, channel profile
pages, new analytics backend contracts, or admin cleanup tools here.

## Frontend Tasks

- [ ] Change `/` from redirect to a real page.
- [ ] Use Feature 3 analytics endpoints for total logged messages, tracked channels, active
      senders, recent activity, top channels, and top emotes.
- [ ] Add clear navigation to `/search`, `/admin`, GitHub, and support links.
- [ ] Keep the page dark, compact, and project-focused; avoid oversized hero treatment.
- [ ] Include loading and empty-data states that still make the page useful on a fresh install.
- [ ] Add frontend tests for analytics rendering, empty state, and navigation links.

## Docs And Acceptance

- [ ] Update `docs/design/design.md` with landing page behavior.
- [ ] Update README route list so `/` is no longer described as a redirect.
- [ ] Update context docs.
- [ ] Verify frontend tests/typecheck/lint/build.
- [ ] Acceptance: visitors opening `/` see what Kick Logs is, current public analytics, and a
      clear path into search.
