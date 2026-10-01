import { describe, expect, it, vi } from "vitest";

import { getHomepage } from "@/features/landing/api";
import { ApiClientError, createApiClient } from "@/lib/api-client";

describe("getHomepage", () => {
  it.each([200, 202])(
    "reads a %s response through one uncached request with the abort signal",
    async (status) => {
      const response =
        status === 202
          ? { status: "initializing", retry_after_seconds: 5 }
          : {
              status: "ready",
              as_of: "2026-05-15T00:05:00Z",
              start: "2026-05-01T00:00:00Z",
              end: "2026-05-15T00:00:00Z",
              timezone: "UTC",
              stale: false,
              overview: {
                total_messages: 0,
                total_senders: 0,
                total_channels: 0,
                total_emote_usages: 0,
                first_message_at: null,
                latest_message_at: null
              },
              message_volume: [],
              top_channels: [],
              top_senders: [],
              top_emotes: []
            };
      const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify(response), { status }));
      const client = createApiClient({ baseUrl: "https://api.example.test", fetcher });
      const controller = new AbortController();

      await expect(getHomepage(controller.signal, client)).resolves.toEqual(response);

      expect(fetcher).toHaveBeenCalledExactlyOnceWith(
        "https://api.example.test/analytics/homepage",
        expect.objectContaining({ method: "GET", cache: "no-store", signal: controller.signal })
      );
    }
  );

  it("propagates HTTP errors instead of substituting empty analytics", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response("unavailable", { status: 503 }));
    const client = createApiClient({ baseUrl: "https://api.example.test", fetcher });

    await expect(getHomepage(undefined, client)).rejects.toBeInstanceOf(ApiClientError);
    expect(fetcher).toHaveBeenCalledTimes(1);
  });
});
