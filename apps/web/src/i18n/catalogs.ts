import type { AbstractIntlMessages } from "next-intl";
import { isLocale, type Locale } from "./locales";

export type CatalogScope = "public" | "admin";
export type CatalogMessages = AbstractIntlMessages;

const loaders = {
  en: {
    common: () => import("./messages/en/common.json"),
    public: () => import("./messages/en/public.json"),
    admin: () => import("./messages/en/admin.json")
  },
  tr: {
    common: () => import("./messages/tr/common.json"),
    public: () => import("./messages/tr/public.json"),
    admin: () => import("./messages/tr/admin.json")
  },
  de: {
    common: () => import("./messages/de/common.json"),
    public: () => import("./messages/de/public.json"),
    admin: () => import("./messages/de/admin.json")
  }
};

export function mergeMessages(
  fallback: CatalogMessages,
  selected: CatalogMessages
): CatalogMessages {
  const result = { ...fallback };
  for (const [key, value] of Object.entries(selected)) {
    const previous = result[key];
    result[key] =
      typeof value === "object" && typeof previous === "object"
        ? mergeMessages(previous, value)
        : value;
  }
  return result;
}

async function readMessages(locale: Locale, scope: CatalogScope): Promise<CatalogMessages> {
  const common = (await loaders[locale].common()).default;
  const publicMessages = (await loaders[locale].public()).default;
  const admin = scope === "admin" ? (await loaders[locale].admin()).default : {};
  return { common, ...publicMessages, ...admin };
}

export async function loadMessages(locale: Locale, scope: CatalogScope): Promise<CatalogMessages> {
  const fallback = await readMessages("en", scope);
  if (!isLocale(locale) || locale === "en") return fallback;
  return mergeMessages(fallback, await readMessages(locale, scope));
}
