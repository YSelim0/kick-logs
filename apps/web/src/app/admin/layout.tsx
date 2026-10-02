import { getLocale } from "next-intl/server";
import { AdminShell } from "@/features/admin/admin-shell";
import { AdminCatalogProvider } from "@/i18n/admin-catalog-provider";
import { loadAdminMessages } from "@/i18n/catalogs";
import { isLocale } from "@/i18n/locales";

export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  const requestLocale = await getLocale();
  const locale = isLocale(requestLocale) ? requestLocale : "en";
  const messages = await loadAdminMessages(locale);
  return (
    <AdminCatalogProvider initialLocale={locale} initialMessages={messages}>
      <AdminShell>{children}</AdminShell>
    </AdminCatalogProvider>
  );
}
