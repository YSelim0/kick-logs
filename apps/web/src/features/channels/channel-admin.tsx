"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { Hash, Loader2, Plus, RefreshCcw, Signal, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { addChannel, listChannels, removeChannel } from "@/features/channels/api";
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
    if (!normalizedSlug) {
      return;
    }

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

  const enabledCount = channels.filter((channel) => channel.is_enabled).length;

  return (
    <section className="rounded-lg border border-border bg-black p-5">
      <div className="mb-5 flex flex-wrap items-start justify-between gap-4 border-b border-border pb-4">
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-md bg-kick-background text-primary">
            <Signal className="h-4 w-4" />
          </div>
          <div>
            <h2 className="text-base font-semibold">Takip Edilen Kanallar</h2>
            <p className="text-xs text-muted-foreground">
              Listener tarafından izlenecek Kick kanallarını yönetin
            </p>
          </div>
        </div>

        <div className="rounded-md bg-kick-background px-3 py-2 text-xs">
          <span className="text-muted-foreground">Aktif kanal</span>
          <span className="ml-2 font-semibold text-primary">{enabledCount}</span>
        </div>
      </div>

      <form className="mb-4 grid gap-3 md:grid-cols-[minmax(0,1fr)_160px]" onSubmit={submitChannel}>
        <div>
          <label
            className="mb-2 flex h-5 items-center gap-2 text-sm font-medium"
            htmlFor="channel-slug"
          >
            <Hash className="h-4 w-4 text-accent" />
            Kanal slug/nickname
          </label>
          <Input
            id="channel-slug"
            maxLength={120}
            onChange={(event) => setSlug(event.target.value)}
            placeholder="örn. hype"
            value={slug}
          />
        </div>

        <div className="flex items-end">
          <Button className="h-11 w-full" disabled={isAdding || !slug.trim()}>
            {isAdding ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
            {isAdding ? "Çözülüyor" : "Kanal ekle"}
          </Button>
        </div>
      </form>

      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="text-xs text-muted-foreground">
          Ekleme sırasında backend Kick metadata ve chatroom bilgisini çözer.
        </div>
        <Button
          disabled={isLoading}
          onClick={() => void loadChannels()}
          size="sm"
          type="button"
          variant="outline"
        >
          <RefreshCcw className="h-4 w-4 text-accent" />
          Yenile
        </Button>
      </div>

      {error ? (
        <div className="mb-4 rounded-md border border-accent bg-kick-background px-3 py-2 text-sm">
          {error}
        </div>
      ) : null}

      <div className="overflow-hidden rounded-md border border-border">
        <div className="hidden grid-cols-[minmax(180px,1fr)_140px_140px_110px_150px] bg-kick-background px-3 py-2 text-xs font-medium text-muted-foreground md:grid">
          <div>Kanal</div>
          <div>Kick ID</div>
          <div>Chatroom</div>
          <div>Durum</div>
          <div className="text-right">İşlem</div>
        </div>

        {isLoading ? (
          <div className="px-4 py-10 text-center text-sm text-muted-foreground">
            Kanallar yükleniyor...
          </div>
        ) : channels.length ? (
          channels.map((channel, index) => (
            <ChannelRow
              channel={channel}
              isAlt={index % 2 === 1}
              isRemoving={removingChannelId === channel.id}
              key={channel.id}
              onDisable={() => void disableChannel(channel.id)}
            />
          ))
        ) : (
          <div className="px-4 py-10 text-center text-sm text-muted-foreground">
            Henüz takip edilen kanal yok.
          </div>
        )}
      </div>
    </section>
  );
}

function ChannelRow({
  channel,
  isAlt,
  isRemoving,
  onDisable
}: {
  channel: Channel;
  isAlt: boolean;
  isRemoving: boolean;
  onDisable: () => void;
}) {
  return (
    <div
      className={
        isAlt
          ? "grid gap-3 border-t border-border/70 bg-kick-background px-3 py-3 text-sm md:grid-cols-[minmax(180px,1fr)_140px_140px_110px_150px] md:items-center"
          : "grid gap-3 border-t border-border/70 bg-black px-3 py-3 text-sm md:grid-cols-[minmax(180px,1fr)_140px_140px_110px_150px] md:items-center"
      }
    >
      <div className="min-w-0">
        <div className="truncate font-medium text-foreground">{channel.display_name}</div>
        <div className="truncate text-xs text-accent">#{channel.slug}</div>
      </div>
      <MetaValue label="Kick ID" value={channel.kick_channel_id?.toString() ?? "-"} />
      <MetaValue label="Chatroom" value={channel.kick_chatroom_id?.toString() ?? "-"} />
      <div>
        <span
          className={
            channel.is_enabled
              ? "inline-flex rounded-md bg-primary px-2 py-1 text-xs font-semibold text-primary-foreground"
              : "inline-flex rounded-md bg-kick-background px-2 py-1 text-xs text-muted-foreground"
          }
        >
          {channel.is_enabled ? "Aktif" : "Pasif"}
        </span>
      </div>
      <div className="flex justify-start md:justify-end">
        <Button
          disabled={!channel.is_enabled || isRemoving}
          onClick={onDisable}
          size="sm"
          type="button"
          variant="outline"
        >
          {isRemoving ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Trash2 className="h-4 w-4 text-accent" />
          )}
          Devre dışı bırak
        </Button>
      </div>
    </div>
  );
}

function MetaValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground md:hidden">{label}</div>
      <div className="truncate text-foreground">{value}</div>
    </div>
  );
}

function mergeChannel(current: Channel[], channel: Channel) {
  const next = current.filter((item) => item.id !== channel.id);
  return [...next, channel].sort((first, second) => first.slug.localeCompare(second.slug));
}

function resolveAdminError(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }

  return "Kanal işlemi tamamlanırken hata oluştu.";
}
