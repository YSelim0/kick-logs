"use client";

/* eslint-disable @next/next/no-img-element */

import { Search } from "lucide-react";
import Link from "next/link";
import type { CSSProperties } from "react";
import { useEffect, useState } from "react";

import { SiteHeader } from "@/components/site-header";
import { Button } from "@/components/ui/button";
import { getChannelProfile } from "@/features/channel-profile/api";
import { MessageContent } from "@/features/search/message-content";
import { formatMessageDate } from "@/features/search/search-params";
import { ApiClientError } from "@/lib/api-client";
import { buildUserProfileHref } from "@/lib/kick-profile-slugs";
import type {
  ChannelProfile,
  Message,
  MessageVolumePoint,
  TopEmoteAnalytics,
  TopSenderAnalytics
} from "@/types/api";

type ProfileStatus = "loading" | "ready" | "not-found" | "error";

export function ChannelProfilePage({ slug }: { slug: string }) {
  const [profile, setProfile] = useState<ChannelProfile | null>(null);
  const [status, setStatus] = useState<ProfileStatus>("loading");

  useEffect(() => {
    let isMounted = true;

    async function loadProfile() {
      setStatus("loading");

      try {
        const nextProfile = await getChannelProfile(slug);
        if (!isMounted) {
          return;
        }
        setProfile(nextProfile);
        setStatus("ready");
      } catch (caught) {
        if (!isMounted) {
          return;
        }
        setProfile(null);
        setStatus(
          caught instanceof ApiClientError && caught.status === 404 ? "not-found" : "error"
        );
      }
    }

    void loadProfile();

    return () => {
      isMounted = false;
    };
  }, [slug]);

  return (
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute="channels" />

      <div className="mx-auto max-w-[1280px] px-6 py-6">
        <Breadcrumb slug={slug} />

        <div className="mt-5 space-y-5">
          {status === "loading" ? <ProfileState message="Kanal profili yükleniyor..." /> : null}
          {status === "not-found" ? (
            <ProfileState
              actionHref="/search"
              actionLabel="Search'e dön"
              message="Kanal bulunamadı."
              tone="warning"
            />
          ) : null}
          {status === "error" ? (
            <ProfileState
              actionHref={`/search?channel=${encodeURIComponent(slug)}`}
              actionLabel="Search'te ara"
              message="Kanal profili şu anda alınamadı."
              tone="danger"
            />
          ) : null}
          {status === "ready" && profile ? <ProfileContent profile={profile} /> : null}
        </div>
      </div>
    </main>
  );
}

function Breadcrumb({ slug }: { slug: string }) {
  return (
    <nav aria-label="Breadcrumb">
      <p className="font-mono text-[12px] uppercase tracking-wider text-muted-foreground">
        <Link className="hover:text-foreground" href="/">
          channels
        </Link>{" "}
        <span className="text-faint">/</span> <span className="text-foreground">{slug}</span>
      </p>
    </nav>
  );
}

