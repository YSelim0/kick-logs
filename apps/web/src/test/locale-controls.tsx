import { useLocalePreference } from "@/i18n/locale-provider";
import { LOCALES } from "@/i18n/locales";

export function LocaleTestControls() {
  const { changeLocale } = useLocalePreference();
  return (
    <>
      {LOCALES.map((locale) => (
        <button key={locale} onClick={() => void changeLocale(locale, "admin")}>
          Switch to {locale}
        </button>
      ))}
    </>
  );
}
