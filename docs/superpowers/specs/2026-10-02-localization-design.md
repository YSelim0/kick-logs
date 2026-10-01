# Issue 28: Localization Design

Status: written design approved by the owner. Implementation plan awaiting review:
`docs/superpowers/plans/2026-10-02-localization.md`.

Issue: <https://github.com/YSelim0/kick-logs/issues/28>

Branch: `feat/issue-28-localization`, created from local `dev` at `d7910fe`.

## Outcome

Make every public and admin workflow usable in English, Turkish and German, without requiring
an account or changing existing URLs. Translate product-owned interface text only. Preserve
user content, stored data, API semantics, filters and the existing dark design.

The owner's latest flag-selector decision supersedes the issue's original text-only navigation
selector. This document specifies the target behavior, not completed implementation.

## Language Resolution

1. Use a valid saved preference: exactly `en`, `tr` or `de`.
2. Otherwise negotiate the request's `Accept-Language`, respecting quality and preference order.
   Normalize supported regional tags; ignore zero-quality or malformed entries.
3. Use English if no supported language is available.

Persist the first resolved preference and subsequent manual selections in a host-only
`kick_logs_locale` cookie: one-year lifetime, `Path=/`, `SameSite=Lax`, secure over HTTPS.
It is separate from authentication and contains no user identity. Do not use geolocation.
If cookies are blocked, initial negotiation still works and manual selection remains usable
in the current page session; persistence across reloads cannot be guaranteed.

The initial HTML and first client render must use the same locale. Do not initially show
Turkish and replace it after hydration. Set `<html lang>` and product metadata accordingly.

## Fixed Flag Selector

- Render one shared selector on public, login, admin and not-found screens.
- Keep a square 48px trigger fixed at the bottom-right, with safe-area-aware edge spacing.
- The collapsed trigger contains only the selected language's flag. Give it a localized accessible
  name and tooltip identifying the current language and the language-change action.
- Clicking opens a compact menu above the trigger, aligned to its right edge. It is a popover,
  not a page-blocking modal: no fullscreen backdrop, focus trap or body scroll lock.
- Rows contain the supplied flag and native language name: `Türkçe`, `English`, and `Deutsch`.
  Keep these native labels recognizable in every locale.
- Highlight the current choice with an elevated dark row and expose its selected state to
  assistive technology. Flags supplement names; they are not the sole accessible labels.
- Selecting a language updates interface copy and the trigger flag, saves the preference, and
  closes the menu. Selecting the current language simply closes it.
- Outside click, Escape, or another trigger click closes the menu. Keyboard activation and
  arrow-key navigation work; closing with Escape returns focus to the trigger.
- Preserve route, query parameters, scroll position, entered form values, loaded results and
  open feature state. Do not hard-reload or remount screens based on the locale.
- Use the existing dark tokens, subtle border, 6px trigger corners and up to 8px menu corners.
  The reference defines interaction/layout, not a new purple palette, glow or blur.
- Keep the selector below full-screen dialogs/mobile navigation in the stacking order. Reserve
  enough bottom content space to allow page actions/footer links to scroll clear of the trigger.
- Position the menu within narrow viewports. Use a short reduced-motion-aware open transition.

Copy the owner's three SVG assets into `apps/web/public/language-flags/`, preserving the
original external asset folder. Use fixed image dimensions and do not stretch their aspect ratios.
Do not commit the reference screenshot.

## Shared Architecture

Use `next-intl` without locale routing or a `[locale]` route segment. Version 4.14.8 advertises
Next.js 14 and React 18 peer compatibility; verify the pinned version during installation.
Keep the current framework versions and ordinary Next links.

Centralize the following responsibilities under `apps/web/src/i18n/`:

- Supported locale definitions, pure negotiation, cookie policy and tests.
- Request-scoped server configuration and matching client provider configuration.
- Preference persistence and in-place switching without changing API fetch dependencies.
- Typed, feature-oriented catalogs and shared date/number formatting.
- Localized, operation-aware UI error presentation.

