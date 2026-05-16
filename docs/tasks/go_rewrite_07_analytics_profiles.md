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

- [x] Implement `GET /analytics/overview`.
- [x] Implement `GET /analytics/message-volume`.
- [x] Implement `GET /analytics/top-senders`.
- [x] Implement `GET /analytics/top-channels`.
- [x] Implement `GET /analytics/top-emotes`.
- [x] Implement `GET /users/{slug}/analytics`.
- [x] Implement `GET /channels/{slug}/analytics`.
- [x] Support date filters for analytics endpoints.
- [x] Support channel scope where current endpoints support it.
- [x] Support sender scope where current endpoints support it.
- [x] Preserve `bucket=hour|day` behavior for message volume.
- [x] Preserve top-list limit validation and clamping.
- [x] Preserve user slug lookup behavior, including Kick-style `_` to `-` profile slugs.
- [x] Preserve 404 behavior for unknown user/channel profiles.
- [x] Include latest messages in profile responses using the same message item shape.

## Tests And Checks

- [x] Overview totals match seeded ClickHouse messages.
- [x] Message volume buckets match seeded timestamps.
- [x] Top sender/channel/emote aggregations are correct.
- [x] Date filters scope analytics correctly.
- [x] Sender-scoped and channel-scoped analytics are correct.
- [x] User profile response includes identity, totals, top channels, top emotes, volume, and latest
      messages.
- [x] Channel profile response includes metadata, totals, top senders, top emotes, volume, and latest
      messages.
- [x] Unknown user/channel slugs return 404.

## Acceptance Criteria

- [x] Landing analytics can use the Go API.
- [x] User profile pages can use the Go API.
- [x] Channel profile pages can use the Go API.
- [x] Public analytics routes remain unauthenticated.

## Commit Boundary

Commit analytics/profile parity after message search and ingestion are stable.