function ProfileContent({ profile }: { profile: ChannelProfile }) {
  const searchHref = `/search?channel=${encodeURIComponent(profile.channel.slug)}`;

  return (
    <div className="space-y-5">
      {/* Identity panel */}
      <section className="rounded-lg border border-border bg-panel px-6 py-5">
        <div className="flex flex-wrap items-center justify-between gap-5">
          <div className="flex min-w-0 items-center gap-4">
            <ChannelAvatar profile={profile} />
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h1 className="text-[22px] font-semibold leading-none text-foreground">
                  {profile.channel.display_name}
                </h1>
                <LoggingPill />
              </div>
              <p className="mt-2 font-mono text-[11px] text-muted-foreground">
                channel id{" "}
                <span className="text-muted-foreground">
                  {profile.channel.kick_channel_id ?? "—"}
                </span>{" "}
                · chatroom id{" "}
                <span className="text-muted-foreground">
                  {profile.channel.kick_chatroom_id ?? "—"}
                </span>{" "}
                · ilk log{" "}
                <span className="text-muted-foreground">
                  {formatShortDate(profile.overview.first_message_at)}
                </span>{" "}
                · son aktivite{" "}
                <span className="text-muted-foreground">
                  {formatRelativeTime(profile.overview.latest_message_at)}
                </span>
              </p>
            </div>
          </div>

          <Button asChild>
            <Link href={searchHref}>
              <Search className="h-4 w-4" />
              Kanalda ara
            </Link>
          </Button>
        </div>
      </section>

      {/* Stats bar */}
      <ProfileStatsBar
        cells={[
          { label: "MESAJ", value: formatCompactNumber(profile.overview.total_messages) },
          { label: "KULLANICI", value: formatCompactNumber(profile.overview.total_senders) },
          { label: "EMOTE", value: formatCompactNumber(profile.overview.total_emote_usages) },
          { label: "İLK LOG", value: formatShortDate(profile.overview.first_message_at) }
        ]}
      />

      {/* 3-column analytics grid */}
      <section
        aria-label="Kanal analitiği"
        className="grid grid-cols-1 gap-5 lg:grid-cols-3 lg:[&>*]:min-h-0"
        style={{ alignItems: "stretch" }}
      >
        <AnalyticsPanel title="Mesaj hacmi" subtitle="son 14 gün">
          <VolumeChart points={profile.message_volume} />
        </AnalyticsPanel>

        <AnalyticsPanel title="Top kullanıcılar" subtitle="mesaj sayısı">
          <TopSenders senders={profile.top_senders} />
        </AnalyticsPanel>

        <AnalyticsPanel title="Top emoteler" subtitle="kullanım">
          <TopEmotes emotes={profile.top_emotes} />
        </AnalyticsPanel>
      </section>

      {/* Latest messages */}
      <section className="rounded-lg border border-border bg-panel p-5">
        <header className="mb-4 flex items-baseline justify-between gap-3">
          <div>
            <h2 className="text-[15px] font-semibold leading-none text-foreground">Son mesajlar</h2>
            <p className="mt-0.5 font-mono text-2xs uppercase text-muted-foreground">en son 20</p>
          </div>
          <Link
            className="font-mono text-[12px] text-accent hover:text-accent-hover"
            href={`/search?channel=${encodeURIComponent(profile.channel.slug)}`}
          >
            tümünü ara →
          </Link>
        </header>
        <LatestMessages messages={profile.latest_messages} />
      </section>
    </div>
  );
}

function LoggingPill() {
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-elevated px-2 py-0.5 font-mono text-[9px] font-semibold uppercase tracking-wider text-accent">
      <span aria-hidden className="h-1.5 w-1.5 rounded-full bg-accent" />
      LOGGING
    </span>
  );
}

function ChannelAvatar({ profile }: { profile: ChannelProfile }) {
  const [failed, setFailed] = useState(false);
  const imageUrl = profile.channel.profile_image_url;
  const initial = profile.channel.display_name.slice(0, 1).toUpperCase();

  if (imageUrl && !failed) {
    return (
      <img
        alt={`${profile.channel.display_name} kanal`}
        className="h-[72px] w-[72px] rounded-md border border-border object-cover"
        height={72}
        onError={() => setFailed(true)}
        src={imageUrl}
        width={72}
      />
    );
  }

  return (
    <div className="flex h-[72px] w-[72px] items-center justify-center rounded-md border border-border bg-elevated font-mono text-xl font-semibold text-muted-foreground">
      {initial || "#"}
    </div>
  );
}

function ProfileState({
  actionHref,
  actionLabel,
  message,
  tone = "default"
}: {
  actionHref?: string;
  actionLabel?: string;
  message: string;
  tone?: "default" | "warning" | "danger";
}) {
  const toneClass =
    tone === "danger"
      ? "text-danger"
      : tone === "warning"
        ? "text-warning"
        : "text-muted-foreground";

  return (
    <section className="rounded-lg border border-border bg-panel p-6">
      <p className={`text-sm font-medium ${toneClass}`}>{message}</p>
      <p className="mt-1 text-xs text-muted-foreground">
        Public kanal profilleri loglanan mesaj verileriyle oluşur.
      </p>
      {actionHref && actionLabel ? (
        <Button asChild className="mt-4" size="sm">
          <Link href={actionHref}>{actionLabel}</Link>
        </Button>
      ) : null}
    </section>
  );
}

