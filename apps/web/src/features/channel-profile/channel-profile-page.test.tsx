import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ChannelProfilePage } from "@/features/channel-profile/channel-profile-page";
import { ApiClientError } from "@/lib/api-client";
import type { ChannelProfile } from "@/types/api";

const profileMocks = vi.hoisted(() => ({
  buildChannelSubscribersExportUrl: vi.fn(),
  getChannelProfile: vi.fn(),
  getChannelSubscribers: vi.fn(),
  getChannelSubscriptionSummary: vi.fn()
}));

vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

vi.mock("@/features/channel-profile/api", () => profileMocks);

describe("ChannelProfilePage", () => {
  beforeEach(() => {
    profileMocks.getChannelProfile.mockReset();
    profileMocks.getChannelSubscribers.mockReset();
    profileMocks.getChannelSubscriptionSummary.mockReset();
    profileMocks.buildChannelSubscribersExportUrl.mockReset();
    profileMocks.getChannelProfile.mockResolvedValue(profileFixture());
    profileMocks.getChannelSubscribers.mockResolvedValue({
      channel_slug: "hype",
      gift_only: false,
      count: 1,
      limit: 50,
      offset: 0,
      items: [
        {
          subscriber_kick_user_id: 123,
          username: "subscriber_one",
          slug: "subscriber-one",
          profile_image_url: "",
          is_gift: false,
          started_at: "2026-06-01T10:00:00Z",
          expires_at: "2026-07-01T10:00:00Z"
        }
      ]
    });
    profileMocks.getChannelSubscriptionSummary.mockResolvedValue({
      channel_slug: "hype",
      active_count: 42,
      active_gifted_count: 5,
      latest_event_at: null
    });
    profileMocks.buildChannelSubscribersExportUrl.mockImplementation(
      (slug: string, giftOnly: boolean, format: string) =>
        `http://localhost:8000/channels/${slug}/subscribers/export?format=${format}${giftOnly ? "&gift_only=true" : ""}`
    );
  });

  it("renders channel analytics and latest messages", async () => {
    render(<ChannelProfilePage slug="hype" />);

    await waitFor(() => expect(profileMocks.getChannelProfile).toHaveBeenCalledWith("hype"));
    expect(await screen.findByRole("heading", { name: "Hype" })).toBeInTheDocument();
    expect(screen.getByText("MESAJ")).toBeInTheDocument();
    expect(screen.getByText("KULLANICI")).toBeInTheDocument();
    expect(screen.getByText("AKTİF ABONE")).toBeInTheDocument();
    expect(screen.getByText("HEDİYE ABONE")).toBeInTheDocument();
    expect(screen.getByText("@alpha")).toBeInTheDocument();
    expect(screen.getByText("KEKW")).toBeInTheDocument();
    expect(screen.getByText("latest channel message")).toBeInTheDocument();
    expect(screen.queryByText(/channel id/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/chatroom id/i)).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: /kanalda ara/i })).toHaveAttribute(
      "href",
      "/search?channel=hype"
    );
    expect(screen.getByRole("link", { name: /kick hesabını ziyaret/i })).toHaveAttribute(
      "href",
      "https://kick.com/hype"
    );
    expect(screen.getByRole("link", { name: /@alpha/ })).toHaveAttribute("href", "/users/alpha");
  });

  it("opens the active subscriber modal from the stat cell", async () => {
    render(<ChannelProfilePage slug="hype" />);

    fireEvent.click(await screen.findByRole("button", { name: "Aktif aboneleri görüntüle" }));

    await waitFor(() =>
      expect(profileMocks.getChannelSubscribers).toHaveBeenCalledWith("hype", {
        limit: 50,
        offset: 0,
        gift_only: false
      })
    );
    expect(await screen.findByRole("heading", { name: "Aktif aboneler" })).toBeInTheDocument();
    expect(screen.getByText("subscriber_one")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "subscriber_one" })).toHaveAttribute(
      "href",
      "/users/subscriber-one"
    );
  });

  it("opens the gifted subscriber modal and export menu", async () => {
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);

    render(<ChannelProfilePage slug="hype" />);

    fireEvent.click(
      await screen.findByRole("button", { name: "Hediye aktif aboneleri görüntüle" })
    );

    await waitFor(() =>
      expect(profileMocks.getChannelSubscribers).toHaveBeenCalledWith("hype", {
        limit: 50,
        offset: 0,
        gift_only: true
      })
    );
    expect(
      await screen.findByRole("heading", { name: "Hediye aktif aboneler" })
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Abone listesini indir" }));
    fireEvent.click(screen.getByRole("button", { name: "TXT indir" }));

    expect(openSpy).toHaveBeenCalledWith(
      "http://localhost:8000/channels/hype/subscribers/export?format=txt&gift_only=true",
      "_blank",
      "noopener,noreferrer"
    );

    openSpy.mockRestore();
  });

  it("renders subscriber empty state", async () => {
    profileMocks.getChannelSubscribers.mockResolvedValue({
      channel_slug: "hype",
      gift_only: false,
      count: 0,
      limit: 50,
      offset: 0,
      items: []
    });

    render(<ChannelProfilePage slug="hype" />);

    fireEvent.click(await screen.findByRole("button", { name: "Aktif aboneleri görüntüle" }));

    expect(
      await screen.findByText("Bu kanal için henüz aktif abonelik kaydı yok.")
    ).toBeInTheDocument();
  });

  it("renders loading state", () => {
    profileMocks.getChannelProfile.mockReturnValue(new Promise(() => undefined));

    render(<ChannelProfilePage slug="hype" />);

    expect(screen.getByText("Kanal profili yükleniyor...")).toBeInTheDocument();
  });

  it("renders empty profile sections", async () => {
    profileMocks.getChannelProfile.mockResolvedValue({
      ...profileFixture(),
      overview: {
        total_messages: 0,
        total_senders: 0,
        total_channels: 0,
        total_emote_usages: 0,
        first_message_at: null,
        latest_message_at: null
      },
      message_volume: [],
      top_senders: [],
      top_emotes: [],
      latest_messages: []
    });

    render(<ChannelProfilePage slug="empty" />);

    expect(await screen.findByText("Mesaj hacmi verisi henüz yok.")).toBeInTheDocument();
    expect(screen.getByText("Kullanıcı aktivitesi henüz yok.")).toBeInTheDocument();
    expect(screen.getByText("Emote verisi henüz yok.")).toBeInTheDocument();
    expect(screen.getByText("Son mesaj bulunamadı.")).toBeInTheDocument();
  });

  it("renders not-found state for unknown channels", async () => {
    profileMocks.getChannelProfile.mockRejectedValue(
      new ApiClientError(404, { detail: "missing" })
    );

    render(<ChannelProfilePage slug="missing" />);

    expect(await screen.findByText("Kanal bulunamadı.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /search'e dön/i })).toHaveAttribute("href", "/search");
  });
});

