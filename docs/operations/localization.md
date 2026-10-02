# Localization Operations

## Runtime Contract

- Public and admin pages support English, Turkish and German on their existing URLs.
- Initial language: valid `kick_logs_locale` cookie, supported `Accept-Language` preference,
  then English. Regional tags such as `de-DE` and `tr-TR` are normalized.
- The preference cookie is host-only, lasts one year, uses `Path=/; SameSite=Lax`, and is
  Secure on HTTPS. It is not an authentication cookie and contains no user identity.
- The bottom-right flag menu changes language without navigation, remounting a screen or
  repeating its data requests. It preserves drafts, results, cursors and prediction polling.
- If a public-page language change finishes after entering admin, the mounted admin screen keeps
  its previous working catalog until the replacement is ready. A failed load offers retry without
  discarding its forms or repeating authentication/data reads.
- Chat/replies, names, slugs, prediction source text, submitted notes and raw diagnostics remain
  unchanged. Machine enums, confirmation tokens and query timestamps are not translated.
- Locale changes number/date presentation, not timezone or subscription calculations.
  Native date pickers and browser validation chrome may follow the browser/OS language.
- If cookies are blocked, manual selection works in the current page session and client-side
  navigation. A full reload/new visit may negotiate again; persistence cannot be guaranteed.

## Deploy

After reviewing and pulling the issue branch's changes:

```bash
docker compose build api web
docker compose up -d --no-deps api web
```

Only web and API images need rebuilding for this issue. Listener, processor, NATS and ClickHouse
need no rebuild. There are no schema migrations or historical-data modifications. The API change
only adds an optional validated `locale=en|tr|de` to subscriber TXT downloads. Missing/empty
locale retains the legacy Turkish report. JSON/CSV schemas and data stay unchanged.

Do not run `down -v` as part of this update. Existing environment values and ports stay intact.
If rolling back, redeploy both prior images to keep the frontend/TXT export contract aligned.

## Reverse Proxy And CDN

HTML and RSC responses are visitor-specific and must not be shared across visitors:

- Respect `Cache-Control: private, no-store`. Disable any Cache Everything/page-cache rule for
  application HTML and RSC responses, including requests with `RSC: 1` or an `_rsc` query.
- Never cache responses that write the preference cookie or personalized admin responses.
- Keep static hashed assets under `/_next/static/` cacheable; API analytics caching is unchanged.
- Forward the original HTTPS scheme and cookies to Next.js. Verify `Secure` on the locale cookie
  at the public HTTPS URL, not only on a direct local HTTP request.

Middleware adds language/cookie Vary values, but Next.js 14's final renderer can replace them with
its own RSC Vary list. Shared-cache safety therefore relies on the verified private/no-store
headers and the CDN bypass rule, not on Vary alone. No production proxy/CDN settings were changed
by this branch; inspect the deployed response again after release.

## Smoke Checks

```bash
curl -sD - -o /dev/null -H 'Accept-Language: de-DE' https://kicklogs.net/channels
curl -sD - -o /dev/null -H 'Accept-Language: tr-TR' https://kicklogs.net/channels
curl -sD - -o /dev/null -H 'Accept-Language: pt-BR' https://kicklogs.net/channels
curl -sD - -o /dev/null -H 'Accept-Language: de-DE' -H 'Cookie: kick_logs_locale=tr' https://kicklogs.net/channels
curl -sD - -o /dev/null -H 'RSC: 1' https://kicklogs.net/channels?_rsc=locale-check
```

Inspect the corresponding HTML: `html lang` and channel-page title should be German, Turkish,
English and Turkish respectively. A request without a language header/cookie defaults to English.
New preferences get one-year host-only cookies; existing valid preferences are not rewritten.
HTML/RSC should be private/no-store, never CDN HIT. Static assets should not write locale cookies.

In a browser, switch languages with an unsent search/request draft and loaded results; verify the
URL and source content stay unchanged. Reload to verify persistence. Check admin screens, a long
German mobile page, subscriber/cleanup dialogs and keyboard Escape/focus behavior. Test TXT in all
three languages; JSON/CSV remain machine-readable exports rather than translated schemas.

## Verification Record

Local production preview: separate port 3102, existing Docker stack unchanged. Checked all 16
public/admin screen variants in EN/TR/DE at 390px and 1440px, using deterministic browser-only API
fixtures. No horizontal page overflow or hydration errors occurred. Populated prediction charts,
subscriber/failed-event/request dialogs and cleanup previews were inspected without real mutations.
Cookie-blocked client navigation retained the selected language; keyboard Escape restored focus.
An isolated Playwright browser with reduced motion enabled confirmed that selector transitions are
disabled (`transition-property: none`) and keyboard focus returns to the trigger.

Direct production-mode HTTP checks covered supported/unsupported/absent language headers, cookie
precedence, HTML/RSC no-store and Secure cookies behind an HTTPS forwarded scheme. These local
checks do not claim that the production CDN configuration has been verified.

Frontend tests cover catalog key/ICU parity, missing-key/argument negative fixtures, SSR hydration,
rapid selection races, failures, cookie blocking, preserved drafts/pagination and unchanged polling.
Go tests cover the optional TXT locale and legacy output. Go unit/vet, ClickHouse repository
integration and SQLite/ClickHouse migration smoke checks ran on disposable fixtures, not app data.
The final frontend suite contains 288 passing tests across 42 files; lint, typecheck, production
build and repository Prettier checks passed. Independent review's admin remount finding was
reproduced and fixed with deferred-load/retry tests. Remote GitHub CI awaits the owner's push.
On this Windows checkout, `gofmt -l .` also traverses ignored dependency caches; checking the
Git-tracked Go files is the equivalent clean-checkout CI source gate.