type StatCell = { label: string; value: string };

function ProfileStatsBar({ cells }: { cells: StatCell[] }) {
  return (
    <section
      aria-label="Kanal metrikleri"
      className="grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border md:grid-cols-4"
    >
      {cells.map((cell) => (
        <div key={cell.label} className="bg-panel px-5 py-4">
          <div className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
            {cell.label}
          </div>
          <div className="mt-2 font-sans text-[24px] font-semibold leading-none tracking-tight text-foreground">
            {cell.value}
          </div>
        </div>
      ))}
    </section>
  );
}

function AnalyticsPanel({
  title,
  subtitle,
  children
}: {
  title: string;
  subtitle: string;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col rounded-lg border border-border bg-panel p-5">
      <header className="mb-4 flex flex-col gap-0.5">
        <h2 className="text-[14px] font-semibold leading-none text-foreground">{title}</h2>
        <p className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
          {subtitle}
        </p>
      </header>
      <div className="flex-1">{children}</div>
    </section>
  );
}

function VolumeChart({ points }: { points: MessageVolumePoint[] }) {
  if (points.length === 0) {
    return <SmallEmpty text="Mesaj hacmi verisi henüz yok." />;
  }

  const max = points.reduce((acc, p) => Math.max(acc, p.message_count), 0);

  return (
    <div className="relative flex h-36 items-end gap-1">
      {points.map((point) => {
        const ratio = max > 0 ? point.message_count / max : 0;
        const heightPct = max > 0 ? Math.max(ratio * 100, 4) : 4;
        return (
          <div
            key={point.bucket_start}
            className="group relative flex h-full flex-1 flex-col justify-end"
          >
            <div
              aria-hidden
              className="pointer-events-none absolute left-1/2 z-10 flex -translate-x-1/2 flex-col items-center gap-0.5 whitespace-nowrap rounded-md border border-border bg-elevated px-2 py-1.5 opacity-0 shadow-lg transition-opacity duration-100 group-hover:opacity-100"
              style={{ bottom: `calc(${heightPct}% + 8px)` }}
            >
              <span className="font-mono text-[11px] font-semibold text-foreground">
                {formatCompactNumber(point.message_count)}
              </span>
              <span className="font-mono text-[10px] uppercase text-muted-foreground">
                {formatShortDate(point.bucket_start)}
              </span>
            </div>
            <div
              aria-label={`${formatShortDate(point.bucket_start)} · ${formatCompactNumber(point.message_count)} mesaj`}
              className="rounded-t-sm bg-accent transition-opacity duration-100 group-hover:opacity-80"
              style={{ height: `${heightPct}%` }}
            />
          </div>
        );
      })}
    </div>
  );
}

function TopSenders({ senders }: { senders: TopSenderAnalytics[] }) {
  if (senders.length === 0) {
    return <SmallEmpty text="Kullanıcı aktivitesi henüz yok." />;
  }

  return (
    <ol className="flex flex-col gap-2">
      {senders.map((sender, index) => {
        const href = buildUserProfileHref(sender.slug);
        const inner = (
          <>
            <span className="w-5 shrink-0 font-mono text-[10px] text-faint">
              {(index + 1).toString().padStart(2, "0")}
            </span>
            <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-sm bg-elevated font-mono text-[10px] font-semibold uppercase text-muted-foreground">
              {sender.username.charAt(0).toUpperCase()}
            </span>
            <span className="flex-1 truncate text-[13px] text-foreground">@{sender.username}</span>
            <span className="shrink-0 font-mono text-[13px] text-muted-foreground">
              {formatCompactNumber(sender.message_count)}
            </span>
          </>
        );

        return (
          <li key={sender.sender_id}>
            {href ? (
              <Link
                className="flex items-center gap-2.5 rounded-sm px-1 py-1 -mx-1 transition-colors hover:bg-elevated"
                href={href}
              >
                {inner}
              </Link>
            ) : (
              <div className="flex items-center gap-2.5 px-1 py-1 -mx-1">{inner}</div>
            )}
          </li>
        );
      })}
    </ol>
  );
}

