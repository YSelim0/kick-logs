"use client";

import { NextIntlClientProvider, useTranslations } from "next-intl";
import { useEffect, useState, type ReactNode } from "react";
import { loadAdminMessages, type CatalogMessages } from "./catalogs";
import { useLocalePreference } from "./locale-provider";
import type { Locale } from "./locales";

export function AdminCatalogProvider({
  initialLocale,
  initialMessages,
  children
}: {
  initialLocale: Locale;
  initialMessages: CatalogMessages;
  children: ReactNode;
}) {
  const { locale, messages, timeZone } = useLocalePreference();
  const t = useTranslations("common");
  const [loaded, setLoaded] = useState({ locale: initialLocale, messages: initialMessages });
  const [lastReady, setLastReady] = useState<{
    locale: Locale;
    messages: CatalogMessages;
  } | null>(() =>
    initialLocale === locale ? { locale, messages: { ...messages, ...initialMessages } } : null
  );
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);
  // A manual switch preloads the admin scope in the root provider. Navigation with blocked
  // cookies may instead bring an older server locale; only load the current session's catalog.
  const adminMessages = messages.adminShell
    ? messages
    : loaded.locale === locale
      ? loaded.messages
      : null;

  useEffect(() => {
    if (adminMessages) {
      setLastReady({ locale, messages: { ...messages, ...adminMessages } });
      return;
    }
    let cancelled = false;
    setFailed(false);
    void loadAdminMessages(locale).then(
      (next) => {
        if (!cancelled) setLoaded({ locale, messages: next });
      },
      () => {
        if (!cancelled) setFailed(true);
      }
    );
    return () => {
      cancelled = true;
    };
  }, [adminMessages, messages, locale, retry]);

  // A public catalog can finish loading after the user has already entered admin.
  // Keep its mounted forms alive until the matching admin catalog is ready.
  const ready = adminMessages ? { locale, messages: { ...messages, ...adminMessages } } : lastReady;

  if (!ready) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3 px-6 text-sm text-muted-foreground">
        <p role="status">{failed ? t("errors.languageSwitch") : t("actions.loading")}</p>
        {failed ? (
          <button type="button" onClick={() => setRetry((value) => value + 1)}>
            {t("actions.retry")}
          </button>
        ) : null}
      </div>
    );
  }
  return (
    <NextIntlClientProvider locale={ready.locale} messages={ready.messages} timeZone={timeZone}>
      {!adminMessages ? (
        <div className="flex flex-wrap items-center justify-center gap-3 px-6 py-3 text-sm text-muted-foreground">
          <p role="status">{failed ? t("errors.languageSwitch") : t("actions.loading")}</p>
          {failed ? (
            <button type="button" onClick={() => setRetry((value) => value + 1)}>
              {t("actions.retry")}
            </button>
          ) : null}
        </div>
      ) : null}
      {children}
    </NextIntlClientProvider>
  );
}
