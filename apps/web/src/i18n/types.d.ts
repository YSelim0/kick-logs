import common from "./messages/en/common.json";
import publicMessages from "./messages/en/public.json";
import admin from "./messages/en/admin.json";
import type { Locale as SupportedLocale } from "./locales";

declare module "next-intl" {
  interface AppConfig {
    Locale: SupportedLocale;
    Messages: { common: typeof common } & typeof publicMessages & typeof admin;
  }
}
