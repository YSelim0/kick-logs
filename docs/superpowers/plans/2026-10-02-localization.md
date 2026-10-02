# Issue 28 Localization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Localize every product-owned public/admin interface in EN/TR/DE without changing URLs, source content or data behavior.

**Architecture:** Use request-scoped `next-intl` configuration and a persistent client provider for in-place language changes. Negotiate the initial language from cookie/header, use feature catalogs and shared formatters, and keep admin catalogs out of public page payloads. Only the subscriber TXT formatter needs a narrowly scoped backend extension.

**Tech Stack:** Existing Next.js 14 / React 18, pnpm 8.11.0, TypeScript, Vitest, Tailwind, lucide-react; pinned compatible `next-intl`; existing Go backend.

**Spec:** `docs/superpowers/specs/2026-10-02-localization-design.md` (owner approved).

**Status:** Tasks 1-10 completed and locally verified. Independent review finding fixed with regression tests. Awaiting owner review; no push or merge.

## Global Constraints

- Use `feat/issue-28-localization` for the whole issue, based on `dev` at `d7910fe`.
- Commit feature-sized units locally; never push or merge without the owner's final approval.
- Use `feat(scope): title`; never include collaborator/co-author/assistant attribution.
- Supported locales are exactly `en`, `tr`, `de`; saved cookie wins, then `Accept-Language`, then `en`.
- Cookie: `kick_logs_locale`, one year, host-only, `Path=/`, `SameSite=Lax`, Secure over HTTPS.
- No `/en`, `/tr`, `/de`, `[locale]` route segment, language query parameter, redirect or translated slug.
- Translate product copy only. Keep `Heaven`, usernames, messages, replies, feedback and prediction source text unchanged.
- Preserve datetime-local conversions, end-minute inclusion, UTC chart/report boundaries and subscription expiry.
- No framework upgrade, auth requirement, database migration, query change or ingestion modification.
- One 48px fixed square flag trigger; upward menu with flags and native names; existing dark tokens.
- Keep intermediate feature commits verifiable. Mixed-language intermediate commits are not deployment-ready.
- Read AGENTS.md and the required context; update relevant context in each completed work unit.

## Review Focus

1. Rapid language changes and slow catalog loading: an older selection must never overwrite the newest one (Task 1).
2. Cookies blocked, or navigation into admin after a manual choice: keep the current session usable without a false persistence claim (Tasks 1, 8).
3. An existing error/success notice during switching: translate the notice without repeating its mutation or losing entered values (Tasks 4, 7, 8, 9).
4. A locale change during pagination/polling: no new data request, duplicate results, timer reset or chart remount (Tasks 4, 5, 7).
5. Locale-independent browser controls and CDN HTML reuse: preserve input semantics and verify real production response isolation, not only component tests (Tasks 1, 4, 10).

## File And Interface Map

New shared code under `apps/web/src/i18n/`:

| File                                             | Responsibility / Contract                                                                                                                                                                                                  |
| ------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `locales.ts`                                     | `Locale = "en" \| "tr" \| "de"`, `isLocale(value: unknown): value is Locale`, native names and flag paths.                                                                                                                 |
| `resolve-locale.ts`                              | `resolveLocale(cookie: string \| undefined, acceptLanguage: string \| null): Locale`; pure, no browser/global state.                                                                                                       |
| `preference.ts`                                  | Shared cookie constants; `saveLocalePreference(locale: Locale): boolean`, best-effort browser persistence after allowlist validation.                                                                                      |
| `catalogs.ts`                                    | `CatalogScope = "public" \| "admin"`; `loadMessages(locale: Locale, scope: CatalogScope): Promise<CatalogMessages>`; explicit allowlisted loaders, English fallback.                                                       |
| `request.ts`                                     | `getRequestConfig` reads headers/cookie through Next APIs and supplies the selected public catalog, locale and deterministic initial timezone.                                                                             |
| `locale-provider.tsx`                            | Stable `LocaleProvider` with initial locale/messages; `useLocalePreference()` returns `{locale, changeLocale(locale, scope?): Promise<void>, pending}`. Atomic newest-selection-wins transitions, no locale-keyed remount. |
| `admin-catalog-provider.tsx`                     | `AdminCatalogProvider({initialLocale, initialMessages, children})`; combine common/public messages with selected admin messages only inside admin routes.                                                                  |
| `format.ts`                                      | `createFormatters(locale: Locale, timeZone: string)` returns `number`, `compact`, `percent`, `dateTime`, `dayLabel`, `bytes`; all pure and explicit.                                                                       |
| `use-ui-format.ts`                               | `useUiFormat()` supplies formatters with deterministic UTC before hydration, detected browser timezone afterward; UTC buckets/reports remain explicit.                                                                     |
| `errors.ts`                                      | `getUiErrorKey(error: unknown, operation: UiOperation): UiErrorKey`; typed keys based on HTTP status/operation, never arbitrary error prose.                                                                               |
| `messages/{en,tr,de}/{common,public,admin}.json` | Product catalogs. `common`: navigation/metadata/actions/errors; `public`: landing/search/directory/profiles/prediction/requests/auth; `admin`: management/operations/data.                                                 |
| `types.d.ts`                                     | `next-intl` type augmentation from the complete English catalog shape.                                                                                                                                                     |