function profileFixture(): ChannelProfile {
  return {
    channel: {
      id: 1,
      kick_channel_id: 100,
      kick_chatroom_id: 200,
      slug: "hype",
      display_name: "Hype",
      profile_image_url: "https://example.com/hype.png",
      banner_image_url: "https://example.com/hype-banner.png",
      is_enabled: true
    },
    overview: {
      total_messages: 12,
      total_senders: 2,
      total_channels: 1,
      total_emote_usages: 4,
      first_message_at: "2026-05-01T10:00:00Z",
      latest_message_at: "2026-05-14T09:30:00Z"
    },
    message_volume: [
      { bucket_start: "2026-05-13T00:00:00Z", message_count: 5 },
      { bucket_start: "2026-05-14T00:00:00Z", message_count: 7 }
    ],
    top_senders: [
      {
        sender_id: 1,
        kick_user_id: 10,
        username: "alpha",
        slug: "alpha",
        profile_image_url: null,
        message_count: 8,
        first_message_at: "2026-05-01T10:00:00Z",
        latest_message_at: "2026-05-14T09:30:00Z"
      }
    ],
    top_emotes: [
      {
        id: "37226",
        name: "KEKW",
        token: "[emote:37226:KEKW]",
        image_url: "https://files.kick.com/emotes/37226/fullsize",
        usage_count: 4,
        message_count: 3
      }
    ],
    latest_messages: [
      {
        id: 1,
        kick_message_id: "message-1",
        chatroom_id: 200,
        content: "latest channel message",
        message_type: "message",
        sender_username_snapshot: "alpha",
        sender_slug_snapshot: "alpha",
        sender_color_snapshot: null,
        sender_badges: [],
        emotes: [],
        reply_metadata: {},
        thread_parent_id: null,
        message_created_at: "2026-05-14T09:30:00Z",
        ingested_at: "2026-05-14T09:30:01Z",
        sender: {
          id: 1,
          kick_user_id: 10,
          username: "alpha",
          slug: "alpha",
          profile_image_url: null
        },
        channel: {
          id: 1,
          slug: "hype",
          display_name: "Hype",
          profile_image_url: "https://example.com/hype.png",
          banner_image_url: "https://example.com/hype-banner.png"
        }
      }
    ]
  };
}
