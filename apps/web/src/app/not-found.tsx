import { ArrowLeft, Search } from "lucide-react";
import Link from "next/link";

import { SiteHeader } from "@/components/site-header";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute={null} />

      <section className="mx-auto flex min-h-[calc(100vh-3.5rem)] max-w-[1280px] items-center px-6 py-16">
        <div className="grid w-full gap-8 lg:grid-cols-[minmax(0,1fr)_360px] lg:items-center">
          <div className="max-w-2xl">
            <p className="font-mono text-2xs uppercase tracking-wider text-accent">404</p>
            <h1 className="mt-4 text-[32px] font-semibold leading-tight text-foreground md:text-[40px]">
              Aradığın sayfa bulunamadı.
            </h1>
            <p className="mt-4 max-w-xl text-sm leading-6 text-muted-foreground md:text-base">
              Bu bağlantı taşınmış, silinmiş veya hiç oluşmamış olabilir. Log aramasına dönebilir ya
              da kayıtlı kanal ve kullanıcı sayfalarını inceleyebilirsin.
            </p>

            <div className="mt-7 flex flex-col gap-2 sm:flex-row">
              <Button asChild>
                <Link href="/search">
                  <Search className="h-4 w-4" />
                  Arama sayfası
                </Link>
              </Button>
              <Button asChild variant="outline">
                <Link href="/">
                  <ArrowLeft className="h-4 w-4" />
                  Ana sayfaya dön
                </Link>
              </Button>
            </div>
          </div>

          <div className="rounded-lg border border-border bg-panel p-5">
            <div className="mb-4 flex items-center justify-between border-b border-border pb-3">
              <span className="font-mono text-2xs uppercase text-muted-foreground">
                Hızlı bağlantılar
              </span>
              <span className="h-1.5 w-1.5 rounded-full bg-accent" aria-hidden />
            </div>
            <nav className="grid gap-2" aria-label="404 hızlı bağlantılar">
              <QuickLink
                href="/channels"
                label="Kanallar"
                description="Loglanan kanalları keşfet"
              />
              <QuickLink
                href="/users"
                label="Kullanıcılar"
                description="Kullanıcı profillerine bak"
              />
              <QuickLink
                href="/prediction"
                label="Prediction"
                description="Tahmin panellerini aç"
              />
              <QuickLink
                href="/request"
                label="Talep"
                description="Kanal veya geri bildirim gönder"
              />
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
