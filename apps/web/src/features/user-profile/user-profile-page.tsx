"use client";

/* eslint-disable @next/next/no-img-element */

import { Search } from "lucide-react";
import { useTranslations } from "next-intl";
import { useUiFormat } from "@/i18n/use-ui-format";
import Link from "next/link";
import { useEffect, useState } from "react";

import { KickProfileLink } from "@/components/kick-profile-link";
import { ProfileLoading } from "@/components/profile-loading";
import { SiteHeader } from "@/components/site-header";
import { Button } from "@/components/ui/button";
import { getUserProfile } from "@/features/user-profile/api";
import { MessageContent } from "@/features/search/message-content";
import { getReplyContext } from "@/features/search/reply-metadata";
import { ApiClientError } from "@/lib/api-client";
import { buildKickProfileUrl, buildUserProfileHref } from "@/lib/kick-profile-slugs";
import type {
  Message,
  MessageVolumePoint,
  TopChannelAnalytics,
  TopEmoteAnalytics,
  UserProfile
} from "@/types/api";

type ProfileStatus = "loading" | "ready" | "not-found" | "error";

export function UserProfilePage({ slug }: { slug: string }) {
  const t = useTranslations("profiles");
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [status, setStatus] = useState<ProfileStatus>("loading");

  useEffect(() => {
    let isMounted = true;

    async function loadProfile() {
      setStatus("loading");

      try {
        const nextProfile = await getUserProfile(slug);
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
      <SiteHeader activeRoute="users" />

      <div className="mx-auto max-w-[1280px] px-6 py-6">
        <Breadcrumb slug={slug} />

        <div className="mt-5 space-y-5">
          {status === "loading" ? <ProfileLoading kind="user" /> : null}
          {status === "not-found" ? (
            <ProfileState
              actionHref="/search"
              actionLabel={t("backSearch")}
              message={t("userMissing")}
              tone="warning"
            />
          ) : null}
          {status === "error" ? (
            <ProfileState
              actionHref={`/search?sender=${encodeURIComponent(slug)}`}
              actionLabel={t("searchAction")}
              message={t("userError")}
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
  const t = useTranslations("profiles");
  return (
    <nav aria-label={t("breadcrumb")}>
      <p className="font-mono text-[12px] uppercase tracking-wider text-muted-foreground">
        <Link className="hover:text-foreground" href="/">
          {t("users")}
        </Link>{" "}
        <span className="text-faint">/</span> <span className="text-foreground">{slug}</span>
      </p>
    </nav>
  );
}

function ProfileContent({ profile }: { profile: UserProfile }) {
  const t = useTranslations("profiles");
  const format = useUiFormat();
  const searchHref = `/search?sender=${encodeURIComponent(profile.sender.slug)}`;
  const kickProfileUrl = buildKickProfileUrl(profile.sender.slug);

  return (
    <div className="space-y-5">
      {/* Identity panel */}
      <section className="rounded-lg border border-border bg-panel px-6 py-5">
        <div className="flex flex-wrap items-center justify-between gap-5">
          <div className="flex min-w-0 items-center gap-4">
            <ProfileAvatar profile={profile} />
            <div className="min-w-0">
              <h1 className="text-[22px] font-semibold leading-none text-foreground">
                {profile.sender.username}
              </h1>
              <p className="mt-1 font-mono text-[12px] text-muted-foreground">
                @{profile.sender.slug}
              </p>
              <p className="mt-2 font-mono text-[11px] text-muted-foreground">
                {t("firstMessage")}{" "}
                <span className="text-muted-foreground">
                  {format.shortDate(profile.overview.first_message_at)}
                </span>{" "}
                · {t("lastActivity")}{" "}
                <span className="text-muted-foreground">
                  {format.relativeTime(profile.overview.latest_message_at)}
                </span>
              </p>
            </div>
          </div>

          <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto">
            <KickProfileLink href={kickProfileUrl} />
            <Button asChild className="w-full sm:w-auto">
              <Link href={searchHref}>
                <Search className="h-4 w-4" />
                {t("searchUser")}
              </Link>
            </Button>
          </div>
        </div>
      </section>

      {/* Stats bar */}
      <ProfileStatsBar
        cells={[
          { label: t("messages"), value: format.compact(profile.overview.total_messages) },
          { label: t("channelCount"), value: format.number(profile.overview.total_channels) },
          { label: t("emotes"), value: format.compact(profile.overview.total_emote_usages) },
          {
            label: t("firstMessageStat"),
            value: format.shortDate(profile.overview.first_message_at)
          }
        ]}
      />

      {/* 3-column analytics grid */}
      <section
        aria-label={t("userAnalytics")}
        className="grid grid-cols-1 gap-5 lg:grid-cols-3 lg:[&>*]:min-h-0"
        style={{ alignItems: "stretch" }}
      >
        <AnalyticsPanel title={t("volume")} subtitle={t("period")}>
          <VolumeChart points={profile.message_volume} />
        </AnalyticsPanel>

        <AnalyticsPanel title={t("topChannels")} subtitle={t("messageCount")}>
          <TopChannels channels={profile.top_channels} senderSlug={profile.sender.slug} />
        </AnalyticsPanel>

        <AnalyticsPanel title={t("topEmotes")} subtitle={t("usage")}>
          <TopEmotes emotes={profile.top_emotes} />
        </AnalyticsPanel>
      </section>

      {/* Latest messages */}
      <section className="rounded-lg border border-border bg-panel p-5">
        <header className="mb-4 flex items-baseline justify-between gap-3">
          <div>
            <h2 className="text-[15px] font-semibold leading-none text-foreground">
              {t("latest")}
            </h2>
            <p className="mt-0.5 font-mono text-2xs uppercase text-muted-foreground">
              {t("latestLimit")}
            </p>
          </div>
          <Link
            className="font-mono text-[12px] text-accent hover:text-accent-hover"
            href={`/search?sender=${encodeURIComponent(profile.sender.slug)}`}
          >
            {t("searchAll")}
          </Link>
        </header>
        <LatestMessages messages={profile.latest_messages} />
      </section>
    </div>
  );
}

function ProfileAvatar({ profile }: { profile: UserProfile }) {
  const [failed, setFailed] = useState(false);
  const imageUrl = profile.sender.profile_image_url;
  const initial = profile.sender.username.slice(0, 1).toUpperCase();

  if (imageUrl && !failed) {
    return (
      <img
        alt={profile.sender.username}
        className="h-[72px] w-[72px] min-w-[72px] rounded-full border border-border object-cover"
        height={72}
        onError={() => setFailed(true)}
        src={imageUrl}
        width={72}
      />
    );
  }

  return (
    <div className="flex h-[72px] w-[72px] min-w-[72px] items-center justify-center rounded-full border border-border bg-elevated font-mono text-xl font-semibold text-muted-foreground">
      {initial}
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
  const t = useTranslations("profiles");
  const toneClass =
    tone === "danger"
      ? "text-danger"
      : tone === "warning"
        ? "text-warning"
        : "text-muted-foreground";

  return (
    <section className="rounded-lg border border-border bg-panel p-6">
      <p className={`text-sm font-medium ${toneClass}`}>{message}</p>
      <p className="mt-1 text-xs text-muted-foreground">{t("userDescription")}</p>
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
  const t = useTranslations("profiles");
  return (
    <section
      aria-label={t("userMetrics")}
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
  const t = useTranslations("profiles");
  const format = useUiFormat();
  if (points.length === 0) {
    return <SmallEmpty text={t("emptyVolume")} />;
  }

  const max = points.reduce((acc, p) => Math.max(acc, p.message_count), 0);

  return (
    <div className="relative flex h-36 items-end gap-1">
      {points.map((point) => {
        const ratio = max > 0 ? point.message_count / max : 0;
        const heightPct = max > 0 && point.message_count > 0 ? Math.max(ratio * 100, 4) : 2;
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
                {format.compact(point.message_count)}
              </span>
              <span className="font-mono text-[10px] uppercase text-muted-foreground">
                {format.dayLabel(point.bucket_start)}
              </span>
            </div>
            <div
              aria-label={t("volumeLabel", {
                date: format.dayLabel(point.bucket_start),
                count: point.message_count,
                value: format.compact(point.message_count)
              })}
              className="rounded-t-sm bg-accent transition-opacity duration-100 group-hover:opacity-80"
              style={{ height: `${heightPct}%` }}
            />
          </div>
        );
      })}
    </div>
  );
}

function TopChannels({
  channels,
  senderSlug
}: {
  channels: TopChannelAnalytics[];
  senderSlug: string;
}) {
  const t = useTranslations("profiles");
  const format = useUiFormat();
  if (channels.length === 0) {
    return <SmallEmpty text={t("emptyChannels")} />;
  }

  return (
    <ol className="flex flex-col gap-2">
      {channels.map((channel, index) => (
        <li key={channel.channel_id}>
          <Link
            className="flex items-center gap-2.5 rounded-sm px-1 py-1 -mx-1 transition-colors hover:bg-elevated"
            href={`/search?sender=${encodeURIComponent(senderSlug)}&channel=${encodeURIComponent(channel.slug)}`}
          >
            <span className="w-5 shrink-0 font-mono text-[10px] text-faint">
              {(index + 1).toString().padStart(2, "0")}
            </span>
            {channel.profile_image_url ? (
              <img
                alt={channel.display_name}
                className="h-5 w-5 shrink-0 rounded-sm bg-elevated object-cover"
                src={channel.profile_image_url}
              />
            ) : (
              <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-sm bg-elevated font-mono text-[10px] font-semibold uppercase text-muted-foreground">
                {channel.display_name.charAt(0).toUpperCase()}
              </span>
            )}
            <span className="flex-1 truncate text-[13px] text-foreground">{channel.slug}</span>
            <span className="shrink-0 font-mono text-[13px] text-muted-foreground">
              {format.compact(channel.message_count)}
            </span>
          </Link>
        </li>
      ))}
    </ol>
  );
}

function TopEmotes({ emotes }: { emotes: TopEmoteAnalytics[] }) {
  const t = useTranslations("profiles");
  const format = useUiFormat();
  if (emotes.length === 0) {
    return <SmallEmpty text={t("emptyEmotes")} />;
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
            {format.compact(emote.usage_count)}
          </span>
        </li>
      ))}
    </ol>
  );
}

function LatestMessages({ messages }: { messages: Message[] }) {
  const t = useTranslations("profiles");
  const format = useUiFormat();
  if (messages.length === 0) {
    return <SmallEmpty text={t("emptyMessages")} />;
  }

  return (
    <div className="overflow-hidden rounded-md border border-border">
      {messages.map((message) => {
        const replyContext = getReplyContext(message);
        const replyTitle = replyContext
          ? `@${replyContext.senderUsername}: ${replyContext.content}`
          : undefined;
        const replySenderProfileHref = buildUserProfileHref(replyContext?.senderSlug);

        return (
          <div
            className="grid grid-cols-1 gap-2 border-b border-border px-3 py-2.5 text-[13px] last:border-b-0 md:grid-cols-[120px_minmax(0,1fr)_auto] md:items-start md:gap-4"
            key={message.id}
          >
            {/* channel label */}
            <div className="flex min-w-0 flex-col">
              <Link
                className="truncate font-mono text-[11px] text-accent hover:text-accent-hover"
                href={`/channels/${encodeURIComponent(message.channel.slug)}`}
              >
                #{message.channel.slug}
              </Link>
            </div>

            {/* message content */}
            <div className="min-w-0 text-foreground">
              {replyContext ? (
                <div
                  className="mb-1 flex w-fit max-w-full items-center gap-1.5 rounded-md bg-elevated px-2 py-1 text-[12px] text-muted-foreground"
                  title={replyTitle}
                >
                  <span className="text-faint">↳</span>
                  {replySenderProfileHref ? (
                    <Link
                      className="font-medium text-foreground/80 hover:underline"
                      href={replySenderProfileHref}
                    >
                      @{replyContext.senderUsername}:
                    </Link>
                  ) : (
                    <span className="font-medium text-foreground/80">
                      @{replyContext.senderUsername}:
                    </span>
                  )}{" "}
                  <span className="truncate">{replyContext.content}</span>
                </div>
              ) : null}
              <MessageContent content={message.content} emotes={message.emotes} />
            </div>

            {/* timestamp */}
            <div className="text-right font-mono text-[11px] text-muted-foreground md:whitespace-nowrap">
              {format.dateTime(message.message_created_at)}
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
