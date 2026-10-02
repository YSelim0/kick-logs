import { render, type RenderOptions } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { LocaleProvider } from "@/i18n/locale-provider";
import type { CatalogScope } from "@/i18n/catalogs";
import type { Locale } from "@/i18n/locales";
import enCommon from "@/i18n/messages/en/common.json";
import trCommon from "@/i18n/messages/tr/common.json";
import deCommon from "@/i18n/messages/de/common.json";
import enPublic from "@/i18n/messages/en/public.json";
import trPublic from "@/i18n/messages/tr/public.json";
import dePublic from "@/i18n/messages/de/public.json";
import enAdmin from "@/i18n/messages/en/admin.json";
import trAdmin from "@/i18n/messages/tr/admin.json";
import deAdmin from "@/i18n/messages/de/admin.json";

const messages = {
  en: { common: enCommon, public: enPublic, admin: enAdmin },
  tr: { common: trCommon, public: trPublic, admin: trAdmin },
  de: { common: deCommon, public: dePublic, admin: deAdmin }
};

export function renderWithLocale(
  ui: ReactElement,
  {
    locale,
    scope = "public",
    wrapper: Wrapper,
    ...options
  }: RenderOptions & { locale: Locale; scope?: CatalogScope }
) {
  const selected = messages[locale];
  const catalog = {
    common: selected.common,
    ...selected.public,
    ...(scope === "admin" ? selected.admin : {})
  };
  return render(ui, {
    ...options,
    wrapper: ({ children }: { children: ReactNode }) => (
      <LocaleProvider initialLocale={locale} initialMessages={catalog}>
        {Wrapper ? <Wrapper>{children}</Wrapper> : children}
      </LocaleProvider>
    )
  });
}

export function createLocaleRenderer(locale: Locale, scope: CatalogScope = "public") {
  return (ui: ReactElement, options?: RenderOptions) =>
    renderWithLocale(ui, { ...options, locale, scope });
}
