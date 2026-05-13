import { describe, expect, it } from "vitest";

import { getReplyContext } from "@/features/search/reply-metadata";
import type { Message } from "@/types/api";

describe("reply metadata", () => {
  it("extracts Kick reply context from the observed metadata shape", () => {
    expect(
      getReplyContext({
        ...messageFixture(),
        message_type: "reply",
        reply_metadata: {
          original_sender: {
            id: 97891494,
            username: "Cansu98xx",
            slug: "cansu98xx"
          },
          original_message: {
            id: "1be196b8-55c7-4980-8022-a1112723acea",
            content: "senin saat ne saati 5dk 1 saatmiş"
          },
          message_ref: "1778535344619"
        }
      })
    ).toEqual({
      senderUsername: "Cansu98xx",
      senderSlug: "cansu98xx",
      messageId: "1be196b8-55c7-4980-8022-a1112723acea",
      content: "senin saat ne saati 5dk 1 saatmiş"
    });
  });

  it("falls back to a username-derived slug when Kick reply metadata has no sender slug", () => {
    expect(
      getReplyContext({
        ...messageFixture(),
        message_type: "reply",
        reply_metadata: {
          original_sender: {
            id: 97891494,
            username: "Cansu98xx"
          },
          original_message: {
            id: "1be196b8-55c7-4980-8022-a1112723acea",
            content: "previous content"
          }
        }
      })
    ).toMatchObject({
      senderUsername: "Cansu98xx",
      senderSlug: "cansu98xx"
    });
  });

  it("does not treat normal messages as replies", () => {
    expect(
      getReplyContext({
        ...messageFixture(),
        reply_metadata: {
          original_sender: { username: "Cansu98xx" },
          original_message: {
            id: "1be196b8-55c7-4980-8022-a1112723acea",
            content: "previous"
          }
        }
      })
    ).toBeNull();
  });

  it("ignores malformed reply metadata", () => {
    expect(
      getReplyContext({
        ...messageFixture(),
        message_type: "reply",
        reply_metadata: {
          original_sender: { username: "Cansu98xx" },
          original_message: { content: "missing id" }
        }
      })
    ).toBeNull();
  });
});

function messageFixture(): Message {
  return {
    id: 1,
    kick_message_id: "kick-1",
    chatroom_id: 10,
    content: "current reply content",
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
      slug: "yavuz",
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
