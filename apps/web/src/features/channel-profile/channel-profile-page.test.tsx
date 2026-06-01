import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ChannelProfilePage } from "@/features/channel-profile/channel-profile-page";
import { ApiClientError } from "@/lib/api-client";
import type { ChannelProfile } from "@/types/api";

const profileMocks = vi.hoisted(() => ({
  getChannelProfile: vi.fn(),
  getChannelSubscriptionSummary: vi.fn()
}));

vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

vi.mock("@/features/channel-profile/api", () => profileMocks);

describe("ChannelProfilePage", () => {
  beforeEach(() => {
    profileMocks.getChannelProfile.mockReset();
    profileMocks.getChannelSubscriptionSummary.mockReset();
    profileMocks.getChannelProfile.mockResolvedValue(profileFixture());
    profileMocks.getChannelSubscriptionSummary.mockResolvedValue({
      channel_slug: "hype",
      active_count: 42,
      active_gifted_count: 5,
      latest_event_at: null
    });
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
