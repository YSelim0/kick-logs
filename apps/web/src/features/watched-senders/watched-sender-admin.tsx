"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { Loader2, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  addWatchedSender,
  getNotificationSettings,
  listWatchedSenders,
  removeWatchedSender,
  updateNotificationSettings
} from "@/features/watched-senders/api";
import type { WatchedSender } from "@/types/api";

const MIN_COOLDOWN_MINUTES = 1;
const MAX_COOLDOWN_MINUTES = 1440;

export function WatchedSenderAdmin() {
  const [senders, setSenders] = useState<WatchedSender[]>([]);
  const [username, setUsername] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isAdding, setIsAdding] = useState(false);
  const [removingSenderId, setRemovingSenderId] = useState<number | null>(null);

  const loadSenders = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      setSenders(await listWatchedSenders());
    } catch (caught) {
      setError(resolveAdminError(caught));
      setSenders([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadSenders();
  }, [loadSenders]);

  async function submitUsername(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const normalizedUsername = username.trim();
    if (!normalizedUsername) return;

    setIsAdding(true);
    setError(null);

    try {
      const sender = await addWatchedSender({ username: normalizedUsername });
      setUsername("");
      setSenders((current) => mergeSender(current, sender));
    } catch (caught) {
      setError(resolveAdminError(caught));
    } finally {
      setIsAdding(false);
    }
  }

  async function removeSender(senderId: number) {
    setRemovingSenderId(senderId);
    setError(null);

    try {
      await removeWatchedSender(senderId);
      setSenders((current) => current.filter((sender) => sender.id !== senderId));
    } catch (caught) {
      setError(resolveAdminError(caught));
    } finally {
      setRemovingSenderId(null);
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <CooldownSettings />

      <section className="rounded-lg border border-border bg-panel p-5">
        <div className="mb-5 flex items-center justify-between">
          <div className="flex flex-col gap-0.5">
            <span className="text-[14px] font-semibold text-foreground">İzlenen kullanıcılar</span>
            <span className="font-mono text-[11px] text-faint">
              {senders.length} kullanıcı · mesaj atınca e-posta gönderilir
            </span>
          </div>

          <form
            className="flex flex-col items-stretch gap-2 sm:flex-row sm:items-center"
            onSubmit={submitUsername}
          >
            <label className="sr-only" htmlFor="watched-sender-username">
              Kick kullanıcı adı
            </label>
            <div className="flex h-8 w-full items-center gap-1.5 rounded-md border border-border bg-elevated px-2.5 sm:w-60">
              <Plus className="h-3 w-3 shrink-0 text-faint" />
              <input
                className="flex-1 bg-transparent font-sans text-[12px] text-foreground outline-none placeholder:text-faint"
                disabled={isAdding}
                id="watched-sender-username"
                maxLength={60}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="kullanıcı adı ekle"
                value={username}
              />
            </div>
            <Button
              className="h-8 px-3.5 text-[12px]"
              disabled={isAdding || !username.trim()}
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
              KULLANICI
            </span>
            <span className="w-40 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
              EKLENME
            </span>
            <span className="w-28" />
          </div>

          {isLoading ? (
            <div className="px-4 py-10 text-center text-[13px] text-muted-foreground">
              Kullanıcılar yükleniyor...
            </div>
          ) : senders.length ? (
            senders.map((sender) => (
              <WatchedSenderRow
                isRemoving={removingSenderId === sender.id}
                key={sender.id}
                onRemove={() => void removeSender(sender.id)}
                sender={sender}
              />
            ))
          ) : (
            <div className="px-4 py-10 text-center text-[13px] text-muted-foreground">
              Henüz izlenen kullanıcı yok.
            </div>
          )}
        </div>
      </section>
    </div>
  );
}

function CooldownSettings() {
  const [cooldownMinutes, setCooldownMinutes] = useState<number | null>(null);
  const [inputValue, setInputValue] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);

  const loadSettings = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const settings = await getNotificationSettings();
      const minutes = secondsToMinutes(settings.cooldown_seconds);
      setCooldownMinutes(minutes);
      setInputValue(String(minutes));
    } catch (caught) {
      setError(resolveAdminError(caught));
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadSettings();
  }, [loadSettings]);

  async function submitCooldown(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const minutes = Number(inputValue);
    if (
      !Number.isFinite(minutes) ||
      minutes < MIN_COOLDOWN_MINUTES ||
      minutes > MAX_COOLDOWN_MINUTES
    ) {
      setError(
        `Bekleme süresi ${MIN_COOLDOWN_MINUTES}-${MAX_COOLDOWN_MINUTES} dakika arasında olmalı.`
      );
      return;
    }

    setIsSaving(true);
    setError(null);

    try {
      const settings = await updateNotificationSettings({ cooldown_seconds: minutes * 60 });
      setCooldownMinutes(secondsToMinutes(settings.cooldown_seconds));
    } catch (caught) {
      setError(resolveAdminError(caught));
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <section className="rounded-lg border border-border bg-panel p-5">
      <div className="mb-3 flex flex-col gap-0.5">
        <span className="text-[14px] font-semibold text-foreground">Bekleme süresi (cooldown)</span>
        <span className="font-mono text-[11px] text-faint">
          Aynı kullanıcı için art arda mail gönderimi arasındaki minimum süre
          {cooldownMinutes !== null && !isLoading ? ` · şu an ${cooldownMinutes} dk` : ""}
        </span>
      </div>

      <form
        className="flex flex-col items-stretch gap-2 sm:flex-row sm:items-center"
        onSubmit={submitCooldown}
      >
        <label className="sr-only" htmlFor="notification-cooldown-minutes">
          Bekleme süresi (dakika)
        </label>
        <div className="flex h-8 w-full items-center gap-1.5 rounded-md border border-border bg-elevated px-2.5 sm:w-40">
          <input
            className="w-full bg-transparent font-mono text-[12px] text-foreground outline-none"
            disabled={isLoading || isSaving}
            id="notification-cooldown-minutes"
            max={MAX_COOLDOWN_MINUTES}
            min={MIN_COOLDOWN_MINUTES}
            onChange={(e) => setInputValue(e.target.value)}
            type="number"
            value={inputValue}
          />
          <span className="shrink-0 font-mono text-[11px] text-faint">dk</span>
        </div>
        <Button
          className="h-8 px-3.5 text-[12px]"
          disabled={isLoading || isSaving || !inputValue.trim()}
          size="sm"
          type="submit"
        >
          {isSaving ? <Loader2 className="h-3 w-3 animate-spin" /> : null}
          Kaydet
        </Button>
      </form>

      {error ? (
        <div className="mt-3 rounded-md border border-danger bg-elevated px-3 py-2 text-[13px]">
          {error}
        </div>
      ) : null}
    </section>
  );
}

