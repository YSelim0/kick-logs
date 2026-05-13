"use client";

/* eslint-disable @next/next/no-img-element */

import {
  Activity,
  BarChart3,
  Coffee,
  Database,
  Github,
  Hash,
  MessageSquareText,
  Search,
  Shield,
  Smile,
  Users
} from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

import {
  getAnalyticsOverview,
  getMessageVolume,
  getTopChannels,
  getTopEmotes,
  getTopSenders
} from "@/features/analytics/api";
import { Button } from "@/components/ui/button";
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
        const recentRange = getRecentActivityRange();
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
  const hasMessages = overview.total_messages > 0;
  const recentActivity = useMemo(
    () => summarizeRecentActivity(analytics?.volume ?? []),
    [analytics]
  );

  return (
    <main className="min-h-screen bg-background text-foreground">
      <LandingHeader />

      <section className="border-b border-border bg-black">
        <div className="mx-auto grid max-w-[1240px] gap-8 px-4 py-8 md:px-8 lg:grid-cols-[minmax(0,1fr)_360px] lg:items-center">
          <div className="max-w-3xl">
            <div className="mb-4 inline-flex items-center gap-2 rounded-md border border-border bg-kick-background px-3 py-1.5 text-xs text-primary">
              <Database className="h-3.5 w-3.5" />
              Self-hosted Kick chat archive
            </div>
            <h1 className="text-2xl font-semibold tracking-normal md:text-3xl">Kick Logs</h1>
            <p className="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground md:text-base">
              Kendi sunucunda çalışan, takip ettiğin Kick kanallarındaki sohbet mesajlarını
              arşivleyen ve geçmiş mesajlarda hızlı arama yapmanı sağlayan açık kaynak log sistemi.
            </p>

            <div className="mt-5 flex flex-wrap gap-3">
              <Button asChild>
                <Link href="/search">
                  <Search className="h-4 w-4" />
                  Mesajlarda ara
                </Link>
              </Button>
              <Button asChild variant="outline">
                <Link href="/admin">
                  <Shield className="h-4 w-4 text-accent" />
                  Admin
                </Link>
              </Button>
              <Button asChild variant="ghost">
                <a
                  href="https://github.com/YSelim0/kick-logs"
                  rel="noopener noreferrer"
                  target="_blank"
                >
                  <Github className="h-4 w-4 text-accent" />
                  GitHub
                </a>
              </Button>
            </div>
          </div>

          <div className="flex items-center gap-4 border-t border-border pt-5 lg:border-l lg:border-t-0 lg:pl-6 lg:pt-0">
            <Image
              alt="Kick Logs"
              className="h-20 w-20 rounded-lg object-contain"
              height={80}
              priority
              src="/app-logo.png"
              width={80}
            />
            <div className="min-w-0">
              <div className="text-sm text-muted-foreground">Canlı özet</div>
              <div className="mt-1 text-xl font-semibold">
                {formatNumber(overview.total_messages)}
              </div>
              <div className="text-xs text-muted-foreground">loglanmış mesaj</div>
              <div className="mt-3 text-xs text-muted-foreground">
                Son aktivite: {formatDateTime(overview.latest_message_at)}
              </div>
            </div>
          </div>
        </div>
      </section>

      <div className="mx-auto max-w-[1240px] px-4 py-6 md:px-8">
        <AnalyticsStatus status={status} />

        <section aria-label="Genel metrikler" className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <MetricCard
            icon={<MessageSquareText className="h-4 w-4" />}
            label="Loglanan Mesaj"
            value={formatNumber(overview.total_messages)}
          />
          <MetricCard
            icon={<Hash className="h-4 w-4" />}
            label="Takip Edilen Kanal"
            value={formatNumber(overview.total_channels)}
          />
          <MetricCard
            icon={<Users className="h-4 w-4" />}
            label="Aktif Kullanıcı"
            value={formatNumber(overview.total_senders)}
          />
          <MetricCard
            icon={<Smile className="h-4 w-4" />}
            label="Emote Kullanımı"
            value={formatNumber(overview.total_emote_usages)}
          />
        </section>

        <section className="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
          <div>
            <SectionHeading
              description="Gün bazında mesaj hacmi ve en hareketli kanallar."
              icon={<Activity className="h-4 w-4" />}
              title="Sohbet Aktivitesi"
            />
            {hasMessages ? (
              <div className="mt-3 grid gap-3 md:grid-cols-2">
                <ActivitySummary summary={recentActivity} />
                <TopChannelsList channels={analytics?.topChannels ?? []} />
              </div>
            ) : (
              <EmptyState />
            )}
          </div>

          <div>
            <SectionHeading
              description="Mesajlarda en çok görünen emote ve kullanıcılar."
              icon={<BarChart3 className="h-4 w-4" />}
              title="Öne Çıkanlar"
            />
            <div className="mt-3 grid gap-3">
              <TopEmotesList emotes={analytics?.topEmotes ?? []} />
              <TopSendersList senders={analytics?.topSenders ?? []} />
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}

