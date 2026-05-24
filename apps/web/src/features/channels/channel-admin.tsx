"use client";

import Image from "next/image";
import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { addChannel, listChannels, removeChannel } from "@/features/channels/api";
import { buildChannelProfileHref } from "@/lib/channel-profile-slugs";
import type { Channel } from "@/types/api";

export function ChannelAdmin() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [slug, setSlug] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isAdding, setIsAdding] = useState(false);
  const [removingChannelId, setRemovingChannelId] = useState<number | null>(null);

  const loadChannels = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      setChannels(await listChannels());
    } catch (caught) {
      setError(resolveAdminError(caught));
      setChannels([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadChannels();
  }, [loadChannels]);

  async function submitChannel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const normalizedSlug = slug.trim();
    if (!normalizedSlug) return;

    setIsAdding(true);
    setError(null);

    try {
      const channel = await addChannel({ slug: normalizedSlug });
      setSlug("");
      setChannels((current) => mergeChannel(current, channel));
    } catch (caught) {
      setError(resolveAdminError(caught));
    } finally {
      setIsAdding(false);
    }
  }

  async function disableChannel(channelId: number) {
    setRemovingChannelId(channelId);
    setError(null);

    try {
      const channel = await removeChannel(channelId);
      setChannels((current) => mergeChannel(current, channel));
    } catch (caught) {
      setError(resolveAdminError(caught));
    } finally {
      setRemovingChannelId(null);
    }
  }

  const enabledCount = channels.filter((c) => c.is_enabled).length;

  return (
    <section className="rounded-lg border border-border bg-panel p-5">
      <div className="mb-5 flex items-center justify-between">
        <div className="flex flex-col gap-0.5">
          <span className="text-[14px] font-semibold text-foreground">Takip edilen kanallar</span>
          <span className="font-mono text-[11px] text-faint">{enabledCount} aktif</span>
        </div>

        <form
          className="flex flex-col items-stretch gap-2 sm:flex-row sm:items-center"
          onSubmit={submitChannel}
        >
          <label className="sr-only" htmlFor="channel-slug">
            Kanal slug/nickname
          </label>
          <div className="flex h-8 w-full items-center gap-1.5 rounded-md border border-border bg-elevated px-2.5 sm:w-60">
            <Plus className="h-3 w-3 shrink-0 text-faint" />
            <input
              className="flex-1 bg-transparent font-sans text-[12px] text-foreground outline-none placeholder:text-faint"
              disabled={isAdding}
              id="channel-slug"
              maxLength={120}
              onChange={(e) => setSlug(e.target.value)}
              placeholder="channel slug ekle"
              value={slug}
            />
          </div>
          <Button
            className="h-8 px-3.5 text-[12px]"
            disabled={isAdding || !slug.trim()}
            size="sm"
            type="submit"
          >
            {isAdding ? <Loader2 className="h-3 w-3 animate-spin" /> : null}
            Ekle
          </Button>
        </form>
      </div>

      {error ? (
        <div className="mb-4 rounded-md border border-danger bg-elevated px-3 py-2 text-[13px]">
          {error}
        </div>
      ) : null}

      <div className="rounded-lg border border-border">
        <div className="hidden items-center border-b border-border px-3 py-2 sm:flex">
          <span className="flex-1 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            KANAL
          </span>
          <span className="w-24 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            DURUM
          </span>
          <span className="w-28 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            MESAJ
          </span>
          <span className="w-40 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            SON AKTİVİTE
          </span>
          <span className="w-36" />
        </div>

        {isLoading ? (
          <div className="px-4 py-10 text-center text-[13px] text-muted-foreground">
            Kanallar yükleniyor...
          </div>
        ) : channels.length ? (
          channels.map((channel) => (
            <ChannelRow
              channel={channel}
              isRemoving={removingChannelId === channel.id}
              key={channel.id}
              onDisable={() => void disableChannel(channel.id)}
            />
          ))
        ) : (
          <div className="px-4 py-10 text-center text-[13px] text-muted-foreground">
            Henüz takip edilen kanal yok.
          </div>
        )}
      </div>
    </section>
  );
}

