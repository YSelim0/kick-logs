import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LandingPage } from "@/features/landing/landing-page";

const analyticsMocks = vi.hoisted(() => ({
  getAnalyticsOverview: vi.fn(),
  getMessageVolume: vi.fn(),
  getTopChannels: vi.fn(),
  getTopEmotes: vi.fn(),
  getTopSenders: vi.fn()
}));

vi.mock("@/features/analytics/api", () => analyticsMocks);

describe("LandingPage", () => {
  beforeEach(() => {
    analyticsMocks.getAnalyticsOverview.mockReset();
    analyticsMocks.getMessageVolume.mockReset();
    analyticsMocks.getTopChannels.mockReset();
    analyticsMocks.getTopEmotes.mockReset();
    analyticsMocks.getTopSenders.mockReset();

    analyticsMocks.getAnalyticsOverview.mockResolvedValue({
      total_messages: 0,
      total_senders: 0,
      total_channels: 0,
      total_emote_usages: 0,
      first_message_at: null,
      latest_message_at: null
    });
    analyticsMocks.getMessageVolume.mockResolvedValue({ items: [] });
    analyticsMocks.getTopChannels.mockResolvedValue({ items: [] });
    analyticsMocks.getTopEmotes.mockResolvedValue({ items: [] });
    analyticsMocks.getTopSenders.mockResolvedValue({ items: [] });
  });

  it("renders the v2 hero, stats bar, and analytics panels", async () => {
    analyticsMocks.getAnalyticsOverview.mockResolvedValue({
      total_messages: 482,
      total_senders: 76,
      total_channels: 5,
      total_emote_usages: 314,
      first_message_at: "2026-05-01T10:00:00Z",
      latest_message_at: "2026-05-14T09:30:00Z"
    });
    analyticsMocks.getMessageVolume.mockResolvedValue({
      items: [
        { bucket_start: "2026-05-13T00:00:00Z", message_count: 40 },
        { bucket_start: "2026-05-14T00:00:00Z", message_count: 60 }
      ]
    });
    analyticsMocks.getTopChannels.mockResolvedValue({
      items: [
        {
          channel_id: 1,
          slug: "hype",
          display_name: "Hype",
          profile_image_url: null,
          banner_image_url: null,
          message_count: 500,
          first_message_at: "2026-05-01T10:00:00Z",
          latest_message_at: "2026-05-14T09:30:00Z"
        }
      ]
    });
    analyticsMocks.getTopEmotes.mockResolvedValue({
      items: [
        {
          id: "37226",
          name: "KEKW",
          token: "[emote:37226:KEKW]",
          image_url: "https://files.kick.com/emotes/37226/fullsize",
          usage_count: 99,
          message_count: 80
        }
      ]
    });
    analyticsMocks.getTopSenders.mockResolvedValue({
      items: [
        {
          sender_id: 1,
          kick_user_id: 10,
          username: "Yavuz",
          slug: "yavuz",
          profile_image_url: null,
          message_count: 120,
          first_message_at: "2026-05-01T10:00:00Z",
          latest_message_at: "2026-05-14T09:30:00Z"
        }
      ]
    });

    render(<LandingPage />);

    expect(
      await screen.findByRole("heading", { name: /kick chat için kalıcı log\./i })
    ).toBeInTheDocument();

    expect(screen.getByText("TOPLAM MESAJ")).toBeInTheDocument();
    expect(screen.getByText("KANAL")).toBeInTheDocument();
    expect(screen.getByText("KULLANICI")).toBeInTheDocument();
    expect(screen.getByText("EMOTE")).toBeInTheDocument();

    expect(screen.getByRole("heading", { name: "Mesaj hacmi" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Top kanallar" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Top kullanıcılar" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Top emoteler" })).toBeInTheDocument();

    expect(screen.getByText("Hype")).toBeInTheDocument();
    expect(screen.getByText("KEKW")).toBeInTheDocument();
    expect(screen.getByText("Yavuz")).toBeInTheDocument();
  });

  it("shows panel-level empty hints on a fresh install", async () => {
    render(<LandingPage />);

    expect(await screen.findByText("Kanal verisi henüz yok.")).toBeInTheDocument();
    expect(screen.getByText("Kullanıcı verisi henüz yok.")).toBeInTheDocument();
    expect(screen.getByText("Emote verisi henüz yok.")).toBeInTheDocument();
    expect(screen.getByText("Henüz veri yok.")).toBeInTheDocument();
  });

  it("renders v2 header and CTA links", async () => {
    render(<LandingPage />);

    await waitFor(() => expect(analyticsMocks.getAnalyticsOverview).toHaveBeenCalledTimes(1));

    expect(linkExists(/^search$/i, "/search")).toBe(true);
    expect(linkExists(/^admin$/i, "/admin")).toBe(true);
    expect(linkExists(/arama başlat/i, "/search")).toBe(true);
    expect(linkExists(/github/i, "https://github.com/YSelim0/kick-logs")).toBe(true);

    expect(screen.queryByRole("link", { name: /support/i })).toBeNull();
  });
});

function linkExists(name: RegExp, href: string) {
  return screen.getAllByRole("link", { name }).some((link) => link.getAttribute("href") === href);
}
