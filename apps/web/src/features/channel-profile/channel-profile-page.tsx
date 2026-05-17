"use client";

/* eslint-disable @next/next/no-img-element */

import {
  Activity,
  AlertTriangle,
  ArrowLeft,
  BarChart3,
  CalendarClock,
  MessageSquareText,
  Search,
  Smile,
  UserRound
} from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import type { CSSProperties } from "react";
import { useEffect, useMemo, useState } from "react";

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
    <main className="min-h-screen bg-background text-foreground">
      <ProfileHeader />

      <div className="mx-auto max-w-[1240px] px-4 py-6 md:px-8">
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
    </main>
  );
}

function ProfileHeader() {
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
        </div>
      </nav>
    </header>
  );
}

function ProfileContent({ profile }: { profile: ChannelProfile }) {
  const searchHref = `/search?channel=${encodeURIComponent(profile.channel.slug)}`;
  const volumeSummary = useMemo(
    () => summarizeVolume(profile.message_volume),
    [profile.message_volume]
  );

  return (
    <div className="space-y-6">
      <section className="rounded-lg border border-border bg-black p-5 md:p-6">
        <div className="flex flex-wrap items-start justify-between gap-5">
          <div className="flex min-w-0 items-center gap-4">
            <ChannelAvatar profile={profile} />
            <div className="min-w-0">
              <div className="mb-2 inline-flex items-center gap-2 rounded-md border border-border bg-kick-background px-3 py-1 text-xs text-primary">
                <Activity className="h-3.5 w-3.5" />
                Public channel profile
              </div>
              <h1 className="truncate text-2xl font-semibold md:text-3xl">
                {profile.channel.display_name}
              </h1>
              <p className="mt-1 text-sm text-muted-foreground">#{profile.channel.slug}</p>
              <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <span>Kick ID: {profile.channel.kick_channel_id ?? "-"}</span>
                <span>Chatroom: {profile.channel.kick_chatroom_id ?? "-"}</span>
                <span>İlk kayıt: {formatDateTime(profile.overview.first_message_at)}</span>
              </div>
            </div>
          </div>

          <div className="flex flex-wrap gap-3">
            <Button asChild>
              <Link href={searchHref}>
                <Search className="h-4 w-4" />
                Mesajlarda ara
              </Link>
            </Button>
            <Button asChild variant="outline">
              <Link href="/search">
                <ArrowLeft className="h-4 w-4 text-accent" />
                Search
              </Link>
            </Button>
          </div>
        </div>
      </section>

      <section aria-label="Kanal metrikleri" className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          icon={<MessageSquareText className="h-4 w-4" />}
          label="Toplam Mesaj"
          value={formatNumber(profile.overview.total_messages)}
        />
        <MetricCard
          icon={<UserRound className="h-4 w-4" />}
          label="Aktif Gönderen"
          value={formatNumber(profile.overview.total_senders)}
        />
        <MetricCard
          icon={<Smile className="h-4 w-4" />}
          label="Emote Kullanımı"
          value={formatNumber(profile.overview.total_emote_usages)}
        />
        <MetricCard
          icon={<CalendarClock className="h-4 w-4" />}
          label="Son Aktivite"
          value={formatDateTime(profile.overview.latest_message_at)}
          valueClassName="text-base"
        />
      </section>

      <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
        <div className="space-y-6">
          <Panel
            description="Gün bazında kanalın loglanan mesaj hacmi."
            icon={<BarChart3 className="h-4 w-4" />}
            title="Mesaj Hacmi"
          >
            <VolumeList points={profile.message_volume} summary={volumeSummary} />
          </Panel>

          <Panel
            description="Bu kanalda loglanan en yeni mesajlar."
            icon={<MessageSquareText className="h-4 w-4" />}
            title="Son Mesajlar"
          >
            <LatestMessages messages={profile.latest_messages} />
          </Panel>
        </div>

        <div className="space-y-6">
          <Panel
            description="Bu kanalda en çok mesaj atan kullanıcılar."
            icon={<UserRound className="h-4 w-4" />}
            title="Top Kullanıcılar"
          >
            <TopSenders senders={profile.top_senders} />
          </Panel>
          <Panel
            description="Bu kanalda mesajlarda en çok görünen emote'lar."
            icon={<Smile className="h-4 w-4" />}
            title="Top Emoteler"
          >
            <TopEmotes emotes={profile.top_emotes} />
          </Panel>
        </div>
      </section>
    </div>
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
        className="h-16 w-16 rounded-full border border-border object-cover md:h-20 md:w-20"
        height={80}
        onError={() => setFailed(true)}
        src={imageUrl}
        width={80}
      />
    );
  }

  return (
    <div className="flex h-16 w-16 items-center justify-center rounded-full border border-border bg-secondary text-xl font-semibold text-secondary-foreground md:h-20 md:w-20">
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
  return (
    <section className="rounded-lg border border-border bg-black p-6">
      <div className="flex items-center gap-3">
        <div className="flex h-9 w-9 items-center justify-center rounded-md bg-kick-background text-primary">
          {tone === "default" ? (
            <Activity className="h-4 w-4" />
          ) : (
            <AlertTriangle className="h-4 w-4 text-accent" />
          )}
        </div>
        <div>
          <div className="text-sm font-medium">{message}</div>
          <p className="mt-1 text-xs text-muted-foreground">
            Public kanal profilleri loglanan mesaj verileriyle oluşur.
          </p>
        </div>
      </div>
      {actionHref && actionLabel ? (
        <Button asChild className="mt-5" size="sm">
          <Link href={actionHref}>{actionLabel}</Link>
        </Button>
      ) : null}
    </section>
  );
}

