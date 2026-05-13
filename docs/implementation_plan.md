# Kick Logs Post-MVP Development Plan

This is the active implementation plan for feature development after the MVP.
The original MVP plan is archived at `docs/archive/mvp_implementation_plan.md`, with
its old phase task files under `docs/archive/tasks/`.

## Execution Rules

- Work features in order unless the user explicitly reprioritizes.
- Each feature has a matching task file under `docs/tasks/`.
- Keep commits scoped to completed units inside the current feature.
- Backend/API contracts and tests should land before UI when a feature needs new data.
- UI changes must follow `docs/design/design.md`.
- Keep `docs/context/recent_changes.md` and `docs/context/change_log.md` current after
  meaningful changes.
- Do not continue using archived MVP task files as active implementation scope.

## Feature Map

| Feature | Task File                                        | Goal                                | Depends On         |
| ------- | ------------------------------------------------ | ----------------------------------- | ------------------ |
| 1       | `docs/tasks/post_mvp_01_admin_operations.md`     | Admin operations dashboard          | Current MVP        |
| 2       | `docs/tasks/post_mvp_02_search_improvements.md`  | Search quality, filters, and export | Feature 1 optional |
| 3       | `docs/tasks/post_mvp_03_analytics_foundation.md` | Shared analytics API foundation     | Current MVP        |
| 4       | `docs/tasks/post_mvp_04_landing_analytics.md`    | Landing page with analytics blocks  | Feature 3          |
| 5       | `docs/tasks/post_mvp_05_user_profiles.md`        | User profile analytics              | Feature 3          |
| 6       | `docs/tasks/post_mvp_06_channel_profiles.md`     | Channel/publisher profile analytics | Feature 3          |
| 7       | `docs/tasks/post_mvp_07_data_management.md`      | Admin data retention and cleanup    | Feature 1          |
| 8       | `docs/tasks/post_mvp_08_final_smoke.md`          | Full smoke, docs, and cleanup       | Features 1-7       |

## Feature 1: Admin Operations Dashboard

Give admins a single place to see whether the self-hosted system is healthy.

Expected output:

- Admin-only operations summary API.
- Database size, table sizes, and core row counts.
- Raw event status counts for pending, processing, processed, and failed.
- Listener heartbeat/freshness tracking so the UI can show whether ingestion is alive even
  when channels are quiet.
- `/admin` dashboard section with compact operational cards.

## Feature 2: Search Improvements

Improve the public search workflow without changing the core search contract.

Expected output:

- Highlight matched search text in result rows.
- Render URLs inside message content as safe clickable links.
- Date presets: last 1 hour, 24 hours, 7 days, and 30 days.
- Additional filters for reply-only and emote-only messages.
- CSV and JSON export for filtered results with a safe maximum export size.
- Tests for query mapping, highlighting, presets, filters, and export behavior.

## Feature 3: Analytics Foundation

Build reusable backend analytics endpoints for landing, user profiles, and channel profiles.

Expected output:

- Public read-only analytics endpoints for overview, message volume, top senders, top
  channels, and top emotes.
- Shared application use cases and repository query methods for aggregate stats.
- Date range and optional channel/sender scoping where relevant.
- Backend tests that verify aggregation correctness against seeded message data.

## Feature 4: Landing Page With Analytics

Replace the root redirect with a useful public landing page backed by real analytics.

Expected output:

- `/` becomes a public landing page instead of redirecting to `/search`.
- Landing page includes project identity, concise self-hosted positioning, and analytics
  blocks from Feature 3.
- The page links clearly to `/search`, `/admin`, GitHub, and the support page.
- Design stays dark, compact, and product-focused rather than oversized marketing treatment.

## Feature 5: User Profile Analytics

Expose searchable sender-level analytics and history.

Expected output:

- Public `/users/[slug]` profile page.
- User analytics API for totals, first/last seen, most active channels, top emotes, message
  volume, and latest messages.
- Search result sender names link to user profiles when slug data is available.
- Profile page integrates with existing search behavior for deeper message history.

## Feature 6: Channel / Publisher Profile Analytics

Expose channel-level analytics and history.

Expected output:

- Public `/channels/[slug]` profile page.
- Channel analytics API for totals, message volume, top users, top emotes, recent activity,
  and latest messages.
- Search rows and admin channel rows link to channel profiles.
- Profile page uses channel metadata already stored from Kick resolution.

## Feature 7: Data Management

Add admin-only controls for managing data growth and dangerous cleanup operations.

Expected output:

- Admin data summary screen with database/table sizes and retention state.
- Retention settings for messages and raw events, defaulting to keep forever until changed.
- Dry-run cleanup previews before destructive actions.
- Confirmed cleanup flows for old raw events, old messages, a specific channel, or a
  specific sender.
- Backup/restore and cleanup docs for self-hosted operators.

## Feature 8: Final Smoke And Documentation

Close the post-MVP feature set with verification and documentation.

Expected output:

- Full backend and frontend checks pass.
- Docker Compose smoke covers admin dashboard, search, landing analytics, user profiles,
  channel profiles, and data management.
- README and docs describe the new features and operational commands.
- Context docs reflect the final post-MVP state.
