import { cookies, headers } from "next/headers";
import { getRequestConfig } from "next-intl/server";
import { loadMessages } from "./catalogs";
import { LOCALE_COOKIE } from "./preference";
import { resolveLocale } from "./resolve-locale";

export default getRequestConfig(async () => {
  const locale = resolveLocale(
    cookies().get(LOCALE_COOKIE)?.value,
    headers().get("accept-language")
  );
  return { locale, messages: await loadMessages(locale, "public"), timeZone: "UTC" };
});
