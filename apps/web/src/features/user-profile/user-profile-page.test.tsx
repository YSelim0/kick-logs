import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UserProfilePage } from "@/features/user-profile/user-profile-page";
import { ApiClientError } from "@/lib/api-client";
import type { UserProfile } from "@/types/api";

const profileMocks = vi.hoisted(() => ({
  getUserProfile: vi.fn()
}));

vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

vi.mock("@/features/user-profile/api", () => profileMocks);

describe("UserProfilePage", () => {
  beforeEach(() => {
    profileMocks.getUserProfile.mockReset();
    profileMocks.getUserProfile.mockResolvedValue(profileFixture());
  });

  it("renders profile analytics and latest messages", async () => {
    render(<UserProfilePage slug="yavuz" />);

    await waitFor(() => expect(profileMocks.getUserProfile).toHaveBeenCalledWith("yavuz"));
    expect(await screen.findByRole("heading", { name: "Yavuz" })).toBeInTheDocument();
    expect(screen.getByText("@yavuz")).toBeInTheDocument();
    expect(screen.queryByText((content) => content.includes("kanal 2 ·"))).not.toBeInTheDocument();
    expect(screen.getByText("MESAJ")).toBeInTheDocument();
    expect(screen.getByText("#hype")).toBeInTheDocument();
    expect(screen.getByText("KEKW")).toBeInTheDocument();
    expect(screen.getByText("↳")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "@reply_user:" })).toHaveAttribute(
      "href",
      "/users/reply-user"
    );
    expect(screen.getByText("older profile context")).toBeInTheDocument();
    expect(screen.getByText("hello profile message")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /mesajlarda ara/i })).toHaveAttribute(
      "href",
      "/search?sender=yavuz"
    );
    expect(screen.getByRole("link", { name: /kick hesabını ziyaret/i })).toHaveAttribute(
      "href",
      "https://kick.com/yavuz"
    );
  });

  it("renders not-found state for unknown senders", async () => {
    profileMocks.getUserProfile.mockRejectedValue(new ApiClientError(404, { detail: "missing" }));

    render(<UserProfilePage slug="missing" />);

    expect(await screen.findByText("Kullanıcı bulunamadı.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /search'e dön/i })).toHaveAttribute("href", "/search");
  });
});

function profileFixture(): UserProfile {
  return {
    sender: {
      id: 1,
      kick_user_id: 10,
      username: "Yavuz",
      slug: "yavuz",
      profile_image_url: "https://example.com/yavuz.png"
    },
    overview: {
      total_messages: 12,
      total_senders: 1,
      total_channels: 2,
      total_emote_usages: 4,
      first_message_at: "2026-05-01T10:00:00Z",
      latest_message_at: "2026-05-14T09:30:00Z"
    },
    message_volume: [
      { bucket_start: "2026-05-13T00:00:00Z", message_count: 5 },
      { bucket_start: "2026-05-14T00:00:00Z", message_count: 7 }
    ],
    top_channels: [
      {
        channel_id: 1,
        slug: "hype",
        display_name: "Hype",
        profile_image_url: null,
        banner_image_url: null,
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
        chatroom_id: 100,
        content: "hello profile message",
        message_type: "reply",
        sender_username_snapshot: "Yavuz",
        sender_slug_snapshot: "yavuz",
        sender_color_snapshot: null,
        sender_badges: [],
        emotes: [],
        reply_metadata: {
          original_sender: {
            username: "reply_user",
            slug: "reply_user"
          },
          original_message: {
            id: "parent-message-1",
            content: "older profile context"
          }
        },
        thread_parent_id: null,
        message_created_at: "2026-05-14T09:30:00Z",
        ingested_at: "2026-05-14T09:30:01Z",
        sender: {
          id: 1,
          kick_user_id: 10,
          username: "Yavuz",
          slug: "yavuz",
          profile_image_url: "https://example.com/yavuz.png"
        },
        channel: {
          id: 1,
          slug: "hype",
          display_name: "Hype",
          profile_image_url: null,
          banner_image_url: null
        }
      }
    ]
  };
}
