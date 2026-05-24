"use client";

/* eslint-disable @next/next/no-img-element */

import { Hash, Search } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";

import { SiteHeader } from "@/components/site-header";
import { getTopChannels } from "@/features/analytics/api";
import type { TopChannelAnalytics } from "@/types/api";

type SearchState = "idle" | "loading" | "ready" | "empty" | "error";

const DEBOUNCE_MS = 300;
const RESULT_LIMIT = 20;

export function ChannelsIndexPage() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<TopChannelAnalytics[]>([]);
  const [state, setState] = useState<SearchState>("idle");
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const search = useCallback(async (q: string) => {
    if (q.trim() === "") {
      setState("idle");
      setResults([]);
      return;
    }

    setState("loading");
    try {
      const data = await getTopChannels({ q: q.trim(), limit: RESULT_LIMIT });
      const items = data?.items ?? [];
      setResults(items);
      setState(items.length === 0 ? "empty" : "ready");
    } catch {
      setState("error");
      setResults([]);
    }
  }, []);

  useEffect(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
    debounceRef.current = setTimeout(() => {
      void search(query);
    }, DEBOUNCE_MS);

    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [query, search]);

  return (
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute="channels" />

      <div className="mx-auto max-w-[1280px] px-6 py-6">
        {/* Page title */}
        <div className="mb-5">
          <h1 className="text-[22px] font-semibold leading-none tracking-tight text-foreground">
            Channels
          </h1>
          <p className="mt-1 font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
            Mesaj sayısına göre sıralı
          </p>
        </div>

        {/* Search input */}
        <div className="relative mb-5 max-w-lg">
          <Search
            aria-hidden
            className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
          />
          <input
            aria-label="Kanal ara"
            autoComplete="off"
            className="h-10 w-full rounded-md border border-border bg-elevated pl-9 pr-4 text-[13px] text-foreground placeholder:text-muted-foreground focus:border-border-strong focus:outline-none"
            id="channels-search-input"
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Kanal adı veya slug ile ara…"
            spellCheck={false}
            type="search"
            value={query}
          />
        </div>

        {/* Content area */}
        {state === "idle" ? <IdlePrompt /> : null}
        {state === "loading" ? <LoadingState /> : null}
        {state === "error" ? <ErrorState /> : null}
        {state === "empty" ? <EmptyState query={query} /> : null}
        {state === "ready" ? <ChannelList channels={results} /> : null}
      </div>
    </main>
  );
}

function IdlePrompt() {
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-elevated">
        <Hash className="h-5 w-5 text-muted-foreground" />
      </div>
      <p className="text-[15px] font-medium text-foreground">Kanal bulmak için arama yapın</p>
      <p className="mt-1 text-[13px] text-muted-foreground">
        Kanal adı veya slug girin — sonuçlar otomatik gelir.
      </p>
    </div>
  );
}

function LoadingState() {
  return (
    <div className="flex items-center gap-2 py-8 text-[13px] text-muted-foreground">
      <span aria-hidden className="inline-block h-1.5 w-1.5 animate-pulse rounded-full bg-accent" />
      <span className="font-mono">aranıyor…</span>
    </div>
  );
}

function ErrorState() {
  return <p className="py-8 text-[13px] text-danger">Sonuçlar alınamadı. Lütfen tekrar deneyin.</p>;
}

function EmptyState({ query }: { query: string }) {
  return (
    <p className="py-8 text-[13px] text-muted-foreground">
      &quot;{query}&quot; için kanal bulunamadı.
    </p>
  );
}

function ChannelList({ channels }: { channels: TopChannelAnalytics[] }) {
  return (
    <section aria-label="Kanal sonuçları" className="rounded-lg border border-border bg-panel">
      <div className="divide-y divide-border">
        {channels.map((channel) => (
          <ChannelRow channel={channel} key={channel.channel_id} />
        ))}
      </div>
    </section>
  );
}

function ChannelRow({ channel }: { channel: TopChannelAnalytics }) {
  const [imgFailed, setImgFailed] = useState(false);
  const initial = channel.display_name.charAt(0).toUpperCase() || "#";

  return (
    <Link
      className="flex items-center gap-4 px-4 py-3 transition-colors hover:bg-elevated"
      href={`/channels/${encodeURIComponent(channel.slug)}`}
    >
      {/* Channel image */}
      {channel.profile_image_url && !imgFailed ? (
        <img
          alt={channel.display_name}
          className="h-10 w-10 shrink-0 rounded-md border border-border object-cover"
          height={40}
          onError={() => setImgFailed(true)}
          src={channel.profile_image_url}
          width={40}
        />
      ) : (
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md border border-border bg-elevated font-mono text-sm font-semibold text-muted-foreground">
          {initial}
        </div>
      )}

      {/* Name & slug */}
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium text-foreground">{channel.display_name}</p>
        <p className="truncate font-mono text-[11px] text-muted-foreground">#{channel.slug}</p>
      </div>

      {/* Message count */}
      <div className="shrink-0 text-right">
        <p className="font-mono text-[13px] font-semibold text-foreground">
          {formatCompactNumber(channel.message_count)}
        </p>
        <p className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
          mesaj
        </p>
      </div>

      {/* Last activity */}
      <div className="hidden shrink-0 text-right sm:block">
        <p className="font-mono text-[11px] text-muted-foreground">
          {formatShortDate(channel.latest_message_at)}
        </p>
        <p className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
          son aktivite
        </p>
      </div>
    </Link>
  );
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
