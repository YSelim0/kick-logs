import { describe, expect, it, vi } from "vitest";

import {
  getAnalyticsOverview,
  getMessageVolume,
  getTopChannels,
  getTopEmotes,
  getTopSenders
} from "@/features/analytics/api";
import type { ApiClient } from "@/lib/api-client";

describe("analytics api", () => {
  it("maps overview query params", async () => {
    const client = fakeClient();

    await getAnalyticsOverview(
      {
        start: "2026-05-01T00:00:00Z",
        end: "2026-05-02T00:00:00Z",
        channel: "hype",
        sender: "yavuz"
      },
      client
    );

    expect(client.get).toHaveBeenCalledWith("/analytics/overview", {
      start: "2026-05-01T00:00:00Z",
      end: "2026-05-02T00:00:00Z",
      channel: "hype",
      sender: "yavuz",
      limit: undefined,
      bucket: undefined
    });
  });

  it("maps message volume bucket params", async () => {
    const client = fakeClient();

    await getMessageVolume({ channel: "hype", bucket: "hour" }, client);

    expect(client.get).toHaveBeenCalledWith("/analytics/message-volume", {
      start: undefined,
      end: undefined,
      channel: "hype",
      sender: undefined,
      limit: undefined,
      bucket: "hour"
    });
  });

  it("maps top list limits and scopes", async () => {
    const client = fakeClient();

    await getTopSenders({ channel: "hype", limit: 5 }, client);
    await getTopChannels({ sender: "yavuz", limit: 3 }, client);
    await getTopEmotes({ channel: "hype", sender: "yavuz", limit: 10 }, client);

    expect(client.get).toHaveBeenNthCalledWith(1, "/analytics/top-senders", {
      start: undefined,
      end: undefined,
      channel: "hype",
      sender: undefined,
      limit: 5,
      bucket: undefined
    });
    expect(client.get).toHaveBeenNthCalledWith(2, "/analytics/top-channels", {
      start: undefined,
      end: undefined,
      channel: undefined,
      sender: "yavuz",
      limit: 3,
      bucket: undefined
    });
    expect(client.get).toHaveBeenNthCalledWith(3, "/analytics/top-emotes", {
      start: undefined,
      end: undefined,
      channel: "hype",
      sender: "yavuz",
      limit: 10,
      bucket: undefined
    });
  });
});

function fakeClient() {
  return {
    get: vi.fn().mockResolvedValue({ items: [] })
  } as unknown as ApiClient;
}
