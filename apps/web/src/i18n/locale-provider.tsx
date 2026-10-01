"use client";

import { NextIntlClientProvider } from "next-intl";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode
} from "react";
import { loadMessages, type CatalogMessages, type CatalogScope } from "./catalogs";
import { isLocale, type Locale } from "./locales";
import { saveLocalePreference } from "./preference";

type Preference = {
  locale: Locale;
  messages: CatalogMessages;
  pending: boolean;
  failed: boolean;
  timeZone: string;
  changeLocale: (locale: Locale, scope?: CatalogScope) => Promise<void>;
};
const PreferenceContext = createContext<Preference | null>(null);

export function LocaleProvider({
  initialLocale,
  initialMessages,
  children
}: {
  initialLocale: Locale;
  initialMessages: CatalogMessages;
  children: ReactNode;
}) {
  const [selection, setSelection] = useState({ locale: initialLocale, messages: initialMessages });
  const [pending, setPending] = useState(false);
  const [failed, setFailed] = useState(false);
  const [timeZone, setTimeZone] = useState("UTC");
  const request = useRef(0);

  useEffect(() => {
    setTimeZone(Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC");
  }, []);
  useEffect(() => {
    document.documentElement.lang = selection.locale;
    const common = selection.messages.common as CatalogMessages;
    const metadata = common.metadata as CatalogMessages;
    document
      .querySelector('meta[name="description"]')
      ?.setAttribute("content", String(metadata.description));
  }, [selection]);

  const changeLocale = useCallback(async (locale: Locale, scope: CatalogScope = "public") => {
    if (!isLocale(locale)) return;
    const sequence = ++request.current;
    setPending(true);
    setFailed(false);
    try {
      const messages = await loadMessages(locale, scope);
      if (sequence !== request.current) return;
      saveLocalePreference(locale);
      setSelection({ locale, messages });
    } catch {
      if (sequence === request.current) setFailed(true);
    } finally {
      if (sequence === request.current) setPending(false);
    }
  }, []);
  const value = useMemo(
    () => ({ ...selection, pending, failed, timeZone, changeLocale }),
    [selection, pending, failed, timeZone, changeLocale]
  );

  return (
    <PreferenceContext.Provider value={value}>
      <NextIntlClientProvider
        locale={selection.locale}
        messages={selection.messages}
        timeZone={timeZone}
      >
        {children}
      </NextIntlClientProvider>
    </PreferenceContext.Provider>
  );
}

export function useLocalePreference() {
  const context = useContext(PreferenceContext);
  if (!context) throw new Error("LocaleProvider is required");
  return context;
}
