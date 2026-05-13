"use client";

import Image from "next/image";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";

import { MessageList } from "@/features/search/message-list";
import { buildMessageExportUrl, searchMessages } from "@/features/search/api";
import { SearchForm } from "@/features/search/search-form";
import {
  EMPTY_SEARCH_STATE,
  appendUniqueMessages,
  applyDatePreset,
  getDefaultSearchState,
  readSearchState,
  searchStateToMessageParams,
  searchStateToUrlSearchParams,
  type SearchFormState
} from "@/features/search/search-params";
import { SearchSummary } from "@/features/search/search-summary";
import { DEFAULT_MESSAGE_LIMIT } from "@/lib/constants";
import type { Message, MessageExportFormat } from "@/types/api";

export function SearchScreen() {
  return (
    <Suspense fallback={<SearchScreenLoading />}>
      <SearchScreenInner />
    </Suspense>
  );
}

function SearchScreenInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryKey = searchParams.toString();
  const sentinelRef = useRef<HTMLDivElement>(null);
  const requestSequenceRef = useRef(0);
  // The default date range depends on browser timezone, so fill it after hydration.
  const [formState, setFormState] = useState<SearchFormState>(EMPTY_SEARCH_STATE);
  const [submittedState, setSubmittedState] = useState<SearchFormState>(EMPTY_SEARCH_STATE);
  const [messages, setMessages] = useState<Message[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [isInitialLoading, setIsInitialLoading] = useState(false);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasSearched, setHasSearched] = useState(false);

  const clearResults = useCallback(() => {
    requestSequenceRef.current += 1;
    setMessages([]);
    setNextCursor(null);
    setIsInitialLoading(false);
    setIsLoadingMore(false);
    setError(null);
  }, []);

  const loadFirstPage = useCallback(async (state: SearchFormState) => {
    const requestId = ++requestSequenceRef.current;
    setIsInitialLoading(true);
    setError(null);

    try {
      const page = await searchMessages({
        ...searchStateToMessageParams(state),
        limit: DEFAULT_MESSAGE_LIMIT
      });

      if (requestId !== requestSequenceRef.current) {
        return;
      }

      setMessages(page.items);
      setNextCursor(page.next_cursor);
    } catch (caught) {
      if (requestId === requestSequenceRef.current) {
        setMessages([]);
        setNextCursor(null);
        setError(resolveSearchError(caught));
      }
    } finally {
      if (requestId === requestSequenceRef.current) {
        setIsInitialLoading(false);
      }
    }
  }, []);

  const loadNextPage = useCallback(async () => {
    if (!nextCursor || isInitialLoading || isLoadingMore) {
      return;
    }

    const requestId = ++requestSequenceRef.current;
    setIsLoadingMore(true);
    setError(null);

    try {
      const page = await searchMessages({
        ...searchStateToMessageParams(submittedState),
        cursor: nextCursor,
        limit: DEFAULT_MESSAGE_LIMIT
      });

      if (requestId !== requestSequenceRef.current) {
        return;
      }

      setMessages((current) => appendUniqueMessages(current, page.items));
      setNextCursor(page.next_cursor);
    } catch (caught) {
      if (requestId === requestSequenceRef.current) {
        setError(resolveSearchError(caught));
      }
    } finally {
      if (requestId === requestSequenceRef.current) {
        setIsLoadingMore(false);
      }
    }
  }, [isInitialLoading, isLoadingMore, nextCursor, submittedState]);

  useEffect(() => {
    const hasUrlSearch = queryKey.length > 0;
    const nextState = readSearchState(new URLSearchParams(queryKey));

    setFormState(nextState);
    setSubmittedState(hasUrlSearch ? nextState : EMPTY_SEARCH_STATE);
    setHasSearched(hasUrlSearch);

    if (hasUrlSearch) {
      void loadFirstPage(nextState);
      return;
    }

    clearResults();
  }, [clearResults, loadFirstPage, queryKey]);

  useEffect(() => {
    const node = sentinelRef.current;

    if (!node || !nextCursor) {
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          void loadNextPage();
        }
      },
      { rootMargin: "360px" }
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, [loadNextPage, nextCursor]);

  const activeQueryString = useMemo(
    () => searchStateToUrlSearchParams(formState).toString(),
    [formState]
  );

  function submitSearch() {
    setSubmittedState(formState);
    setHasSearched(true);

    if (activeQueryString === queryKey) {
      void loadFirstPage(formState);
      return;
    }

    router.push(activeQueryString ? `/search?${activeQueryString}` : "/search");
  }

  function resetSearch() {
    const defaultState = getDefaultSearchState();

    setFormState(defaultState);
    setSubmittedState(EMPTY_SEARCH_STATE);
    setHasSearched(false);
    clearResults();

    if (queryKey) {
      router.push("/search");
    }
  }

  function exportCurrentSearch(format: MessageExportFormat) {
    const url = buildMessageExportUrl(searchStateToMessageParams(submittedState), format);
    window.open(url, "_blank", "noopener,noreferrer");
  }

  return (
    <main className="min-h-screen bg-background px-4 py-4 text-foreground md:px-8 md:py-6">
      <div className="mx-auto flex max-w-[1440px] flex-col gap-6">
        <SearchHeader />

        <div className="grid gap-6 xl:grid-cols-[minmax(0,900px)_428px]">
          <SearchForm
            canExport={hasSearched}
            isLoading={isInitialLoading}
            onChange={setFormState}
            onDatePreset={(preset) => setFormState((current) => applyDatePreset(current, preset))}
            onExport={exportCurrentSearch}
            onReset={resetSearch}
            onSubmit={submitSearch}
            value={formState}
          />
          <SearchSummary
            error={error}
            isLoading={isInitialLoading || isLoadingMore}
            messages={messages}
            state={submittedState}
          />
        </div>

        <MessageList
          error={error}
          hasMore={Boolean(nextCursor)}
          hasSearched={hasSearched}
          highlightQuery={submittedState.q}
          isInitialLoading={isInitialLoading}
          isLoadingMore={isLoadingMore}
          messages={messages}
          onRetry={() => void loadFirstPage(submittedState)}
          sentinelRef={sentinelRef}
        />
      </div>
    </main>
  );
}

