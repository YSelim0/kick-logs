import { NextRequest, NextResponse } from "next/server";
import { isLocale } from "./i18n/locales";
import { LOCALE_COOKIE, LOCALE_COOKIE_MAX_AGE } from "./i18n/preference";
import { resolveLocale } from "./i18n/resolve-locale";

export function protectLocalizedResponse(response: NextResponse) {
  const vary = new Set(
    (response.headers.get("Vary") || "")
      .split(",")
      .map((value) => value.trim())
      .filter(Boolean)
  );
  vary.add("Accept-Language");
  vary.add("Cookie");
  response.headers.set("Vary", [...vary].join(", "));
  response.headers.set("Cache-Control", "private, no-store");
}

export function middleware(request: NextRequest) {
  const response = NextResponse.next();
  const path = request.nextUrl.pathname;
  if (
    path === "/api" ||
    path.startsWith("/api/") ||
    path.startsWith("/_next/") ||
    path.includes(".")
  )
    return response;
  const saved = request.cookies.get(LOCALE_COOKIE)?.value;
  if (!isLocale(saved)) {
    response.cookies.set(
      LOCALE_COOKIE,
      resolveLocale(saved, request.headers.get("accept-language")),
      {
        path: "/",
        sameSite: "lax",
        maxAge: LOCALE_COOKIE_MAX_AGE,
        secure: request.nextUrl.protocol === "https:"
      }
    );
  }
  protectLocalizedResponse(response);
  return response;
}

export const config = { matcher: ["/((?!api|_next|.*\\..*).*)"] };