`UiOperation` is the union `login | session | logout | search | directory | profile | prediction |
requestSubmit | adminRead | adminMutation | cleanup`; `UiErrorKey` is the key union from
`common.errors`. At minimum provide invalidCredentials, sessionExpired, forbidden, notFound,
conflict, invalidInput, rateLimited, network and unavailable. Unknown failures use unavailable.

`CatalogMessages` is the supported `next-intl` message-tree type, with typed namespace/key access
derived from English catalogs. Files in this table receive adjacent `.test.ts` or `.test.tsx` where
their contracts are independently testable. Do not build a custom translation/ICU engine.

Other new shared files:

- `apps/web/src/middleware.ts`: preference initialization and page-response cache safeguards, no routing.
- `apps/web/src/components/language-switcher.tsx` and `.test.tsx`: flag popover only, no fetching logic.
- `apps/web/src/test/render-with-locale.tsx`: provider-aware test renderer with explicit locale/scope.
- `apps/web/public/language-flags/{english-flag,germany-flag,turkish-flag}.svg`: copied supplied assets.
- `.github/workflows/web-tests.yml`: frontend/catalog validations on push and pull_request.

## Verification Gates

Every frontend feature commit runs, from repository root:

```text
pnpm --filter @kick-logs/web test
pnpm --filter @kick-logs/web typecheck
pnpm --filter @kick-logs/web lint
pnpm --filter @kick-logs/web build
pnpm format:check
git diff --check
```

Backend Task 6 additionally runs, from `apps/api-go`:

```text
gofmt -l .
go test ./...
go vet ./...
```

No output is expected from gofmt. Execute the existing Go CI integration/migration smoke commands
against an isolated fixture database at final verification, never the application's stored data.
Record actual results before committing; a blocked check is reported, not marked passed.

Use a red/green test cycle per task. Keep all three catalogs complete for that task's added keys.
Existing Turkish-oriented tests may use an explicit Turkish provider; add EN/DE assertions rather
than weakening tests to ignore visible copy. Native input chrome is browser-owned; translate its
labels/custom validation and verify its values, not the OS date-picker's language.

## Task 1: Locale Foundation And Validation

**Files:** Create the `i18n/` modules/catalogs/types and test renderer above, `src/middleware.ts`
and `src/middleware.test.ts`. Modify `apps/web/src/app/layout.tsx`, `apps/web/next.config.mjs`,
`apps/web/package.json`, `pnpm-lock.yaml` and existing component `*.test.tsx` files to adopt the
test renderer; create `.github/workflows/web-tests.yml`.

**Interfaces:** Produces all shared interfaces in the map. The public scope loads common/public;
admin scope adds admin. Modules may cache immutable catalogs by locale/scope, never a global
"current language". Initial provider props come from the same request resolver as metadata.

- [x] Write `resolve-locale.test.ts` with these exact expectations plus malformed/absent headers,
      duplicate preferences, regional tags, wildcards, equal priorities and invalid cookie values:

```ts
expect(resolveLocale("de", "tr-TR,en;q=0.8")).toBe("de");
expect(resolveLocale(undefined, "pt-BR, de-AT;q=0.8, tr;q=0.6")).toBe("de");
expect(resolveLocale(undefined, "tr;q=0, en-US;q=0.8")).toBe("en");
expect(resolveLocale("../../other", "pt-BR")).toBe("en");
```

- [x] Add provider tests for identical SSR/hydrated copy, blocked cookie writes, fallback catalog,
      preserved child input state and out-of-order catalog completion. Add error-key tests for 401,
      403, 404, 409, 422, 429, network/unknown errors, without displaying raw exceptions.
