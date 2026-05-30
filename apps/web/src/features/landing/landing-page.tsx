"use client";

/* eslint-disable @next/next/no-img-element */

import { Copyright, Github, Search } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

import {
  getAnalyticsOverview,
  getMessageVolume,
  getTopChannels,
  getTopEmotes,
  getTopSenders
} from "@/features/analytics/api";
import { Button } from "@/components/ui/button";
import { SiteHeader } from "@/components/site-header";
import { cn } from "@/lib/utils";
import type {
  AnalyticsOverview,
  MessageVolumePoint,
  TopChannelAnalytics,
  TopEmoteAnalytics,
  TopSenderAnalytics
} from "@/types/api";

type LandingAnalyticsState = {
  overview: AnalyticsOverview;
  volume: MessageVolumePoint[];
  topChannels: TopChannelAnalytics[];
  topEmotes: TopEmoteAnalytics[];
  topSenders: TopSenderAnalytics[];
};

const EMPTY_OVERVIEW: AnalyticsOverview = {
  total_messages: 0,
  total_senders: 0,
  total_channels: 0,
  total_emote_usages: 0,
  first_message_at: null,
  latest_message_at: null
};

export function LandingPage() {
  const [analytics, setAnalytics] = useState<LandingAnalyticsState | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");

  useEffect(() => {
    let isMounted = true;

    async function loadAnalytics() {
      setStatus("loading");

      try {
        const recentRange = getRecentVolumeRange();
        const [overview, volume, topChannels, topEmotes, topSenders] = await Promise.all([
          getAnalyticsOverview(),
          getMessageVolume({ bucket: "day", end: recentRange.end, start: recentRange.start }),
          getTopChannels({ limit: 5 }),
          getTopEmotes({ limit: 5 }),
          getTopSenders({ limit: 5 })
        ]);

        if (!isMounted) {
          return;
        }

        setAnalytics({
          overview,
          volume: volume.items,
          topChannels: topChannels.items,
          topEmotes: topEmotes.items,
          topSenders: topSenders.items
        });
        setStatus("ready");
      } catch {
        if (isMounted) {
          setAnalytics(null);
          setStatus("error");
        }
      }
    }

    void loadAnalytics();

    return () => {
      isMounted = false;
    };
  }, []);

  const overview = analytics?.overview ?? EMPTY_OVERVIEW;

  return (
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute="search" />

      <div className="mx-auto max-w-[1280px] px-6 py-16 md:pt-16 md:pb-20">
        <div className="flex flex-col gap-12">
          <Hero />
          <StatsBar overview={overview} />
          <StatusBanner status={status} />
          <AnalyticsGrid
            volume={analytics?.volume ?? []}
            topChannels={analytics?.topChannels ?? []}
            topSenders={analytics?.topSenders ?? []}
            topEmotes={analytics?.topEmotes ?? []}
          />
        </div>
      </div>
      <Footer />
    </main>
  );
}

function Footer() {
  return (
    <footer className="border-t border-border py-3">
      <div className="mx-auto max-w-[1280px] px-6">
        <p className="flex items-center gap-1.5 font-mono text-2xs uppercase text-muted-foreground">
          <Copyright className="h-3 w-3 shrink-0" aria-hidden />
          {new Date().getFullYear()} kick-logs · Tüm hakları saklıdır.
        </p>
      </div>
    </footer>
  );
}

function Hero() {
  return (
    <section className="flex flex-col gap-5">
      <span className="inline-flex w-fit items-center gap-1.5 rounded-full border border-border bg-panel px-2.5 py-1 font-mono text-2xs uppercase tracking-wider text-muted-foreground">
        <span aria-hidden className="h-1.5 w-1.5 rounded-full bg-accent" />
        Self-hosted · Açık kaynak
      </span>
      <h1 className="max-w-3xl text-4xl font-semibold leading-[1.1] tracking-[-0.02em] text-foreground md:text-[48px]">
        Kick chat için kalıcı log.
      </h1>
      <p className="max-w-[720px] text-base leading-relaxed text-muted-foreground">
        Takip ettiğin Kick kanallarındaki tüm mesajları kaydet, ara, analiz et. Veri sende kalır.
      </p>
      <div className="mt-1 flex flex-wrap items-center gap-2.5">
        <Button asChild>
          <Link href="/search">
            <Search className="h-4 w-4" />
            Arama başlat
          </Link>
        </Button>
        <Button asChild variant="outline">
          <a href="https://github.com/YSelim0/kick-logs" rel="noopener noreferrer" target="_blank">
            <Github className="h-4 w-4" />
            GitHub
          </a>
        </Button>
      </div>
    </section>
  );
}

