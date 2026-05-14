import { render, screen } from "@testing-library/react";
import { createRef } from "react";
import { describe, expect, it, vi } from "vitest";

import { MessageList } from "@/features/search/message-list";
import type { Message } from "@/types/api";

describe("MessageList", () => {
  it("renders reply context above the current message", () => {
    renderMessageList([
      {
        ...messageFixture(),
        message_type: "reply",
        content: "current reply content",
        reply_metadata: {
          original_sender: {
            id: 97891494,
            username: "Cansu98xx",
            slug: "cansu_98xx"
          },
          original_message: {
            id: "1be196b8-55c7-4980-8022-a1112723acea",
            content: "senin saat ne saati 5dk 1 saatmiş"
          },
          message_ref: "1778535344619"
        }
      }
    ]);

    expect(screen.getByText("@Cansu98xx:")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "@Cansu98xx:" })).toHaveAttribute(
      "href",
      "/users/cansu-98xx"
    );
    expect(screen.getByText(/senin saat ne saati/)).toBeInTheDocument();
    expect(screen.getByText("current reply content")).toBeInTheDocument();
    expect(screen.getByTitle("@Cansu98xx: senin saat ne saati 5dk 1 saatmiş")).toBeInTheDocument();
  });

  it("does not render reply context for normal messages", () => {
    renderMessageList([messageFixture()]);

    expect(screen.queryByText("@Cansu98xx:")).not.toBeInTheDocument();
    expect(screen.getByText("hello")).toBeInTheDocument();
  });

  it("links sender name and avatar to the public user profile", () => {
    renderMessageList([
      {
        ...messageFixture(),
        sender: {
          ...messageFixture().sender,
          profile_image_url: "https://example.com/yavuz.png"
        }
      }
    ]);

    expect(screen.getByRole("link", { name: "yavuz" })).toHaveAttribute(
      "href",
      "/users/yavuz-user"
    );
    expect(screen.getByRole("link", { name: "yavuz profil" })).toHaveAttribute(
      "href",
      "/users/yavuz-user"
    );
  });
});

function renderMessageList(messages: Message[]) {
  return render(
    <MessageList
      error={null}
      hasMore={false}
      hasSearched={true}
      isInitialLoading={false}
      isLoadingMore={false}
      messages={messages}
      onRetry={vi.fn()}
      sentinelRef={createRef<HTMLDivElement>()}
    />
  );
}

function messageFixture(): Message {
  return {
    id: 1,
    kick_message_id: "kick-1",
    chatroom_id: 10,
    content: "hello",
    message_type: "message",
    sender_username_snapshot: "yavuz",
    sender_slug_snapshot: "yavuz",
    sender_color_snapshot: null,
    sender_badges: [],
    emotes: [],
    reply_metadata: {},
    thread_parent_id: null,
    message_created_at: "2026-05-09T02:41:00Z",
    ingested_at: "2026-05-09T02:41:01Z",
    sender: {
      id: 1,
      kick_user_id: 1,
      username: "yavuz",
      slug: "yavuz_user",
      profile_image_url: null
    },
    channel: {
      id: 1,
      slug: "hype",
      display_name: "hype",
      profile_image_url: null,
      banner_image_url: null
    }
  };
}
