# Go Rewrite Phase 7: Analytics And Profiles

## Scope

Rebuild analytics endpoints, user profile analytics, and channel profile analytics in Go.

This phase owns aggregate ClickHouse queries and public profile response mapping.

## Out Of Scope

- Do not change frontend profile page design.
- Do not implement migration.
- Do not change message search response behavior except shared helper fixes required by analytics.
- Do not change public access boundaries.

## Checklist

- [ ] Implement `GET /analytics/overview`.
- [ ] Implement `GET /analytics/message-volume`.
- [ ] Implement `GET /analytics/top-senders`.
- [ ] Implement `GET /analytics/top-channels`.
- [ ] Implement `GET /analytics/top-emotes`.
- [ ] Implement `GET /users/{slug}/analytics`.
- [ ] Implement `GET /channels/{slug}/analytics`.
- [ ] Support date filters for analytics endpoints.
- [ ] Support channel scope where current endpoints support it.
- [ ] Support sender scope where current endpoints support it.
- [ ] Preserve `bucket=hour|day` behavior for message volume.
- [ ] Preserve top-list limit validation and clamping.
- [ ] Preserve user slug lookup behavior, including Kick-style `_` to `-` profile slugs.
- [ ] Preserve 404 behavior for unknown user/channel profiles.
- [ ] Include latest messages in profile responses using the same message item shape.

## Tests And Checks

- [ ] Overview totals match seeded ClickHouse messages.
- [ ] Message volume buckets match seeded timestamps.
- [ ] Top sender/channel/emote aggregations are correct.
- [ ] Date filters scope analytics correctly.
- [ ] Sender-scoped and channel-scoped analytics are correct.
- [ ] User profile response includes identity, totals, top channels, top emotes, volume, and latest
      messages.
- [ ] Channel profile response includes metadata, totals, top senders, top emotes, volume, and latest
      messages.
- [ ] Unknown user/channel slugs return 404.

## Acceptance Criteria

- [ ] Landing analytics can use the Go API.
- [ ] User profile pages can use the Go API.
- [ ] Channel profile pages can use the Go API.
- [ ] Public analytics routes remain unauthenticated.

## Commit Boundary

Commit analytics/profile parity after message search and ingestion are stable.
