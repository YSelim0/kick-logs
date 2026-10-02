"use client";

import { Timer } from "lucide-react";
import { useTranslations } from "next-intl";
import { getUiErrorKey, type UiErrorKey } from "@/i18n/errors";
import { useUiFormat } from "@/i18n/use-ui-format";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";

import { SiteHeader } from "@/components/site-header";
import { MessageList } from "@/features/search/message-list";
import { buildMessageExportUrl, searchMessages } from "@/features/search/api";
import { SearchForm } from "@/features/search/search-form";
import {
  EMPTY_SEARCH_STATE,
  appendUniqueMessages,
  applyDatePreset,
  dedupeMessages,
  getDefaultSearchState,
  readSearchState,
  searchStateToMessageParams,
  searchStateToUrlSearchParams,
  type SearchFormState
} from "@/features/search/search-params";
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
  const t = useTranslations("search");
  const errors = useTranslations("common.errors");
  const format = useUiFormat();
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryKey = searchParams.toString();
  const sentinelRef = useRef<HTMLDivElement>(null);
  const requestSequenceRef = useRef(0);
  const [formState, setFormState] = useState<SearchFormState>(EMPTY_SEARCH_STATE);
  const [submittedState, setSubmittedState] = useState<SearchFormState>(EMPTY_SEARCH_STATE);
  const [messages, setMessages] = useState<Message[]>([]);
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [isInitialLoading, setIsInitialLoading] = useState(false);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [error, setError] = useState<UiErrorKey | null>(null);
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

      setMessages(dedupeMessages(page.items));
      setNextCursor(page.next_cursor);
    } catch (caught) {
      if (requestId === requestSequenceRef.current) {
        setMessages([]);
        setNextCursor(null);
        setError(getUiErrorKey(caught, "search"));
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
        setError(getUiErrorKey(caught, "search"));
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

  const scopeLabel = useMemo(() => {
    const channel = submittedState.channel.trim();
    const sender = submittedState.sender.trim();
    const q = submittedState.q.trim();
    const parts = [channel || t("allChannels"), t("newestFirst")];
    if (sender) {
      parts.push(sender);
    }
    if (q) {
      parts.push(q);
    }
    return parts.join(" · ");
  }, [submittedState.channel, submittedState.sender, submittedState.q, t]);

  const resultCountLabel = useMemo(() => {
    if (!hasSearched) {
      return null;
    }
    return t("resultCount", { count: messages.length, value: format.number(messages.length) });
  }, [hasSearched, messages.length, t, format]);

  const lastMatchLabel = useMemo(() => {
    const first = messages[0];
    if (!first) {
      return null;
    }
    return t("lastMatch", { date: format.dateTime(first.message_created_at) });
  }, [messages, t, format]);

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
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute="search" />

      <div className="mx-auto flex max-w-[1280px] flex-col gap-5 px-6 py-6 md:py-10">
        <header className="flex flex-wrap items-baseline gap-3">
          <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
          <span className="font-mono text-[11px] text-muted-foreground">{scopeLabel}</span>
        </header>

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

        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-baseline gap-2">
            <h2 className="text-[13px] font-semibold text-foreground">{t("results")}</h2>
            {resultCountLabel ? (
              <span className="font-mono text-[11px] text-muted-foreground">
                {resultCountLabel}
              </span>
            ) : null}
          </div>
          {lastMatchLabel ? (
            <div className="inline-flex items-center gap-1.5 font-mono text-[11px] text-faint">
              <Timer className="h-3 w-3" />
              {lastMatchLabel}
            </div>
          ) : null}
        </div>

        <MessageList
          error={error ? (error === "unavailable" ? t("loadError") : errors(error)) : null}
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

function SearchScreenLoading() {
  const t = useTranslations("search");
  return (
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute="search" />
      <div className="mx-auto max-w-[1280px] px-6 py-10 text-[13px] text-muted-foreground">
        {t("screenLoading")}
      </div>
    </main>
  );
}
