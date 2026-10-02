import { fireEvent, screen, waitFor } from "@testing-library/react";
import { LocaleTestControls } from "@/test/locale-controls";
import { createLocaleRenderer } from "@/test/render-with-locale";
const render = createLocaleRenderer("tr", "admin");
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ChannelAdmin } from "@/features/channels/channel-admin";
import { ApiClientError } from "@/lib/api-client";
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
  it("preserves a pending mutation and retranslates its eventual error", async () => {
    let rejectAdd!: (error: unknown) => void;
    channelApiMocks.listChannels.mockResolvedValue([]);
    channelApiMocks.addChannel.mockImplementation(
      () =>
        new Promise((_resolve, reject) => {
          rejectAdd = reject;
        })
    );
    render(
      <>
        <LocaleTestControls />
        <ChannelAdmin />
      </>
    );
    await screen.findByText("Henüz takip edilen kanal yok.");
    fireEvent.change(screen.getByLabelText("Kanal slug/nickname"), { target: { value: "Heaven" } });
    fireEvent.click(screen.getByRole("button", { name: "Ekle" }));
    fireEvent.click(screen.getByRole("button", { name: "Switch to en" }));
    expect(await screen.findByRole("button", { name: "Add" })).toBeDisabled();
    expect(screen.getByLabelText("Channel slug/nickname")).toHaveValue("Heaven");
    rejectAdd(new ApiClientError(409, { detail: "RAW INTERNAL DETAIL" }));
    expect(
      await screen.findByText("This change conflicts with an existing record.")
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Switch to tr" }));
    await waitFor(() =>
      expect(
        screen.queryByText("This change conflicts with an existing record.")
      ).not.toBeInTheDocument()
    );
    expect(screen.queryByText("RAW INTERNAL DETAIL")).not.toBeInTheDocument();
    expect(channelApiMocks.addChannel).toHaveBeenCalledTimes(1);
    expect(channelApiMocks.listChannels).toHaveBeenCalledTimes(1);
  });
  it("keeps a channel draft and loaded list when switching languages", async () => {
    channelApiMocks.listChannels.mockResolvedValue([channelFixture()]);
    render(
      <>
        <LocaleTestControls />
        <ChannelAdmin />
      </>
    );
    await screen.findAllByText("#hype");
    fireEvent.change(screen.getByLabelText("Kanal slug/nickname"), { target: { value: "Heaven" } });
    fireEvent.click(screen.getByRole("button", { name: "Switch to de" }));
    expect(await screen.findByLabelText("Kanal-Slug/Nickname")).toHaveValue("Heaven");
    expect(screen.getAllByText("#hype").length).toBeGreaterThan(0);
    expect(channelApiMocks.listChannels).toHaveBeenCalledTimes(1);
    expect(channelApiMocks.addChannel).not.toHaveBeenCalled();
  });
  beforeEach(() => {
    channelApiMocks.addChannel.mockReset();
    channelApiMocks.listChannels.mockReset();
    channelApiMocks.removeChannel.mockReset();
  });

  it("lists followed channels with Kick metadata", async () => {
    channelApiMocks.listChannels.mockResolvedValue([channelFixture()]);

    render(<ChannelAdmin />);

    expect(await screen.findAllByText("hype")).not.toHaveLength(0);
    expect(screen.getAllByText("#hype")).not.toHaveLength(0);
    for (const link of screen.getAllByRole("link", { name: "hype #hype" })) {
      expect(link).toHaveAttribute("href", "/channels/hype");
    }
    expect(screen.getAllByText("Aktif")).not.toHaveLength(0);
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
    expect(await screen.findAllByText("#hype")).not.toHaveLength(0);
  });

  it("disables a followed channel", async () => {
    channelApiMocks.listChannels.mockResolvedValue([channelFixture()]);
    channelApiMocks.removeChannel.mockResolvedValue({
      ...channelFixture(),
      is_enabled: false
    });

    render(<ChannelAdmin />);

    expect(await screen.findAllByText("#hype")).not.toHaveLength(0);
    fireEvent.click(screen.getAllByRole("button", { name: /devre dışı bırak/i })[0]);

    await waitFor(() => expect(channelApiMocks.removeChannel).toHaveBeenCalledWith(1));
    expect(await screen.findAllByText("Pasif")).not.toHaveLength(0);
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
