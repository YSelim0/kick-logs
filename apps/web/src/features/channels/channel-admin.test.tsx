import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ChannelAdmin } from "@/features/channels/channel-admin";
import type { Channel } from "@/types/api";

const channelApiMocks = vi.hoisted(() => ({
  addChannel: vi.fn(),
  listChannels: vi.fn(),
  removeChannel: vi.fn()
}));

vi.mock("@/features/channels/api", () => ({
  addChannel: channelApiMocks.addChannel,
  listChannels: channelApiMocks.listChannels,
  removeChannel: channelApiMocks.removeChannel
}));

describe("ChannelAdmin", () => {
  beforeEach(() => {
    channelApiMocks.addChannel.mockReset();
    channelApiMocks.listChannels.mockReset();
    channelApiMocks.removeChannel.mockReset();
  });

  it("lists followed channels with Kick metadata", async () => {
    channelApiMocks.listChannels.mockResolvedValue([channelFixture()]);

    render(<ChannelAdmin />);

    expect(await screen.findByText("hype")).toBeInTheDocument();
    expect(screen.getByText("#hype")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "hype #hype" })).toHaveAttribute(
      "href",
      "/channels/hype"
    );
    expect(screen.getByText("Aktif")).toBeInTheDocument();
  });

  it("adds a channel by slug", async () => {
    const channel = channelFixture();
    channelApiMocks.listChannels.mockResolvedValue([]);
    channelApiMocks.addChannel.mockResolvedValue(channel);

    render(<ChannelAdmin />);

    expect(await screen.findByText("Henüz takip edilen kanal yok.")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Kanal slug/nickname"), {
      target: { value: " hype " }
    });
    fireEvent.click(screen.getByRole("button", { name: /ekle/i }));

    await waitFor(() => expect(channelApiMocks.addChannel).toHaveBeenCalledWith({ slug: "hype" }));
    expect(await screen.findByText("#hype")).toBeInTheDocument();
  });

  it("disables a followed channel", async () => {
    channelApiMocks.listChannels.mockResolvedValue([channelFixture()]);
    channelApiMocks.removeChannel.mockResolvedValue({
      ...channelFixture(),
      is_enabled: false
    });

    render(<ChannelAdmin />);

    expect(await screen.findByText("#hype")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /devre dışı bırak/i }));

    await waitFor(() => expect(channelApiMocks.removeChannel).toHaveBeenCalledWith(1));
    expect(await screen.findByText("Pasif")).toBeInTheDocument();
  });
});

function channelFixture(): Channel {
  return {
    id: 1,
    kick_channel_id: 100,
    kick_chatroom_id: 200,
    slug: "hype",
    display_name: "hype",
    profile_image_url: null,
    banner_image_url: null,
    is_enabled: true,
    message_count: 0,
    last_message_at: null
  };
}
