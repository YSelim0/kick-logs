import { isLocale, type Locale } from "./locales";

export function resolveLocale(cookie: string | undefined, acceptLanguage: string | null): Locale {
  if (isLocale(cookie)) return cookie;

  const candidates = (acceptLanguage ?? "")
    .slice(0, 4096)
    .split(",")
    .flatMap((entry, index) => {
      const match =
        /^\s*([a-z]{2,8}(?:-[a-z0-9]{1,8})*|\*)(?:\s*;\s*q=(0(?:\.\d{0,3})?|1(?:\.0{0,3})?))?\s*$/i.exec(
          entry
        );
      if (!match) return [];
      const quality = match[2] === undefined ? 1 : Number(match[2]);
      const locale = match[1].toLowerCase().split("-")[0];
      return quality > 0 && isLocale(locale) ? [{ locale, quality, index }] : [];
    });

  candidates.sort((a, b) => b.quality - a.quality || a.index - b.index);
  return candidates[0]?.locale ?? "en";
}
