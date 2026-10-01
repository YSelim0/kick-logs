import { isLocale, type Locale } from "./locales";

export const LOCALE_COOKIE = "kick_logs_locale";
export const LOCALE_COOKIE_MAX_AGE = 31536000;

export function saveLocalePreference(locale: Locale): boolean {
  if (!isLocale(locale)) return false;
  try {
    document.cookie = `${LOCALE_COOKIE}=${locale}; Path=/; Max-Age=${LOCALE_COOKIE_MAX_AGE}; SameSite=Lax${location.protocol === "https:" ? "; Secure" : ""}`;
    return document.cookie
      .split(";")
      .some((cookie) => cookie.trim() === `${LOCALE_COOKIE}=${locale}`);
  } catch {
    return false;
  }
}