- [x] Run `pnpm --filter @kick-logs/web test -- src/i18n` and observe the expected missing-module failures.
- [x] Pin a peer-compatible `next-intl` version through pnpm; implement resolver, explicit catalog
      loaders and stable provider. Initialize the cookie through middleware on page requests only;
      manual selection writes the same validated cookie from the browser. Catch blocked cookie writes
      and retain session state. Do not add a preference API or require authentication.
- [x] Implement atomic locale/catalog changes: preload the requested scope, ignore superseded
      transitions, then update provider, document language/metadata and cookie together. No hard reload,
      route replacement or API fetch. Failed catalog loads leave the previous locale usable.
- [x] Configure root SSR metadata/lang with the selected locale. Middleware must not intercept Go/API
      paths, Next assets or public files. Preserve existing Vary values and prevent shared HTML/RSC caching;
      confirm real headers in Task 10. Leave analytics and asset caches alone.
- [x] Implement locale formatters and semantic error keys. Form/API state stores keys/raw failure
      context, not already-translated prose. Freeze now/timezone in tests; don't derive timezone from locale.
- [x] Wrap existing component tests with `renderWithLocale(ui, {locale: "tr"})` explicitly so later
      shared-header hooks have a real provider without breaking unrelated suites. Support optional `scope`
      and normal RTL render options; preserve the wrapper on rerender. Do not globally mock translation
      hooks or change the product's English fallback to satisfy legacy Turkish test assertions.
- [x] Add catalog tests using an established ICU parser for equal leaf keys, valid syntax, matching
      argument names/types and required plural `other` branches. Allow legitimate locale-specific plural
      categories. Test the checker with deliberately invalid fixtures as well as real catalogs.
- [x] Add a Node 20 / pnpm 8.11.0 frontend workflow using frozen-lockfile install and the frontend gate.
      Ensure typecheck has generated Next types or runs in an order that succeeds on a clean checkout.
- [x] Run the frontend gate; update architecture/contributor guidance for adding translated copy and
      commit only this tested shared foundation, including required context.

## Task 2: Flag Popover And Shared Chrome

**Files:** New switcher/test/assets listed above; modify `src/app/layout.tsx`, `src/app/globals.css`,
`src/components/site-header.tsx`, `src/components/kick-profile-link.tsx`,
`src/components/profile-loading.tsx`, `src/components/ui/dialog.tsx`, `src/app/not-found.tsx`;
add/update their adjacent component tests and common catalogs.

**Interfaces:** Consumes `useLocalePreference`, `Locale` and common translation keys. The selector
passes admin scope for `/admin` and its children, public otherwise. It never controls page data.

- [x] Write tests for collapsed current flag, expanded native names, selected-row indication, choice,
      outside click, Escape/focus return, trigger toggle, arrow/Home/End keys and disabled duplicate requests.
      Assert trigger size is stable and the popover is not a blocking dialog.
- [x] Run `pnpm --filter @kick-logs/web test -- language-switcher` and observe missing-component failure.
- [x] Copy the three supplied SVGs unchanged into the public asset folder. Implement the 48px trigger,
      upward right-aligned menu, accessible names/menu-radio semantics and reduced-motion transition.
      Use 16px safe-area-aware edge spacing and a viewport-constrained menu with 44px minimum row targets.
- [x] Mount once in root layout. Keep it below dialogs/mobile menus; add bottom clearance so page
      controls can scroll past it. Translate shared public navigation, 404, Kick link, loading descriptions
      and close-button accessibility names; do not change route hrefs or brand names.
- [x] Update existing test assertions for formerly hardcoded English navigation now rendered through
      the explicit Turkish provider; retain exact labels and href checks rather than weakening selectors.
- [x] Run selector/shared tests and frontend gate. Inspect narrow/desktop open/closed states, then commit.

## Task 3: Homepage And Identity Directories

**Files:** Modify `features/landing/landing-page.tsx`, `features/directory/directory-index.tsx`,
`features/directory/use-directory-search.ts` only for semantic error storage, their tests,
`features/channel-profile/channels-index-page.test.tsx`,
`features/user-profile/users-index-page.test.tsx` and public catalogs.

**Interfaces:** Use public namespaces `landing`, `directory`, common errors and shared formatters.
Do not change landing/directory APIs, submit behavior, prefix matching or pagination contracts.

