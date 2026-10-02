import { useEffect, useRef, useState } from "react";

import { getDirectoryChannels, getDirectoryUsers } from "@/features/directory/api";
import type { DirectoryIdentity, DirectoryKind } from "@/types/directory";
import { getUiErrorKey, type UiErrorKey } from "@/i18n/errors";

type SearchState = "idle" | "loading" | "ready" | "empty" | "error";
type ActiveRequest = { prefix: string; append: boolean; controller: AbortController };

export function validDirectoryPrefix(value: string) {
  const length = Array.from(value.trim()).length;
  return (
    length >= 2 &&
    length <= 160 &&
    !Array.from(value).some((character) => {
      const code = character.codePointAt(0)!;
      return code < 32 || (code >= 127 && code <= 159);
    })
  );
}

export function useDirectorySearch(kind: DirectoryKind) {
  const [submittedQuery, setSubmittedQuery] = useState("");
  const [items, setItems] = useState<DirectoryIdentity[]>([]);
  const [state, setState] = useState<SearchState>("idle");
  const [nextCursor, setNextCursor] = useState<string | null>(null);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<UiErrorKey | null>(null);
  const active = useRef<ActiveRequest | null>(null);

  useEffect(
    () => () => {
      active.current?.controller.abort();
      active.current = null;
    },
    []
  );

  async function search(value: string, append = false) {
    if (!validDirectoryPrefix(value)) return;
    const prefix = value.trim();
    if (append && (active.current || !nextCursor)) return;
    if (!append && active.current?.prefix === prefix && !active.current.append) return;

    active.current?.controller.abort();
    const request: ActiveRequest = { prefix, append, controller: new AbortController() };
    active.current = request;
    setError(null);
    setLoadingMore(append);
    if (!append) {
      setSubmittedQuery(prefix);
      setItems([]);
      setNextCursor(null);
      setState("loading");
    }

    try {
      const fetchPage = kind === "users" ? getDirectoryUsers : getDirectoryChannels;
      const data = await fetchPage(
        { prefix, limit: 50, ...(append && nextCursor ? { after: nextCursor } : {}) },
        { signal: request.controller.signal }
      );
      // A cancelled fetch may still settle; only the current submission owns UI state.
      if (active.current !== request) return;
      setItems((previous) => {
        if (!append) return data.items;
        const seen = new Set(previous.map((item) => item.id));
        return [...previous, ...data.items.filter((item) => !seen.has(item.id))];
      });
      setNextCursor(data.next_cursor);
      setState(append || data.items.length > 0 ? "ready" : "empty");
    } catch (cause) {
      if (active.current !== request) return;
      setError(getUiErrorKey(cause, "directory"));
      if (!append) setState("error");
    } finally {
      if (active.current === request) {
        active.current = null;
        setLoadingMore(false);
      }
    }
  }

  return {
    submittedQuery,
    items,
    state,
    nextCursor,
    loadingMore,
    error,
    search: (prefix: string) => search(prefix),
    loadMore: () => search(submittedQuery, true)
  };
}