Use stable semantic keys grouped by common/navigation, landing, search, directory/profiles,
prediction, requests, authentication, admin, operations and data management. English is the
fallback catalog; all three catalogs must have matching keys and interpolation/plural contracts.
Load only the necessary locale/catalog scope; do not expose admin-only copy in every public page
when it can remain in the admin provider. Switching language must not refetch analytics/history.

Cookie/header-based rendering makes page HTML request-specific. Prevent shared caching of
localized HTML, preference writes and personalized RSC responses. Do not disable static asset
caching or the backend's language-neutral analytics caches. Document the matching CDN constraint.

## Translation Boundaries

Translate headings, navigation, actions, labels, placeholders, validation, empty/loading/error
states, dialogs, chart labels, captions, tooltips, accessibility copy, metadata and product-owned
human-readable export labels. Update already-visible UI notices when language changes.

Never translate chat/reply text, channel/user names or slugs, submitted requests/notes, Kick
prediction titles/outcomes, emote names, IDs, URLs, API keys/enums, cursor values or stored payloads.
For example, a channel named `Heaven` remains `Heaven` in every language.

Display localized status labels separately from the machine values used in API requests. Preserve
exact cleanup confirmation tokens. Do not translate raw operational diagnostics or turn exception
messages into translation keys. UI errors use existing status plus operation context, localized
client validation and a safe localized fallback; do not match arbitrary backend prose.

The existing subscriber TXT export contains Turkish product copy generated by Go. Add an optional
validated `locale=en|tr|de` for TXT only, passed by the UI; omission retains its legacy Turkish
output. JSON/CSV fields, values and schema remain unchanged. No new queries or historical data
migration are needed. API paths remain unchanged.

## Dates And Numbers

Format product numbers, percentages, plurals and dates for the selected language. Locale is not
a timezone: changing language must never shift an instant, chart bucket, search boundary or
subscription expiry. Keep UTC chart/report semantics and browser-local datetime input semantics.

Keep server/client formatting deterministic before hydration. Browser timezone detection, where
needed for existing local display, happens after hydration without changing form values. Preserve
the search end-minute rule and ISO timestamps sent to the API.

## Delivery Boundaries

Implement foundation first, then migrate public and admin feature groups in meaningful commits,
then complete cross-language verification. The implementation plan will specify exact files,
tests and commit boundaries after this design is reviewed.

No new account requirement, database migration, ingestion change, query optimization, framework
upgrade, translated URL, automatic machine translation or redesign belongs to this issue.
Use this one branch throughout. Do not push. Merge into `dev` only after the owner's final approval.

## Acceptance Evidence

- Negotiation tests cover regional tags, quality/order, absent/malformed headers, unsupported
  languages, invalid cookies and saved-preference precedence.
- Provider/selector tests cover all three languages, outside click, Escape, keyboard access,
  selected flag, persistence and cookie-disabled selection.
- Regression tests preserve route/query/form/result state and prove a locale change does not
  initiate extra analytics/history requests or reset prediction polling.
- Catalog tests reject missing keys, malformed ICU messages and mismatched interpolation/plurals.
- Browser checks cover initial server/client language agreement, reload/navigation persistence,
  reduced motion and longer German labels on mobile and desktop, including admin workflows.
- Source content such as `Heaven`, chat messages and prediction choices is unchanged in each locale.
- Frontend tests, typecheck, lint, build and Prettier pass before relevant commits. Run Go checks
  for the TXT-export change; never claim remote CI passed before the owner pushes.

## References

- [next-intl App Router and preference-based locale setup](https://next-intl.dev/docs/getting-started/app-router)
- [next-intl request configuration](https://next-intl.dev/docs/usage/configuration)
- Existing UI source of truth: `docs/design/design.md`.