- [x] Add locale-switch tests with a loaded homepage and directory results: assert translated headings,
      unchanged `Heaven`, unchanged request counts and maintained query/result state. Cover the six-hour
      review prompt and existing 14-day chart including zero-value days.
- [x] Run `pnpm --filter @kick-logs/web test -- landing directory channels-index-page users-index-page`;
      confirm new English/German expectations fail before migration.
- [x] Move hero/footer/stat/chart/ranking/loading/error/directory copy into catalogs. Keep UTC day
      labels explicit. Preserve the inline request link and user-facing promise of review, not acceptance.
- [x] Keep fetch hooks language-neutral; render notices from keys rather than capturing translator
      functions in request effects. Existing messages/query strings are passed as interpolation values.
- [x] Run targeted tests and frontend gate; inspect German labels and empty states, update context, commit.

## Task 4: Message Search Without State Loss

**Files:** Modify `features/search/search-screen.tsx`, `search-form.tsx`, `message-list.tsx`,
`search-params.ts`, their tests and public catalogs. Inspect `message-content.tsx` but preserve
source rendering/highlighting behavior; do not mechanically replace its Turkish case-folding rules.

**Interfaces:** Keep URL/API conversion, date presets, deduplication and cursor signatures unchanged.
Change display-only filter/preset descriptors to semantic keys; callers translate at render time.
Migrate `formatMessageDate` consumers to shared explicit-locale formatting without changing instants.

- [x] Write tests that submit, paginate, edit an unsent field, change locale and assert the submitted
      results/cursor, unsent field, URL and request count are unchanged. Show a search error, change locale,
      assert its wording updates without another request. Preserve raw reply text and emote/link rendering.
- [x] Keep the existing `searchStateToMessageParams` date tests; add the same form fixture across all
      three locale providers and assert deep-equal request parameters, including end `:59.999` semantics.
- [x] Run `pnpm --filter @kick-logs/web test -- search` and observe new localization assertions fail.
- [x] Migrate form, scope/summary, result counts, pagination, empty/loading/error text, export menu and
      accessibility copy. No locale in URL/search request params; JSON/CSV download behavior is unchanged.
- [x] Remove translated strings from async state and avoid adding `t`/locale dependencies to fetch
      effects/callbacks. Keep input values machine-readable; only their labels/display summaries localize.
- [x] Run search tests and frontend gate, verify state preservation in the browser, update context, commit.

## Task 5: Profiles And Subscriber Dialog

**Files:** Modify `features/user-profile/user-profile-page.tsx`,
`features/channel-profile/channel-profile-page.tsx`, `channel-subscribers-dialog.tsx`, existing profile
tests; create `channel-subscribers-dialog.test.tsx`; update public profile/subscriber catalogs.

**Interfaces:** Consume shared formatters/common loading; keep profile API DTOs, subscription math,
identity links, list `limit/offset/gift_only` and display-only date timezone rules unchanged.

- [x] Add tests for locale changes with an open populated subscriber dialog, appended subscriber page
      and profile latest messages. Assert modal stays open, rows/source usernames stay unchanged and API
      mocks receive no extra calls. Cover loading, 404, generic failure and empty list in EN/TR/DE.
- [x] Run `pnpm --filter @kick-logs/web test -- profile channel-subscribers-dialog`; new copy assertions fail.
- [x] Migrate identity captions, statistics, chart labels/tooltips, rankings, message timestamps and
      subscriber controls. Share UTC chart-day formatting without replacing browser-local message times.
      Keep source names and quote/reply contents out of the translation catalogs.
- [x] Preserve existing modal scrolling/footer bounds and profile layout while allowing German text
      to wrap. Keep backend-generated TXT localization for Task 6, not a client-side second export system.
- [x] Run targeted tests and frontend gate; verify mobile dialog bounds, update context, commit.

## Task 6: Localized Subscriber TXT Export

**Files:** Modify `apps/api-go/internal/http/routes/webhook_admin.go`,
`apps/api-go/internal/http/webhook_admin_routes_test.go`,
`apps/web/src/features/channel-profile/api.ts`, `channel-subscribers-dialog.tsx` and their tests.
Create `apps/api-go/internal/http/routes/subscriber_export_text.go` and its unit test for TXT copy.

