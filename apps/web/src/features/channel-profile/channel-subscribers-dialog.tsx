"use client";

/* eslint-disable @next/next/no-img-element */

import { Download, FileJson, FileText, Gift, Loader2 } from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";
import { useEffect, useRef, useState } from "react";

import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog";
import {
  buildChannelSubscribersExportUrl,
  getChannelSubscribers
} from "@/features/channel-profile/api";
import { buildUserProfileHref } from "@/lib/kick-profile-slugs";
import { cn } from "@/lib/utils";
import type { ChannelSubscriber, ChannelSubscriberExportFormat } from "@/types/api";

export type ChannelSubscriberMode = "all" | "gifted";

type LoadStatus = "idle" | "loading" | "loading-more" | "ready" | "error";

const PAGE_SIZE = 50;

type ChannelSubscribersDialogProps = {
  channelSlug: string;
  mode: ChannelSubscriberMode | null;
  onOpenChange: (open: boolean) => void;
};

export function ChannelSubscribersDialog({
  channelSlug,
  mode,
  onOpenChange
}: ChannelSubscribersDialogProps) {
  const open = mode !== null;
  const giftOnly = mode === "gifted";
  const [items, setItems] = useState<ChannelSubscriber[]>([]);
  const [count, setCount] = useState(0);
  const [status, setStatus] = useState<LoadStatus>("idle");
  const [isExportMenuOpen, setIsExportMenuOpen] = useState(false);
  const exportMenuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open || !mode) {
      setItems([]);
      setCount(0);
      setStatus("idle");
      setIsExportMenuOpen(false);
      return;
    }

    let isMounted = true;
    setItems([]);
    setCount(0);
    setStatus("loading");
    setIsExportMenuOpen(false);

    getChannelSubscribers(channelSlug, {
      limit: PAGE_SIZE,
      offset: 0,
      gift_only: giftOnly
    })
      .then((page) => {
        if (!isMounted) return;
        setItems(page.items);
        setCount(page.count);
        setStatus("ready");
      })
      .catch(() => {
        if (!isMounted) return;
        setStatus("error");
      });

    return () => {
      isMounted = false;
    };
  }, [channelSlug, giftOnly, mode, open]);

  useEffect(() => {
    if (!isExportMenuOpen) {
      return;
    }

    function closeOnOutsideClick(event: MouseEvent | TouchEvent) {
      const target = event.target;
      if (target instanceof Node && !exportMenuRef.current?.contains(target)) {
        setIsExportMenuOpen(false);
      }
    }

    document.addEventListener("mousedown", closeOnOutsideClick);
    document.addEventListener("touchstart", closeOnOutsideClick);

    return () => {
      document.removeEventListener("mousedown", closeOnOutsideClick);
      document.removeEventListener("touchstart", closeOnOutsideClick);
    };
  }, [isExportMenuOpen]);

  async function loadMore() {
    if (!mode || status === "loading-more") {
      return;
    }

    setStatus("loading-more");
    try {
      const page = await getChannelSubscribers(channelSlug, {
        limit: PAGE_SIZE,
        offset: items.length,
        gift_only: giftOnly
      });
      setItems((current) => [...current, ...page.items]);
      setCount(page.count);
      setStatus("ready");
    } catch {
      setStatus("error");
    }
  }

  function exportSubscribers(format: ChannelSubscriberExportFormat) {
    if (!mode) {
      return;
    }
    setIsExportMenuOpen(false);
    window.open(
      buildChannelSubscribersExportUrl(channelSlug, giftOnly, format),
      "_blank",
      "noopener,noreferrer"
    );
  }

  const hasMore = items.length < count;
  const isInitialLoading = status === "loading";
  const title = giftOnly ? "Hediye aktif aboneler" : "Aktif aboneler";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[calc(100vh-2rem)] max-w-4xl flex-col overflow-hidden border-border bg-panel !p-0 text-foreground shadow-none sm:max-h-[85vh]">
        <DialogClose onClose={() => onOpenChange(false)} />
        <div className="flex min-h-0 flex-1 flex-col">
          <div className="border-b border-border px-5 py-4">
            <div className="flex flex-col gap-3 pr-8 sm:flex-row sm:items-start sm:justify-between">
              <DialogHeader>
                <DialogTitle className="text-[18px]">{title}</DialogTitle>
                <DialogDescription className="text-[12px] text-muted-foreground">
                  #{channelSlug} · {formatCompactNumber(count)} kayıt
                </DialogDescription>
              </DialogHeader>

              <div className="relative shrink-0" ref={exportMenuRef}>
                <button
                  aria-expanded={isExportMenuOpen}
                  aria-label="Abone listesini indir"
                  className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border bg-elevated text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={isInitialLoading}
                  onClick={() => setIsExportMenuOpen((current) => !current)}
                  title="Abone listesini indir"
                  type="button"
                >
                  <Download className="h-4 w-4" />
                </button>

                {isExportMenuOpen ? (
                  <div className="absolute right-0 z-20 mt-2 grid min-w-[160px] gap-1 rounded-md border border-border bg-panel p-1.5 shadow-lg">
                    <ExportButton
                      icon={<FileJson className="h-4 w-4 text-muted-foreground" />}
                      label="JSON indir"
                      onClick={() => exportSubscribers("json")}
                    />
                    <ExportButton
                      icon={<FileText className="h-4 w-4 text-muted-foreground" />}
                      label="CSV indir"
                      onClick={() => exportSubscribers("csv")}
                    />
                    <ExportButton
                      icon={<FileText className="h-4 w-4 text-muted-foreground" />}
                      label="TXT indir"
                      onClick={() => exportSubscribers("txt")}
                    />
                  </div>
                ) : null}
              </div>
            </div>
          </div>

          <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
            {isInitialLoading ? (
              <StateMessage
                icon={<Loader2 className="h-4 w-4 animate-spin" />}
                text="Aboneler yükleniyor..."
              />
            ) : null}

            {status === "error" ? (
              <StateMessage text="Abone listesi şu anda alınamadı." tone="danger" />
            ) : null}

            {status !== "loading" && status !== "error" && items.length === 0 ? (
              <StateMessage text="Bu kanal için henüz aktif abonelik kaydı yok." />
            ) : null}

            {items.length > 0 ? (
              <div className="overflow-hidden rounded-md border border-border">
                {items.map((subscriber) => (
                  <SubscriberRow key={subscriber.subscriber_kick_user_id} subscriber={subscriber} />
                ))}
              </div>
            ) : null}
          </div>

          {items.length > 0 ? (
            <div className="shrink-0 border-t border-border px-5 py-4">
              <div className="flex items-center justify-between gap-3">
                <p className="font-mono text-[11px] text-muted-foreground">
                  {formatCompactNumber(items.length)} / {formatCompactNumber(count)}
                </p>
                <button
                  className="inline-flex h-9 items-center justify-center rounded-md border border-border bg-elevated px-3 text-[13px] font-medium text-foreground transition-colors hover:border-border-strong disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={!hasMore || status === "loading-more"}
                  onClick={loadMore}
                  type="button"
                >
                  {status === "loading-more" ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : null}
                  Daha fazla yükle
                </button>
              </div>
            </div>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function SubscriberRow({ subscriber }: { subscriber: ChannelSubscriber }) {
  const href = buildUserProfileHref(subscriber.slug);
  const gifterHref = subscriber.gifter_slug ? buildUserProfileHref(subscriber.gifter_slug) : null;
  const username =
    subscriber.username || subscriber.slug || String(subscriber.subscriber_kick_user_id);

  return (
    <div className="grid grid-cols-1 gap-3 border-b border-border px-3 py-3 text-[13px] last:border-b-0 md:grid-cols-[minmax(0,1fr)_150px_150px] md:items-center">
      <div className="flex min-w-0 items-center gap-3">
        <SubscriberAvatar subscriber={subscriber} username={username} />
        <div className="min-w-0">
          {href ? (
            <Link className="truncate font-semibold text-foreground hover:underline" href={href}>
              {username}
            </Link>
          ) : (
            <span className="truncate font-semibold text-foreground">{username}</span>
          )}
          <div className="mt-1 flex min-w-0 flex-wrap items-center gap-2 font-mono text-[10px] uppercase text-muted-foreground">
            <span>ID {subscriber.subscriber_kick_user_id}</span>
            {subscriber.is_gift ? (
              <span className="inline-flex items-center gap-1 text-accent">
                <Gift className="h-3 w-3" />
                Hediye
              </span>
            ) : null}
            {subscriber.is_gift && subscriber.gifter_username ? (
              <span className="min-w-0 normal-case">
                veren{" "}
                {gifterHref ? (
                  <Link className="text-muted-foreground hover:text-foreground" href={gifterHref}>
                    {subscriber.gifter_username}
                  </Link>
                ) : (
                  subscriber.gifter_username
                )}
              </span>
            ) : null}
          </div>
        </div>
      </div>

      <DateBlock label="Başlangıç" value={subscriber.started_at} />
      <DateBlock label="Bitiş" value={subscriber.expires_at} />
    </div>
  );
}

function SubscriberAvatar({
  subscriber,
  username
}: {
  subscriber: ChannelSubscriber;
  username: string;
}) {
  const [failed, setFailed] = useState(false);
  const imageUrl = subscriber.profile_image_url;

  if (imageUrl && !failed) {
    return (
      <img
        alt={`${username} profil`}
        className="h-9 w-9 shrink-0 rounded-full border border-border object-cover"
        height={36}
        onError={() => setFailed(true)}
        src={imageUrl}
        width={36}
      />
    );
  }

  return (
    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-border bg-elevated font-mono text-[12px] font-semibold uppercase text-muted-foreground">
      {username.slice(0, 1) || "?"}
    </div>
  );
}

function DateBlock({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="font-mono text-[10px] uppercase text-muted-foreground">{label}</div>
      <div className="mt-1 font-mono text-[12px] text-foreground">{formatDateTime(value)}</div>
    </div>
  );
}

function ExportButton({
  icon,
  label,
  onClick
}: {
  icon: ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      className="inline-flex h-8 items-center gap-2 rounded-sm px-2 text-[13px] text-foreground hover:bg-elevated"
      onClick={onClick}
      type="button"
    >
      {icon}
      {label}
    </button>
  );
}

function StateMessage({
  icon,
  text,
  tone = "default"
}: {
  icon?: ReactNode;
  text: string;
  tone?: "default" | "danger";
}) {
  return (
    <div
      className={cn(
        "flex min-h-[180px] items-center justify-center gap-2 rounded-md border border-border bg-black/20 px-4 text-center text-[13px]",
        tone === "danger" ? "text-danger" : "text-muted-foreground"
      )}
    >
      {icon}
      {text}
    </div>
  );
}

const COMPACT_FORMATTER = new Intl.NumberFormat("tr-TR", {
  notation: "compact",
  maximumFractionDigits: 1
});

function formatCompactNumber(value: number) {
  return COMPACT_FORMATTER.format(value);
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat("tr-TR", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}
