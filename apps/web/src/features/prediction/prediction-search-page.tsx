"use client";

import { Search, TrendingUp } from "lucide-react";
import { useRouter } from "next/navigation";
import { useCallback, useState } from "react";

import { SiteHeader } from "@/components/site-header";

const MIN_QUERY_LENGTH = 2;

export function PredictionSearchPage() {
  const router = useRouter();
  const [query, setQuery] = useState("");

  const handleSubmit = useCallback(
    (event: React.FormEvent) => {
      event.preventDefault();
      const slug = query.trim().toLowerCase();
      if (slug.length < MIN_QUERY_LENGTH) {
        return;
      }
      router.push(`/prediction/${encodeURIComponent(slug)}`);
    },
    [query, router]
  );

  const canSubmit = query.trim().length >= MIN_QUERY_LENGTH;

  return (
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute="prediction" />

      <div className="mx-auto max-w-[1280px] px-6 py-6">
        <div className="mb-5">
          <h1 className="text-[22px] font-semibold leading-none tracking-tight text-foreground">
            Prediction
          </h1>
          <p className="mt-1 font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
            Kanalın son tahmin oyunu
          </p>
        </div>

        <form className="mb-5 flex max-w-lg gap-2" onSubmit={handleSubmit}>
          <div className="relative flex-1">
            <Search
              aria-hidden
              className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            />
            <input
              aria-label="Kanal adı"
              autoComplete="off"
              className="h-10 w-full rounded-md border border-border bg-elevated pl-9 pr-4 text-[13px] text-foreground placeholder:text-muted-foreground focus:border-border-strong focus:outline-none"
              id="prediction-search-input"
              minLength={MIN_QUERY_LENGTH}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Kanal adı girin…"
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
            Göster
          </button>
        </form>

        <IdlePrompt />
      </div>
    </main>
  );
}

function IdlePrompt() {
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-elevated">
        <TrendingUp className="h-5 w-5 text-muted-foreground" />
      </div>
      <p className="text-[15px] font-medium text-foreground">Tahmin verisi için kanal seçin</p>
      <p className="mt-1 text-[13px] text-muted-foreground">
        Kanal adı girin ve Göster butonuna basın.
      </p>
    </div>
  );
}
