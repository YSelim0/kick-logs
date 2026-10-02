import type { Metadata } from "next";
import { GeistMono } from "geist/font/mono";
import { GeistSans } from "geist/font/sans";
import { Suspense } from "react";
import { getLocale, getMessages } from "next-intl/server";

import "./globals.css";
import { NavigationProgress } from "@/components/navigation-progress";
import { LocaleProvider } from "@/i18n/locale-provider";
import { LanguageSwitcher } from "@/components/language-switcher";
import { PageMetadata } from "@/i18n/page-metadata";
import { getPageMetadata } from "@/i18n/server-metadata";

const sharedMetadata: Metadata = {
  manifest: "/site.webmanifest",
  icons: {
    icon: [
      { url: "/favicon.ico", sizes: "any" },
      { url: "/favicon.svg", type: "image/svg+xml" },
      { url: "/favicon-96x96.png", sizes: "96x96", type: "image/png" }
    ],
    shortcut: "/favicon.ico",
    apple: "/apple-touch-icon.png"
  }
};

export async function generateMetadata(): Promise<Metadata> {
  return { ...sharedMetadata, ...(await getPageMetadata("/")) };
}

export default async function RootLayout({
  children
}: Readonly<{
  children: React.ReactNode;
}>) {
  const locale = await getLocale();
  const messages = await getMessages();
  return (
    <html lang={locale} className={`dark ${GeistSans.variable} ${GeistMono.variable}`}>
      <body>
        <LocaleProvider initialLocale={locale} initialMessages={messages}>
          <Suspense>
            <PageMetadata />
            <NavigationProgress />
          </Suspense>
          {children}
          <LanguageSwitcher />
        </LocaleProvider>
      </body>
    </html>
  );
}
