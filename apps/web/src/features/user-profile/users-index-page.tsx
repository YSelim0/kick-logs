"use client";

/* eslint-disable @next/next/no-img-element */

import { Search, User } from "lucide-react";
import Link from "next/link";
import { useCallback, useState } from "react";

import { SiteHeader } from "@/components/site-header";
import { getTopSenders } from "@/features/analytics/api";
import { buildUserProfileHref } from "@/lib/kick-profile-slugs";
import type { TopSenderAnalytics } from "@/types/api";

type SearchState = "idle" | "loading" | "ready" | "empty" | "error";

const RESULT_LIMIT = 20;
const MIN_QUERY_LENGTH = 2;

export function UsersIndexPage() {
  const [query, setQuery] = useState("");
  const [submittedQuery, setSubmittedQuery] = useState("");
  const [results, setResults] = useState<TopSenderAnalytics[]>([]);
  const [state, setState] = useState<SearchState>("idle");

  const search = useCallback(async (q: string) => {
    setState("loading");
    try {
      const data = await getTopSenders({ q, limit: RESULT_LIMIT });
      const items = data?.items ?? [];
      setResults(items);
      setState(items.length === 0 ? "empty" : "ready");
    } catch {
      setState("error");
      setResults([]);
    }
  }, []);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      const trimmed = query.trim();
      if (trimmed.length < MIN_QUERY_LENGTH) {
        return;
      }
      setSubmittedQuery(trimmed);
      void search(trimmed);
    },
    [query, search]
  );

  const canSubmit = query.trim().length >= MIN_QUERY_LENGTH && state !== "loading";

  return (
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute="users" />

      <div className="mx-auto max-w-[1280px] px-6 py-6">
        {/* Page title */}
        <div className="mb-5">
          <h1 className="text-[22px] font-semibold leading-none tracking-tight text-foreground">
            Users
          </h1>
          <p className="mt-1 font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
            Mesaj sayısına göre sıralı
          </p>
        </div>

        {/* Search form */}
        <form className="mb-5 flex max-w-lg gap-2" onSubmit={handleSubmit}>
          <div className="relative flex-1">
            <Search
              aria-hidden
              className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            />
            <input
              aria-label="Kullanıcı ara"
              autoComplete="off"
              className="h-10 w-full rounded-md border border-border bg-elevated pl-9 pr-4 text-[13px] text-foreground placeholder:text-muted-foreground focus:border-border-strong focus:outline-none"
              id="users-search-input"
              minLength={MIN_QUERY_LENGTH}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Kullanıcı adı veya slug ile ara…"
              spellCheck={false}
              type="search"
              value={query}
            />
          </div>
          <button
            className="h-10 shrink-0 rounded-md bg-accent px-4 text-[13px] font-medium text-text-on-accent transition-colors hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-50"
            disabled={!canSubmit}
            type="submit"
          >
            {state === "loading" ? "Aranıyor…" : "Ara"}
          </button>
        </form>

        {/* Content area */}
        {state === "idle" ? <IdlePrompt /> : null}
        {state === "loading" ? <LoadingState /> : null}
        {state === "error" ? <ErrorState /> : null}
        {state === "empty" ? <EmptyState query={submittedQuery} /> : null}
        {state === "ready" ? <UserList users={results} /> : null}
      </div>
    </main>
  );
}

function IdlePrompt() {
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-elevated">
        <User className="h-5 w-5 text-muted-foreground" />
      </div>
      <p className="text-[15px] font-medium text-foreground">Kullanıcı bulmak için arama yapın</p>
      <p className="mt-1 text-[13px] text-muted-foreground">
        Kullanıcı adı veya slug girin ve Ara butonuna basın.
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
      &quot;{query}&quot; için kullanıcı bulunamadı.
    </p>
  );
}

function UserList({ users }: { users: TopSenderAnalytics[] }) {
  return (
    <section aria-label="Kullanıcı sonuçları" className="rounded-lg border border-border bg-panel">
      <div className="divide-y divide-border">
        {users.map((user) => (
          <UserRow key={user.sender_id} user={user} />
        ))}
      </div>
    </section>
  );
}

function UserRow({ user }: { user: TopSenderAnalytics }) {
  const [imgFailed, setImgFailed] = useState(false);
  const href = buildUserProfileHref(user.slug);
  const initial = user.username.charAt(0).toUpperCase() || "U";

  const inner = (
    <>
      {/* Avatar */}
      {user.profile_image_url && !imgFailed ? (
        <img
          alt={user.username}
          className="h-9 w-9 shrink-0 rounded-full border border-border object-cover"
          height={36}
          onError={() => setImgFailed(true)}
          src={user.profile_image_url}
          width={36}
        />
      ) : (
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-border bg-elevated font-mono text-sm font-semibold text-muted-foreground">
          {initial}
        </div>
      )}

      {/* Username & slug */}
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium text-foreground">@{user.username}</p>
        {user.slug !== user.username ? (
          <p className="truncate font-mono text-[11px] text-muted-foreground">{user.slug}</p>
        ) : null}
      </div>

      {/* Message count */}
      <div className="shrink-0 text-right">
        <p className="font-mono text-[13px] font-semibold text-foreground">
          {formatCompactNumber(user.message_count)}
        </p>
        <p className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
          mesaj
        </p>
      </div>
    </>
  );

  if (href) {
    return (
      <Link
        className="flex items-center gap-4 px-4 py-3 transition-colors hover:bg-elevated"
        href={href}
      >
        {inner}
      </Link>
    );
  }

  return <div className="flex items-center gap-4 px-4 py-3">{inner}</div>;
}

const COMPACT_FORMATTER = new Intl.NumberFormat("tr-TR", {
  notation: "compact",
  maximumFractionDigits: 1
});

function formatCompactNumber(value: number) {
  return COMPACT_FORMATTER.format(value);
}