function TopEmotes({ emotes }: { emotes: TopEmoteAnalytics[] }) {
  if (emotes.length === 0) {
    return <SmallEmpty text="Emote verisi henüz yok." />;
  }

  return (
    <ol className="flex flex-col gap-2">
      {emotes.map((emote, index) => (
        <li className="flex items-center gap-2.5 px-1 py-1 -mx-1" key={emote.id}>
          <span className="w-5 shrink-0 font-mono text-[10px] text-faint">
            {(index + 1).toString().padStart(2, "0")}
          </span>
          <img
            alt={emote.name}
            className="h-5 w-5 shrink-0 rounded-sm object-contain"
            src={emote.image_url}
          />
          <span className="flex-1 truncate text-[13px] text-foreground">{emote.name}</span>
          <span className="shrink-0 font-mono text-[13px] text-muted-foreground">
            {formatCompactNumber(emote.usage_count)}
          </span>
        </li>
      ))}
    </ol>
  );
}

function LatestMessages({ messages }: { messages: Message[] }) {
  if (messages.length === 0) {
    return <SmallEmpty text="Son mesaj bulunamadı." />;
  }

  return (
    <div className="overflow-hidden rounded-md border border-border">
      {messages.map((message) => {
        const senderHref = buildUserProfileHref(message.sender.slug);
        const senderNameStyle = senderColorStyle(message.sender_color_snapshot);

        return (
          <div
            className="grid grid-cols-1 gap-2 border-b border-border px-3 py-2.5 text-[13px] last:border-b-0 md:grid-cols-[120px_minmax(0,1fr)_auto] md:items-start md:gap-4"
            key={message.id}
          >
            {/* sender username */}
            <div className="flex min-w-0 flex-col">
              {senderHref ? (
                <Link
                  className="truncate font-medium text-foreground hover:underline"
                  href={senderHref}
                  style={senderNameStyle}
                >
                  {message.sender.username}
                </Link>
              ) : (
                <span className="truncate font-medium text-foreground" style={senderNameStyle}>
                  {message.sender.username}
                </span>
              )}
            </div>

            {/* message content */}
            <div className="min-w-0 text-foreground">
              <MessageContent content={message.content} emotes={message.emotes} />
            </div>

            {/* timestamp */}
            <div className="text-right font-mono text-[11px] text-muted-foreground md:whitespace-nowrap">
              {formatMessageDate(message.message_created_at)}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function SmallEmpty({ text }: { text: string }) {
  return <p className="text-[13px] text-muted-foreground">{text}</p>;
}

function senderColorStyle(color: string | null): CSSProperties | undefined {
  if (!color || !/^#[0-9a-fA-F]{3,8}$/.test(color)) {
    return undefined;
  }
  return { color };
}

const COMPACT_FORMATTER = new Intl.NumberFormat("tr-TR", {
  notation: "compact",
  maximumFractionDigits: 1
});

function formatCompactNumber(value: number) {
  return COMPACT_FORMATTER.format(value);
}

function formatShortDate(value: string | null) {
  if (!value) return "—";
  return new Intl.DateTimeFormat("tr-TR", {
    day: "2-digit",
    month: "short",
    year: "numeric"
  }).format(new Date(value));
}

function formatRelativeTime(value: string | null) {
  if (!value) return "—";
  const diff = Date.now() - new Date(value).getTime();
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return `${seconds}s önce`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m önce`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}s önce`;
  const days = Math.floor(hours / 24);
  return `${days}g önce`;
}
