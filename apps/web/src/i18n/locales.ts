export const LOCALES = ["en", "tr", "de"] as const;
export type Locale = (typeof LOCALES)[number];

export const LANGUAGE_OPTIONS = [
  { locale: "tr", name: "Türkçe", flag: "/language-flags/turkish-flag.svg" },
  { locale: "en", name: "English", flag: "/language-flags/english-flag.svg" },
  { locale: "de", name: "Deutsch", flag: "/language-flags/germany-flag.svg" }
] as const;

export function isLocale(value: unknown): value is Locale {
  return typeof value === "string" && LOCALES.includes(value as Locale);
}