function NavLink({ children, href }: { children: React.ReactNode; href: string }) {
  return (
    <Link
      className="inline-flex h-9 items-center gap-1.5 rounded-md px-3 text-muted-foreground transition-colors hover:bg-secondary/40 hover:text-foreground"
      href={href}
    >
      {children}
    </Link>
  );
}

function MetricCard({
  icon,
  label,
  value,
  valueClassName = "text-2xl"
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  valueClassName?: string;
}) {
  return (
    <article className="rounded-lg border border-border bg-black p-4">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <span className="text-accent">{icon}</span>
        {label}
      </div>
      <div className={`mt-3 font-semibold text-primary ${valueClassName}`}>{value}</div>
    </article>
  );
}

function Panel({
  children,
  description,
  icon,
  title
}: {
  children: React.ReactNode;
  description: string;
  icon: React.ReactNode;
  title: string;
}) {
  return (
    <section className="rounded-lg border border-border bg-black p-4">
      <div className="mb-4 flex items-start gap-3 border-b border-border pb-3">
        <div className="mt-0.5 flex h-8 w-8 items-center justify-center rounded-md bg-kick-background text-accent">
          {icon}
        </div>
        <div>
          <h2 className="text-base font-semibold">{title}</h2>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
      </div>
      {children}
    </section>
  );
}

function VolumeList({
  points,
  summary
}: {
  points: MessageVolumePoint[];
  summary: { max: number; total: number };
}) {
  if (points.length === 0) {
    return <SmallEmpty text="Mesaj hacmi verisi henüz yok." />;
  }

  return (
    <div className="space-y-3">
      <div className="text-sm text-muted-foreground">
        Toplam hacim: <span className="text-primary">{formatNumber(summary.total)}</span>
      </div>
      <div className="grid gap-2">
        {points.map((point) => {
          const width = Math.max((point.message_count / summary.max) * 100, 4);
          return (
            <div className="grid gap-1" key={point.bucket_start}>
              <div className="flex items-center justify-between gap-3 text-xs">
                <span className="text-muted-foreground">{formatDate(point.bucket_start)}</span>
                <span className="text-primary">{formatNumber(point.message_count)}</span>
              </div>
              <div className="h-2 overflow-hidden rounded-sm bg-kick-background">
                <div className="h-full bg-primary" style={{ width: `${width}%` }} />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function TopSenders({ senders }: { senders: TopSenderAnalytics[] }) {
  if (senders.length === 0) {
    return <SmallEmpty text="Kullanıcı aktivitesi henüz yok." />;
  }

  return (
    <div className="grid gap-2">
      {senders.map((sender) => {
        const href = buildUserProfileHref(sender.slug);
        const content = (
          <>
            <span className="truncate">@{sender.username}</span>
            <span className="shrink-0 text-primary">{formatNumber(sender.message_count)}</span>
          </>
        );

        return href ? (
          <Link
            className="flex items-center justify-between gap-3 rounded-md border border-border bg-kick-background px-3 py-2 text-sm hover:border-primary"
            href={href}
            key={sender.sender_id}
          >
            {content}
          </Link>
        ) : (
          <div
            className="flex items-center justify-between gap-3 rounded-md border border-border bg-kick-background px-3 py-2 text-sm"
            key={sender.sender_id}
          >
            {content}
          </div>
        );
      })}
    </div>
  );
}

function TopEmotes({ emotes }: { emotes: TopEmoteAnalytics[] }) {
  if (emotes.length === 0) {
    return <SmallEmpty text="Emote verisi henüz yok." />;
  }

  return (
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
  );
}

function LatestMessages({ messages }: { messages: Message[] }) {
  if (messages.length === 0) {
    return <SmallEmpty text="Son mesaj bulunamadı." />;
  }

  return (
    <div className="overflow-hidden rounded-md border border-border">
      {messages.map((message, index) => {
        const senderHref = buildUserProfileHref(message.sender.slug);
        const senderNameStyle = senderColorStyle(message.sender_color_snapshot);
        return (
          <div
            className={`grid gap-2 border-t border-border/70 px-3 py-3 text-sm first:border-t-0 ${
              index % 2 === 1 ? "bg-kick-background" : "bg-black"
            }`}
            key={message.id}
          >
            <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
              {senderHref ? (
                <Link
                  className="text-accent hover:text-primary"
                  href={senderHref}
                  style={senderNameStyle}
                >
                  @{message.sender.username}
                </Link>
              ) : (
                <span className="text-accent" style={senderNameStyle}>
                  @{message.sender.username}
                </span>
              )}
              <span>{formatMessageDate(message.message_created_at)}</span>
            </div>
            <MessageContent content={message.content} emotes={message.emotes} />
          </div>
        );
      })}
    </div>
  );
}

function SmallEmpty({ text }: { text: string }) {
  return <div className="text-sm text-muted-foreground">{text}</div>;
}

function senderColorStyle(color: string | null): CSSProperties | undefined {
  if (!color || !/^#[0-9a-fA-F]{3,8}$/.test(color)) {
    return undefined;
  }

  return { color };
}

function summarizeVolume(points: MessageVolumePoint[]) {
  return {
    max: Math.max(...points.map((point) => point.message_count), 1),
    total: points.reduce((sum, point) => sum + point.message_count, 0)
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
    month: "short",
    year: "numeric"
  }).format(new Date(value));
}