**Interfaces:** TXT accepts optional `locale=en|tr|de`; absence retains legacy Turkish output.
Keep `channelSubscribersTXT` as the legacy wrapper and add
`channelSubscribersTXTForLocale(channelSlug string, giftOnly bool, generatedAt time.Time,
items []domain.ChannelSubscriber, locale string) string` for validated inputs.
Extend `buildChannelSubscribersExportUrl(slug, giftOnly, format, baseUrl = API_BASE_URL, locale?)`
with an optional fifth `Locale` argument; existing fourth-argument test/custom-base callers remain valid.

- [x] Add backend tests for absent locale (existing Turkish output), EN/TR/DE headings, gift-only and
      empty cases, unchanged usernames/timestamps and invalid nonempty TXT locale returning 422. Assert
      JSON/CSV bytes/schema remain unchanged even when a locale query parameter is supplied.
- [x] Add frontend URL tests: `locale` is sent only for TXT; JSON/CSV URLs and filter params remain stable.
- [x] Run `go test ./internal/http/... -run 'Subscriber|Subscribers' -count=1` and the relevant frontend
      tests; verify new localization cases fail before implementation.
- [x] Extract the TXT formatter, localize only report labels and validate locale before querying for
      TXT. JSON/CSV ignore locale. Preserve existing UTC RFC3339 output, filenames, schemas and row order.
- [x] Wire the current UI locale into TXT downloads only. Run backend and frontend gates; no migration
      or live-data modification. Update the API/localization documentation and context, then commit.

## Task 7: Prediction, Public Requests And Login

**Files:** Modify `features/prediction/prediction-{search,analysis}-page.tsx`,
`prediction-{distribution,vote-return,top-users}-chart.tsx`, `format.ts`, existing prediction tests;
`features/requests/request-page.tsx` and test; `features/auth/login-screen.tsx`, `auth-errors.ts`,
`use-auth.ts` and tests; update public catalogs.

**Interfaces:** Keep Kick fetching, chart data, outcome colors and the 5000ms polling interval.
`predictionStateBadge` produces a semantic display key/tone while unknown source state remains raw.
Auth/request error presentation consumes typed keys; mutation bodies and session behavior are unchanged.

- [x] Add a fake-timer test that switches language mid-poll and asserts no immediate new request and
      the next request remains on the existing 5000ms schedule. Preserve prediction title/outcomes,
      top-user names and chart mount identity. Add all status/empty/error label cases.
- [x] Add tests that retain a partially completed request and login form on switch, translate already
      visible validation/rate-limit/success states and do not resubmit. Never put passwords in snapshots.
- [x] Run `pnpm --filter @kick-logs/web test -- prediction request-page login-screen`; observe new failures.
- [x] Migrate copy/formatters across these screens and chart tooltips/legends. Use locale number and
      percent formatting instead of prefixing `%`; preserve multiplier values and source outcome strings.
      Keep hooks/timers independent of translation function identity. Translate custom form validation,
      not browser-owned date-picker/credential-manager chrome.
- [x] Run targeted tests and frontend gate; inspect German chart legends and request/login controls,
      update context, commit.

## Task 8: Admin Shell And Management

**Files:** Modify `app/admin/layout.tsx`; extract its existing client shell into
`features/admin/admin-shell.tsx`. Modify `features/channels/channel-admin.tsx`,
`features/users/user-admin.tsx`, `features/requests/request-admin.tsx`, existing admin/channel/user/
request tests and admin catalogs. Implement the mapped `i18n/admin-catalog-provider.tsx` and tests.

**Interfaces:** Server admin layout loads only the request locale's admin catalog and wraps the
existing client auth shell. The nested provider consumes root locale/catalog state, preferring a
preloaded admin catalog on manual switch. Never overwrite a newer client preference with stale props.
For cookie-blocked navigation, load the session-selected admin catalog before showing admin content.

- [x] Test auth-required redirect, super-admin-only navigation, unchanged role/status enums, and all
      admin labels. Test direct admin SSR and public-to-admin navigation for saved and blocked cookies.
- [x] Test switching with an unsaved channel/user form or open request detail: inputs, selected request,
      notes, filters, scroll and pending mutation state persist; no extra API request. Visible operation
      notices retranslate while submitted feedback/notes remain untouched.
- [x] Run `pnpm --filter @kick-logs/web test -- admin channel-admin user-admin request-admin` and observe
      the new localization assertions fail.
- [x] Implement the scoped provider/shell split without weakening auth. Migrate navigation, buttons,
      tables, mobile rows, roles, requests/status/timeline labels and confirmations. Preserve request IDs,
      source content and server-required payload values. Keep admin catalog chunks out of public HTML.
