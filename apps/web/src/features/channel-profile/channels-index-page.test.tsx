import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ChannelsIndexPage } from "@/features/channel-profile/channels-index-page";
import type { TopChannelsResponse } from "@/types/api";

const analyticsMocks = vi.hoisted(() => ({
  getTopChannels: vi.fn()
}));

vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

vi.mock("@/features/analytics/api", () => analyticsMocks);

describe("ChannelsIndexPage", () => {
  beforeEach(() => {
    analyticsMocks.getTopChannels.mockReset();
    analyticsMocks.getTopChannels.mockResolvedValue(channelsFixture());
  });

  it("renders empty idle prompt on initial load without calling the API", () => {
    render(<ChannelsIndexPage />);

    expect(screen.getByText("Kanal bulmak için arama yapın")).toBeInTheDocument();
    expect(analyticsMocks.getTopChannels).not.toHaveBeenCalled();
  });

  it("renders search input with correct label", () => {
    render(<ChannelsIndexPage />);

    expect(screen.getByRole("searchbox", { name: /kanal ara/i })).toBeInTheDocument();
  });

  it("shows channel results after typing a query", async () => {
    const user = userEvent.setup({ delay: null });
    render(<ChannelsIndexPage />);

    const input = screen.getByRole("searchbox", { name: /kanal ara/i });
    await user.type(input, "hype");

    await waitFor(() =>
      expect(analyticsMocks.getTopChannels).toHaveBeenCalledWith(
        expect.objectContaining({ q: "hype", limit: 20 })
      )
    );

    expect(await screen.findByText("Hype")).toBeInTheDocument();
    expect(screen.getByText("#hype")).toBeInTheDocument();
    expect(screen.getByText("GameZone")).toBeInTheDocument();
  });

  it("shows empty state when API returns no results", async () => {
    analyticsMocks.getTopChannels.mockResolvedValue({ items: [] });

    const user = userEvent.setup({ delay: null });
    render(<ChannelsIndexPage />);

    await user.type(screen.getByRole("searchbox", { name: /kanal ara/i }), "xyz");

    await waitFor(() =>
      expect(screen.getByText(/"xyz" için kanal bulunamadı\./)).toBeInTheDocument()
    );
  });

  it("shows error state when API fails", async () => {
    analyticsMocks.getTopChannels.mockRejectedValue(new Error("network error"));

    const user = userEvent.setup({ delay: null });
    render(<ChannelsIndexPage />);

    await user.type(screen.getByRole("searchbox", { name: /kanal ara/i }), "err");

    await waitFor(() => expect(screen.getByText(/sonuçlar alınamadı/i)).toBeInTheDocument());
  });

  it("each channel row links to the channel profile page", async () => {
    const user = userEvent.setup({ delay: null });
    render(<ChannelsIndexPage />);

    await user.type(screen.getByRole("searchbox", { name: /kanal ara/i }), "hype");

    await waitFor(() => expect(screen.getByText("Hype")).toBeInTheDocument());

    const links = screen.getAllByRole("link");
    const hypeLink = links.find((l) => l.getAttribute("href") === "/channels/hype");
    expect(hypeLink).toBeDefined();
  });
});

function channelsFixture(): TopChannelsResponse {
  return {
    items: [
      {
        channel_id: 1,
        slug: "hype",
        display_name: "Hype",
        profile_image_url: null,
        banner_image_url: null,
        message_count: 1500,
        first_message_at: "2026-05-01T10:00:00Z",
        latest_message_at: "2026-05-14T09:30:00Z"
      },
      {
        channel_id: 2,
        slug: "gamezone",
        display_name: "GameZone",
        profile_image_url: null,
        banner_image_url: null,
        message_count: 800,
        first_message_at: "2026-05-02T10:00:00Z",
        latest_message_at: "2026-05-13T08:00:00Z"
      }
    ]
  };
}