function LandingHeader() {
  return (
    <header className="border-b border-border bg-kick-background">
      <nav className="mx-auto flex max-w-[1240px] flex-wrap items-center justify-between gap-3 px-4 py-3 text-sm md:px-8">
        <Link className="flex items-center gap-3 font-semibold" href="/">
          <Image
            alt="Kick Logs"
            className="h-8 w-8 rounded-md object-contain"
            height={32}
            src="/app-logo.png"
            width={32}
          />
          Kick Logs
        </Link>

        <div className="flex flex-wrap items-center gap-2">
          <NavLink href="/search">Search</NavLink>
          <NavLink href="/admin">Admin</NavLink>
          <NavLink href="https://github.com/YSelim0/kick-logs">GitHub</NavLink>
          <NavLink href="https://buymeacoffee.com/yavuzselim">
            <Coffee className="h-3.5 w-3.5" />
            Support
          </NavLink>
        </div>
      </nav>
    </header>
  );
}

function NavLink({ children, href }: { children: React.ReactNode; href: string }) {
  const isExternal = href.startsWith("http");

  if (isExternal) {
    return (
      <a
        className="inline-flex h-9 items-center gap-1.5 rounded-md px-3 text-muted-foreground transition-colors hover:bg-secondary/40 hover:text-foreground"
        href={href}
        rel="noopener noreferrer"
        target="_blank"
      >
        {children}
      </a>
    );
  }

  return (
    <Link
      className="inline-flex h-9 items-center gap-1.5 rounded-md px-3 text-muted-foreground transition-colors hover:bg-secondary/40 hover:text-foreground"
      href={href}
    >
      {children}
    </Link>
  );
}

function AnalyticsStatus({ status }: { status: "loading" | "ready" | "error" }) {
  if (status === "ready") {
    return null;
  }

  return (
    <div className="mb-4 rounded-md border border-border bg-black px-4 py-3 text-sm text-muted-foreground">
      {status === "loading"
        ? "Analytics verileri yükleniyor..."
        : "Analytics verileri şu anda alınamadı. Arama ve admin bağlantıları kullanılabilir."}
    </div>
  );
}

function MetricCard({
  icon,
  label,
  value
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
}) {
  return (
    <article className="rounded-lg border border-border bg-black p-4">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <span className="text-accent">{icon}</span>
        {label}
      </div>
      <div className="mt-3 text-2xl font-semibold text-primary">{value}</div>
    </article>
  );
}