- [x] Run targeted tests and frontend gate; smoke authenticated and anonymous admin access, update
      context, commit. Do not change the backend auth model for localization.

## Task 9: Operations And Data Management

**Files:** Modify `features/operations/operations-dashboard.tsx`, `webhook-health-panel.tsx`,
`failed-events-modal.tsx`, their tests; `features/data-management/data-management-panel.tsx` and test;
admin catalogs. Create `failed-events-modal.test.tsx` if absent.

**Interfaces:** Existing operations/cleanup APIs and diagnostics remain unchanged. Use locale
formatters for counts/bytes and semantic keys for display statuses. Confirmation tokens stay exact.

- [x] Add tests for every webhook status (active, inactive, N errors), backlog/breaker warnings,
      retention choices, preview/confirmation/success/error notices and empty diagnostics in EN/TR/DE.
- [x] Test a locale switch after cleanup preview: same preview ID, exact confirmation token and typed
      input remain; no repeated preview or destructive request. Only a separate explicit confirm submits.
      A notice changes language while a raw diagnostic string and DB table name do not.
- [x] Run `pnpm --filter @kick-logs/web test -- operations webhook-health failed-events data-management`;
      observe new copy/state assertions fail.
- [x] Migrate product labels/captions and accessible names, keeping counters/enum values/data intact.
      Replace translated async-state strings with semantic state; never translate SQL/event names or
      raw logged payloads. Do not alter retries, refresh cadence or mutations.
- [x] Run targeted tests and frontend gate; inspect dense German tables/modals on mobile and desktop,
      update context, commit.

## Task 10: Cross-Language Acceptance And Handoff

**Files:** Add `apps/web/src/i18n/localization-regression.test.tsx`; update focused tests when browser
checks expose issues. Update `README.md`, `docs/architecture.md`, `docs/design/design.md`,
`docs/context/{living_brain,decisions,change_log,recent_changes}.md` and this plan's completed checks.
Create `docs/operations/localization.md` with cookie, cache and rollout verification.

**Interfaces:** No new product contract. This task verifies the combined behavior and deployment.

- [x] Audit all `src/app`, `src/components`, `src/features`, formatter/error helpers and metadata for
      remaining product copy. Check `aria-label`, `title`, alt text, tooltips, constants and validation,
      including pending/error states. Catalog-native language labels and raw data are explicit exceptions.
- [x] Add regression assertions for any missing strings before fixing them; keep a negative fixture
      proving catalog checks fail when a key or ICU argument is removed. Re-run the complete frontend gate.
- [x] Build/start a production web instance on an unused local port. Test direct requests with
      `Accept-Language: de-DE`, `tr-TR`, unsupported `pt-BR`, no header and conflicting locale cookies.
      Assert correct HTML/lang/metadata, initial cookie attributes and no cross-visitor cache reuse.
- [x] Verify production HTML/RSC cache headers and middleware exclusions. Document that Cloudflare must
      bypass HTML/RSC caching; static assets remain cacheable. Do not change VPS/Cloudflare settings here.
- [x] Browser-check every public/admin route in EN/TR/DE at desktop and narrow mobile widths. Exercise
      reload, navigation, language switching, errors, export, menus and dialogs; check console hydration,
      horizontal overflow, bottom-button reachability, keyboard focus and reduced motion. Keep screenshots
      outside the repo. Report inaccessible tooling as incomplete verification, not success.
- [x] Run `go test ./...`, `go vet ./...`, gofmt and the integration/migration smoke commands from
      `.github/workflows/go-tests.yml` on disposable fixtures. Never reuse production/app DBs for tests.
- [x] Record rollout: rebuild web and API for the localized TXT path; listener/processor/ClickHouse/NATS
      need no rebuild for this issue. Existing data remains untouched. Record any unavoidable native-browser
      language behavior and cookie-blocking persistence limitations without claiming them fixed.
- [x] Perform a whole-branch review against the spec; resolve findings and rerun affected checks.
      Commit the final verified corrections/docs, report commit list and verification, then stop for the
      owner's review. Leave the branch local, issue open and `dev` unmerged until instructed otherwise.

## Execution Handoff

Recommended: implement natively in this session, one task/commit at a time, with a separate final
review. Translation and component changes share provider/error/formatter contracts, so serial
integration avoids catalog ownership conflicts. The owner may instead choose fresh implementer and
reviewer agents per task; every agent must read both this plan and the approved design.

The written plan must be reviewed and execution approach confirmed before product-code changes.
