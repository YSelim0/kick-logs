import { describe, expect, it, vi } from "vitest";

import { getDirectoryChannels, getDirectoryUsers } from "@/features/directory/api";
import { createApiClient } from "@/lib/api-client";

describe.each([
  ["users", getDirectoryUsers],
  ["channels", getDirectoryChannels]
] as const)("%s directory API", (kind, search) => {
  it("encodes literal prefixes and keyset cursors, forwarding cancellation", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify({ items: [], next_cursor: null })));
    const client = createApiClient({ baseUrl: "https://api.example.com", fetcher });
    const signal = new AbortController().signal;
    expect(
      await search(
        { prefix: "Yav_%' OR 1=1", limit: 50, after: "opaque+cursor" },
        { signal },
        client
      )
    ).toEqual({ items: [], next_cursor: null });
    const [raw, init] = fetcher.mock.calls[0];
    const url = new URL(raw);
    expect(url.pathname).toBe(`/directory/${kind}`);
    expect(Object.fromEntries(url.searchParams)).toEqual({
      prefix: "Yav_%' OR 1=1",
      limit: "50",
      after: "opaque+cursor"
    });
    expect(init.signal).toBe(signal);
  });

  it("surfaces API errors instead of treating them as empty results", async () => {
    const client = createApiClient({
      baseUrl: "https://api.example.com",
      fetcher: vi
        .fn()
        .mockResolvedValue(new Response('{"detail":"Too many requests."}', { status: 429 }))
    });
    await expect(search({ prefix: "ya" }, {}, client)).rejects.toMatchObject({ status: 429 });
  });
});