function secondsToMinutes(seconds: number) {
  return Math.round(seconds / 60);
}

function WatchedSenderRow({
  sender,
  isRemoving,
  onRemove
}: {
  sender: WatchedSender;
  isRemoving: boolean;
  onRemove: () => void;
}) {
  const removeButton = (
    <Button disabled={isRemoving} onClick={onRemove} size="sm" type="button" variant="outline">
      {isRemoving ? (
        <Loader2 className="h-3 w-3 animate-spin" />
      ) : (
        <Trash2 className="h-3 w-3 text-danger" />
      )}
      <span className="hidden sm:inline">Kaldır</span>
    </Button>
  );

  return (
    <div className="border-b border-border px-3 py-2.5 last:border-b-0">
      {/* Desktop layout */}
      <div className="hidden items-center sm:flex">
        <div className="flex-1 font-mono text-[13px] text-accent">@{sender.username}</div>
        <div className="w-40">
          <span className="font-mono text-[11px] text-faint">{formatDate(sender.created_at)}</span>
        </div>
        <div className="flex w-28 justify-end">{removeButton}</div>
      </div>

      {/* Mobile card layout */}
      <div className="flex items-center justify-between gap-2 sm:hidden">
        <div className="min-w-0">
          <div className="truncate font-mono text-[13px] text-accent">@{sender.username}</div>
          <div className="font-mono text-[11px] text-faint">{formatDate(sender.created_at)}</div>
        </div>
        {removeButton}
      </div>
    </div>
  );
}

function mergeSender(current: WatchedSender[], sender: WatchedSender) {
  const next = current.filter((item) => item.id !== sender.id);
  return [...next, sender].sort((a, b) => a.username.localeCompare(b.username));
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
  return "Kullanıcı işlemi tamamlanırken hata oluştu.";
}
