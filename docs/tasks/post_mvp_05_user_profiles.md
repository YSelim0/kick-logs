# Post-MVP Feature 5 Tasks: User Profile Analytics

## Goal

Add public sender profile pages that combine identity, analytics, and recent
message history.

## Scope

This feature owns user/sender profiles. Do not build channel profile pages or
data management tools here.

## Backend Tasks

- [ ] Add public user analytics endpoint keyed by sender slug.
- [ ] Return sender identity, profile image URL, total messages, first seen, last seen, most
      active channels, top emotes, message volume, and latest messages.
- [ ] Return 404 for unknown sender slugs.
- [ ] Add backend tests for existing sender, unknown sender, message volume, top channels,
      top emotes, and latest messages.

## Frontend Tasks

- [ ] Add `/users/[slug]` page.
- [ ] Link sender names/avatars in search results to user profiles when slug data is present.
- [ ] Render profile summary, activity stats, volume chart/list, top channels, top emotes, and
      latest messages.
- [ ] Add a link from the profile to `/search` with sender filter prefilled.
- [ ] Add loading, empty, and not-found states.
- [ ] Add frontend tests for page rendering, search-row links, and not-found behavior.

## Docs And Acceptance

- [ ] Update README features/routes.
- [ ] Update context docs.
- [ ] Verify backend tests and frontend tests/typecheck/lint/build.
- [ ] Acceptance: a visitor can click a sender from search and understand that user's public
      activity across logged channels.