function SearchHeader() {
  return (
    <header className="flex flex-wrap items-center justify-between gap-4 border-b border-border bg-black px-4 py-4 md:px-6">
      <div className="flex min-w-0 items-center gap-4">
        <Image
          alt="Kick Logs"
          className="h-11 w-11 shrink-0 rounded-md object-contain"
          height={44}
          priority
          src="/app-logo.png"
          width={44}
        />
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-lg font-semibold">Kick Logs</h1>
            <span className="rounded-md bg-kick-background px-2 py-1 text-xs text-primary">
              /search
            </span>
          </div>
          <p className="text-xs text-muted-foreground">Sohbet arşivinde hızlı sorgu</p>
        </div>
      </div>

      <div className="grid w-full gap-0 overflow-hidden rounded-md border border-border bg-kick-background text-xs sm:w-auto sm:grid-cols-3">
        <HeaderMetric label="Kapsam" value="Tüm kanallar" />
        <HeaderMetric label="Sıralama" value="Yeni -> eski" />
        <HeaderMetric label="Filtre" value="AND" isPrimary />
      </div>
    </header>
  );
}

function HeaderMetric({
  isPrimary = false,
  label,
  value
}: {
  isPrimary?: boolean;
  label: string;
  value: string;
}) {
  return (
    <div className="border-b border-border px-4 py-2 last:border-b-0 sm:border-b-0 sm:border-r sm:last:border-r-0">
      <div className="text-muted-foreground">{label}</div>
      <div className={isPrimary ? "font-semibold text-primary" : "text-foreground"}>{value}</div>
    </div>
  );
}

function SearchScreenLoading() {
  return (
    <main className="min-h-screen bg-background px-6 py-6 text-foreground">
      <div className="mx-auto max-w-[1440px] rounded-lg border border-border bg-black p-6 text-sm text-muted-foreground">
        Arama ekranı yükleniyor...
      </div>
    </main>
  );
}

function resolveSearchError(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }

  return "Mesajlar yüklenirken bir hata oluştu.";
}