function ChannelRow({
  channel,
  isRemoving,
  onDisable
}: {
  channel: Channel;
  isRemoving: boolean;
  onDisable: () => void;
}) {
  const profileHref = buildChannelProfileHref(channel.slug);

  const nameContent = (
    <div className="min-w-0">
      <div className="truncate font-sans text-[13px] font-medium text-foreground">
        {channel.display_name}
      </div>
      <div className="font-mono text-[11px] text-accent">#{channel.slug}</div>
    </div>
  );

  const statusDot = (
    <span className={`h-1.5 w-1.5 rounded-full ${channel.is_enabled ? "bg-accent" : "bg-faint"}`} />
  );
  const disableButton = (
    <Button
      disabled={!channel.is_enabled || isRemoving}
      onClick={onDisable}
      size="sm"
      type="button"
      variant="outline"
    >
      {isRemoving ? (
        <Loader2 className="h-3 w-3 animate-spin" />
      ) : (
        <Trash2 className="h-3 w-3 text-danger" />
      )}
      <span className="hidden sm:inline">Devre dışı bırak</span>
    </Button>
  );

  return (
    <div className="border-b border-border px-3 py-2.5 last:border-b-0">
      {/* Desktop layout */}
      <div className="hidden items-center sm:flex">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <ChannelImage channel={channel} />
          {profileHref ? (
            <Link className="min-w-0 hover:opacity-80" href={profileHref}>
              {nameContent}
            </Link>
          ) : (
            nameContent
          )}
        </div>
        <div className="flex w-24 items-center gap-1.5">
          {statusDot}
          <span
            className={`font-mono text-[11px] ${channel.is_enabled ? "text-foreground" : "text-faint"}`}
          >
            {channel.is_enabled ? "Aktif" : "Pasif"}
          </span>
        </div>
        <div className="w-28">
          <span className="font-mono text-[12px] text-muted-foreground">
            {channel.message_count > 0 ? formatNumber(channel.message_count) : "—"}
          </span>
        </div>
        <div className="w-40">
          <span className="font-mono text-[11px] text-faint">
            {channel.last_message_at ? formatDate(channel.last_message_at) : "—"}
          </span>
        </div>
        <div className="flex w-36 justify-end">{disableButton}</div>
      </div>

      {/* Mobile card layout */}
      <div className="sm:hidden">
        <div className="flex items-center gap-2">
          <ChannelImage channel={channel} />
          <div className="min-w-0 flex-1">
            {profileHref ? (
              <Link className="min-w-0 hover:opacity-80" href={profileHref}>
                {nameContent}
              </Link>
            ) : (
              nameContent
            )}
          </div>
          <div className="flex items-center gap-1.5">
            {statusDot}
            <span
              className={`font-mono text-[11px] ${channel.is_enabled ? "text-foreground" : "text-faint"}`}
            >
              {channel.is_enabled ? "Aktif" : "Pasif"}
            </span>
          </div>
          {disableButton}
        </div>
        <div className="mt-1.5 pl-8 font-mono text-[11px] text-faint">
          {channel.message_count > 0 ? formatNumber(channel.message_count) : "—"} mesaj
          {channel.last_message_at ? ` · ${formatDate(channel.last_message_at)}` : ""}
        </div>
      </div>
    </div>
  );
}

function ChannelImage({ channel }: { channel: Channel }) {
  if (channel.profile_image_url) {
    return (
      <Image
        alt={channel.display_name}
        className="shrink-0 rounded object-cover"
        height={24}
        src={channel.profile_image_url}
        width={24}
      />
    );
  }
  return <div className="h-6 w-6 shrink-0 rounded bg-elevated" />;
}

function mergeChannel(current: Channel[], channel: Channel) {
  const next = current.filter((item) => item.id !== channel.id);
  return [...next, channel].sort((a, b) => a.slug.localeCompare(b.slug));
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("tr-TR").format(value);
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("tr-TR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}

function resolveAdminError(error: unknown) {
  if (error instanceof Error) return error.message;
  return "Kanal işlemi tamamlanırken hata oluştu.";
}
