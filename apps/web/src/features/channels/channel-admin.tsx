"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";
import Link from "next/link";

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

        <form className="flex items-center gap-2" onSubmit={submitChannel}>
          <label className="sr-only" htmlFor="channel-slug">
            Kanal slug/nickname
          </label>
          <div className="flex h-8 w-60 items-center gap-1.5 rounded-md border border-border bg-elevated px-2.5">
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
        <div className="flex items-center border-b border-border px-3 py-2">
          <span className="flex-1 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            KANAL
          </span>
          <span className="w-24 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            DURUM
          </span>
          <span className="w-24 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            MESAJ
          </span>
          <span className="w-36 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            SON AKTİVİTE
          </span>
          <span className="w-14" />
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

  return (
    <div className="flex items-center border-b border-border px-3 py-2.5 last:border-b-0">
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <div className="h-6 w-6 shrink-0 rounded bg-elevated" />
        {profileHref ? (
          <Link className="min-w-0 hover:opacity-80" href={profileHref}>
            <div className="truncate font-sans text-[13px] font-medium text-foreground">
              {channel.display_name}
            </div>
            <div className="font-mono text-[11px] text-accent">#{channel.slug}</div>
          </Link>
        ) : (
          <div className="min-w-0">
            <div className="truncate font-sans text-[13px] font-medium text-foreground">
              {channel.display_name}
            </div>
            <div className="font-mono text-[11px] text-accent">#{channel.slug}</div>
          </div>
        )}
      </div>

      <div className="flex w-24 items-center gap-1.5">
        <span
          className={`h-1.5 w-1.5 rounded-full ${channel.is_enabled ? "bg-accent" : "bg-faint"}`}
        />
        <span
          className={`font-mono text-[11px] ${channel.is_enabled ? "text-foreground" : "text-faint"}`}
        >
          {channel.is_enabled ? "Aktif" : "Pasif"}
        </span>
      </div>

      <div className="w-24">
        <span className="font-mono text-[12px] text-muted-foreground">—</span>
      </div>

      <div className="w-36">
        <span className="font-mono text-[11px] text-faint">—</span>
      </div>

      <div className="flex w-14 justify-end">
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
          Devre dışı bırak
        </Button>
      </div>
    </div>
  );
}

function mergeChannel(current: Channel[], channel: Channel) {
  const next = current.filter((item) => item.id !== channel.id);
  return [...next, channel].sort((a, b) => a.slug.localeCompare(b.slug));
}

function resolveAdminError(error: unknown) {
  if (error instanceof Error) return error.message;
  return "Kanal işlemi tamamlanırken hata oluştu.";
}
