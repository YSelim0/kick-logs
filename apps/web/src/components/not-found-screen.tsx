"use client";
import { ArrowLeft, Search } from "lucide-react";
import { useTranslations } from "next-intl";
import Link from "next/link";

import { SiteHeader } from "@/components/site-header";
import { Button } from "@/components/ui/button";

export function NotFoundScreen() {
  const t = useTranslations("common.notFound");
  const nav = useTranslations("common.navigation");
  return (
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute={null} />

      <section className="mx-auto flex min-h-[calc(100vh-3.5rem)] max-w-[1280px] items-center px-6 py-16">
        <div className="grid w-full gap-8 lg:grid-cols-[minmax(0,1fr)_360px] lg:items-center">
          <div className="max-w-2xl">
            <p className="font-mono text-2xs uppercase tracking-wider text-accent">404</p>
            <h1 className="mt-4 text-[32px] font-semibold leading-tight text-foreground md:text-[40px]">
              {t("title")}
            </h1>
            <p className="mt-4 max-w-xl text-sm leading-6 text-muted-foreground md:text-base">
              {t("description")}
            </p>

            <div className="mt-7 flex flex-col gap-2 sm:flex-row">
              <Button asChild>
                <Link href="/search">
                  <Search className="h-4 w-4" />
                  {t("search")}
                </Link>
              </Button>
              <Button asChild variant="outline">
                <Link href="/">
                  <ArrowLeft className="h-4 w-4" />
                  {t("home")}
                </Link>
              </Button>
            </div>
          </div>

          <div className="rounded-lg border border-border bg-panel p-5">
            <div className="mb-4 flex items-center justify-between border-b border-border pb-3">
              <span className="font-mono text-2xs uppercase text-muted-foreground">
                {t("quickLinks")}
              </span>
              <span className="h-1.5 w-1.5 rounded-full bg-accent" aria-hidden />
            </div>
            <nav className="grid gap-2" aria-label={t("navigation")}>
              <QuickLink href="/channels" label={nav("channels")} description={t("channels")} />
              <QuickLink href="/users" label={nav("users")} description={t("users")} />
              <QuickLink
                href="/prediction"
                label={nav("prediction")}
                description={t("prediction")}
              />
              <QuickLink href="/request" label={nav("request")} description={t("request")} />
            </nav>
          </div>
        </div>
      </section>
    </main>
  );
}

function QuickLink({
  description,
  href,
  label
}: {
  description: string;
  href: string;
  label: string;
}) {
  return (
    <Link
      className="rounded-md border border-transparent bg-elevated px-3 py-2.5 transition-colors hover:border-border-strong hover:bg-secondary/50"
      href={href}
    >
      <span className="block text-[13px] font-semibold text-foreground">{label}</span>
      <span className="mt-0.5 block text-xs text-muted-foreground">{description}</span>
    </Link>
  );
}