function StatsBar({ overview }: { overview: AnalyticsOverview }) {
  const cells: { label: string; value: string }[] = [
    { label: "TOPLAM MESAJ", value: formatCompactNumber(overview.total_messages) },
    { label: "KANAL", value: formatCompactNumber(overview.total_channels) },
    { label: "KULLANICI", value: formatCompactNumber(overview.total_senders) },
    { label: "EMOTE", value: formatCompactNumber(overview.total_emote_usages) }
  ];

  return (
    <section
      aria-label="Genel metrikler"
      className="grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border md:grid-cols-4"
    >
      {cells.map((cell) => (
        <div key={cell.label} className="bg-panel px-6 py-5">
          <div className="font-mono text-2xs uppercase text-muted-foreground">{cell.label}</div>
          <div className="mt-2 text-[26px] font-semibold leading-none text-foreground">
            {cell.value}
          </div>
        </div>
      ))}
    </section>
  );
}

function AnalyticsGrid({
  volume,
  topChannels,
  topSenders,
  topEmotes
}: {
  volume: MessageVolumePoint[];
  topChannels: TopChannelAnalytics[];
  topSenders: TopSenderAnalytics[];
  topEmotes: TopEmoteAnalytics[];
}) {
  return (
    <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
      <Panel title="Mesaj hacmi" subtitle="son 14 gün">
        <MessageVolumeChart points={volume} />
      </Panel>
      <Panel title="Top kanallar" subtitle="mesaj sayısı">
        <TopList
          rows={topChannels.map((channel) => ({
            key: String(channel.channel_id),
            label: channel.display_name,
            value: formatCompactNumber(channel.message_count),
            href: `/channels/${encodeURIComponent(channel.slug)}`,
            image: channel.profile_image_url ?? undefined,
            initial: getInitial(channel.display_name)
          }))}
          emptyText="Kanal verisi henüz yok."
        />
      </Panel>
      <Panel title="Top kullanıcılar" subtitle="mesaj sayısı">
        <TopList
          rows={topSenders.map((sender) => ({
            key: String(sender.sender_id),
            label: sender.username,
            value: formatCompactNumber(sender.message_count),
            href: `/users/${encodeURIComponent(sender.slug)}`,
            image: sender.profile_image_url ?? undefined,
            initial: getInitial(sender.username)
          }))}
          emptyText="Kullanıcı verisi henüz yok."
        />
      </Panel>
      <Panel title="Top emoteler" subtitle="kullanım">
        <TopList
          rows={topEmotes.map((emote) => ({
            key: emote.id,
            label: emote.name,
            value: formatCompactNumber(emote.usage_count),
            image: emote.image_url
          }))}
          emptyText="Emote verisi henüz yok."
        />
      </Panel>
    </div>
  );
}

function Panel({
  title,
  subtitle,
  children
}: {
  title: string;
  subtitle: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-lg border border-border bg-panel p-5">
      <header className="mb-4 flex flex-col gap-0.5">
        <h2 className="text-[15px] font-semibold leading-none text-foreground">{title}</h2>
        <p className="font-mono text-2xs uppercase text-muted-foreground">{subtitle}</p>
      </header>
      {children}
    </section>
  );
}

