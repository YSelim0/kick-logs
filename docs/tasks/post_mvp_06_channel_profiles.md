# Post-MVP Feature 6 Tasks: Channel / Publisher Profile Analytics

## Goal

Add public channel profile pages that combine stored Kick channel metadata,
analytics, and recent message history.

## Scope

This feature owns channel/publisher profiles. Do not build user profile pages
or data management tools here.

## Backend Tasks

- [x] Add public channel analytics endpoint keyed by channel slug.
- [x] Return channel metadata, total messages, first logged message, latest logged message, top
      senders, top emotes, message volume, and latest messages.
- [x] Return 404 for unknown channel slugs.
- [x] Add backend tests for existing channel, unknown channel, message volume, top senders,
      top emotes, and latest messages.

## Frontend Tasks

- [ ] Add `/channels/[slug]` page.
- [ ] Link channel labels in search results to channel profiles.
- [ ] Link admin channel rows to channel profiles when slug data is present.
- [ ] Render channel summary, activity stats, volume chart/list, top senders, top emotes, and
      latest messages.
- [ ] Add a link from the profile to `/search` with channel filter prefilled.
- [ ] Add loading, empty, and not-found states.
- [ ] Add frontend tests for page rendering, search/admin links, and not-found behavior.

## Docs And Acceptance

- [ ] Update README features/routes.
- [ ] Update context docs.
- [ ] Verify backend tests and frontend tests/typecheck/lint/build.
- [ ] Acceptance: a visitor can inspect a logged channel's activity and jump into filtered
      message search.