function SectionHeading({
  description,
  icon,
  title
}: {
  description: string;
  icon: React.ReactNode;
  title: string;
}) {
  return (
    <div className="flex items-start gap-3">
      <div className="mt-0.5 flex h-8 w-8 items-center justify-center rounded-md border border-border bg-black text-accent">
        {icon}
      </div>
      <div>
        <h2 className="text-base font-semibold">{title}</h2>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}

function ActivitySummary({ summary }: { summary: { total: number; latestLabel: string } }) {
  return (
    <article className="rounded-lg border border-border bg-black p-4">
      <div className="text-sm text-muted-foreground">Son hacim</div>
      <div className="mt-2 text-2xl font-semibold text-primary">{formatNumber(summary.total)}</div>
      <div className="mt-1 text-xs text-muted-foreground">Son bucket: {summary.latestLabel}</div>
    </article>
  );
}

function TopChannelsList({ channels }: { channels: TopChannelAnalytics[] }) {
  return (
    <article className="rounded-lg border border-border bg-black p-4">
      <div className="mb-3 text-sm font-medium">Top Kanallar</div>
      {channels.length ? (
        <div className="grid gap-2">
          {channels.map((channel) => (
            <Link
              className="flex items-center justify-between gap-3 rounded-md border border-border bg-kick-background px-3 py-2 text-sm hover:border-primary"
              href={`/search?channel=${encodeURIComponent(channel.slug)}`}
              key={channel.channel_id}
            >
              <span className="truncate">{channel.display_name}</span>
              <span className="shrink-0 text-primary">{formatNumber(channel.message_count)}</span>
            </Link>
          ))}
        </div>
      ) : (
        <SmallEmpty text="Kanal aktivitesi henüz yok." />
      )}
    </article>
  );
}

function TopEmotesList({ emotes }: { emotes: TopEmoteAnalytics[] }) {
  return (
    <article className="rounded-lg border border-border bg-black p-4">
      <div className="mb-3 text-sm font-medium">Top Emoteler</div>
      {emotes.length ? (
        <div className="grid gap-2">
          {emotes.map((emote) => (
            <div
              className="flex items-center justify-between gap-3 rounded-md border border-border bg-kick-background px-3 py-2 text-sm"
              key={emote.id}
            >
              <span className="flex min-w-0 items-center gap-2">
                <img
                  alt={emote.name}
                  className="h-5 w-5 shrink-0 object-contain"
                  src={emote.image_url}
                />
                <span className="truncate">{emote.name}</span>
              </span>
              <span className="shrink-0 text-primary">{formatNumber(emote.usage_count)}</span>
            </div>
          ))}
        </div>
      ) : (
        <SmallEmpty text="Emote verisi henüz yok." />
      )}
    </article>
  );
}

function TopSendersList({ senders }: { senders: TopSenderAnalytics[] }) {
  return (
    <article className="rounded-lg border border-border bg-black p-4">
      <div className="mb-3 text-sm font-medium">Aktif Kullanıcılar</div>
      {senders.length ? (
        <div className="grid gap-2">
          {senders.map((sender) => (
            <Link
              className="flex items-center justify-between gap-3 rounded-md border border-border bg-kick-background px-3 py-2 text-sm hover:border-primary"
              href={`/search?sender=${encodeURIComponent(sender.slug)}`}
              key={sender.sender_id}
            >
              <span className="truncate">{sender.username}</span>
              <span className="shrink-0 text-primary">{formatNumber(sender.message_count)}</span>
            </Link>
          ))}
        </div>
      ) : (
        <SmallEmpty text="Kullanıcı aktivitesi henüz yok." />
      )}
    </article>
  );
}

function EmptyState() {
  return (
    <div className="mt-3 rounded-lg border border-border bg-black p-5">
      <div className="text-sm font-medium">Henüz log yok</div>
      <p className="mt-2 text-sm leading-6 text-muted-foreground">
        Bir kanal ekleyip listener çalışmaya başladığında bu alan canlı analytics verileriyle
        dolacak. Bu sırada search ekranına veya admin paneline geçebilirsin.
      </p>
      <div className="mt-4 flex flex-wrap gap-3">
        <Button asChild size="sm">
          <Link href="/search">Search</Link>
        </Button>
        <Button asChild size="sm" variant="outline">
          <Link href="/admin">Admin</Link>
        </Button>
      </div>
    </div>
  );
}

function SmallEmpty({ text }: { text: string }) {
  return <div className="text-sm text-muted-foreground">{text}</div>;
}

function summarizeRecentActivity(points: MessageVolumePoint[]) {
  const total = points.reduce((sum, point) => sum + point.message_count, 0);
  const latest = points.at(-1);

  return {
    total,
    latestLabel: latest ? formatDate(latest.bucket_start) : "Veri yok"
  };
}

function getRecentActivityRange() {
  const end = new Date();
  const start = new Date(end);
  start.setDate(start.getDate() - 7);

  return {
    end: end.toISOString(),
    start: start.toISOString()
  };
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("tr-TR").format(value);
}

function formatDateTime(value: string | null) {
  if (!value) {
    return "Henüz yok";
  }

  return new Intl.DateTimeFormat("tr-TR", {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(new Date(value));
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("tr-TR", {
    day: "2-digit",
    month: "short"
  }).format(new Date(value));
}