function MessageVolumeChart({ points }: { points: MessageVolumePoint[] }) {
  if (!points.length) {
    return <EmptyHint text="Henüz veri yok." />;
  }

  const max = points.reduce((acc, point) => Math.max(acc, point.message_count), 0);

  return (
    <div className="relative flex h-44 items-end gap-1.5">
      {points.map((point) => {
        const ratio = max > 0 ? point.message_count / max : 0;
        const heightPct = max > 0 ? Math.max(ratio * 100, 4) : 4;
        return (
          <div
            key={point.bucket_start}
            className="group relative flex h-full flex-1 flex-col justify-end"
          >
            <div
              className="pointer-events-none absolute left-1/2 z-10 flex -translate-x-1/2 flex-col items-center gap-0.5 whitespace-nowrap rounded-md border border-border bg-elevated px-2.5 py-1.5 opacity-0 shadow-lg transition-opacity duration-100 group-hover:opacity-100"
              style={{ bottom: `calc(${heightPct}% + 8px)` }}
              aria-hidden
            >
              <span className="font-mono text-[11px] font-semibold text-foreground">
                {formatCompactNumber(point.message_count)} mesaj
              </span>
              <span className="font-mono text-2xs uppercase text-muted-foreground">
                {formatShortDate(point.bucket_start)}
              </span>
            </div>
            <div
              aria-label={`${formatShortDate(point.bucket_start)} · ${formatCompactNumber(point.message_count)} mesaj`}
              className="rounded-sm bg-accent transition-opacity duration-100 group-hover:opacity-80"
              style={{ height: `${heightPct}%` }}
            />
          </div>
        );
      })}
    </div>
  );
}

function getInitial(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) {
    return "?";
  }
  return trimmed.charAt(0).toUpperCase();
}

type TopRow = {
  key: string;
  label: string;
  value: string;
  href?: string;
  image?: string;
  initial?: string;
};

function TopList({ rows, emptyText }: { rows: TopRow[]; emptyText: string }) {
  if (!rows.length) {
    return <EmptyHint text={emptyText} />;
  }

  return (
    <ol className="flex flex-col gap-2.5">
      {rows.map((row, index) => {
        const content = (
          <>
            <span className="w-6 shrink-0 font-mono text-2xs uppercase text-faint">
              {(index + 1).toString().padStart(2, "0")}
            </span>
            {row.image ? (
              <img
                alt={row.label}
                className="h-5 w-5 shrink-0 rounded-sm bg-elevated object-cover"
                src={row.image}
              />
            ) : row.initial ? (
              <span
                aria-hidden
                className="flex h-5 w-5 shrink-0 items-center justify-center rounded-sm bg-elevated font-mono text-[10px] font-semibold uppercase text-muted-foreground"
              >
                {row.initial}
              </span>
            ) : (
              <span aria-hidden className="h-5 w-5 shrink-0 rounded-sm bg-elevated" />
            )}
            <span className="flex-1 truncate text-[13px] text-foreground">{row.label}</span>
            <span className="shrink-0 font-mono text-[13px] text-muted-foreground">
              {row.value}
            </span>
          </>
        );

        return (
          <li key={row.key}>
            {row.href ? (
              <Link
                href={row.href}
                className={cn(
                  "flex items-center gap-3 rounded-sm px-1 py-1 -mx-1",
                  "transition-colors hover:bg-elevated"
                )}
              >
                {content}
              </Link>
            ) : (
              <div className="flex items-center gap-3 px-1 py-1 -mx-1">{content}</div>
            )}
          </li>
        );
      })}
    </ol>
  );
}

function StatusBanner({ status }: { status: "loading" | "ready" | "error" }) {
  if (status === "ready") {
    return null;
  }

  return (
    <div className="rounded-md border border-border bg-panel px-4 py-3 text-sm text-muted-foreground">
      {status === "loading"
        ? "Analytics verileri yükleniyor…"
        : "Analytics verileri şu anda alınamadı. Arama ve admin bağlantıları kullanılabilir."}
    </div>
  );
}

function EmptyHint({ text }: { text: string }) {
  return <p className="text-[13px] text-muted-foreground">{text}</p>;
}

function getRecentVolumeRange() {
  const end = new Date();
  const start = new Date(end);
  start.setDate(start.getDate() - 13);
  start.setHours(0, 0, 0, 0);

  return {
    end: end.toISOString(),
    start: start.toISOString()
  };
}

const COMPACT_FORMATTER = new Intl.NumberFormat("tr-TR", {
  notation: "compact",
  maximumFractionDigits: 1
});

function formatCompactNumber(value: number) {
  return COMPACT_FORMATTER.format(value);
}

function formatShortDate(value: string) {
  return new Intl.DateTimeFormat("tr-TR", {
    day: "2-digit",
    month: "short"
  }).format(new Date(value));
}
