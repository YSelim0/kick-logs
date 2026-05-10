import { describe, expect, it } from "vitest";

import {
  EMPTY_SEARCH_STATE,
  appendUniqueMessages,
  getActiveFilters,
  readSearchState,
  searchStateToMessageParams,
  searchStateToUrlSearchParams
} from "@/features/search/search-params";
import type { Message } from "@/types/api";

describe("search params", () => {
  it("maps filled form fields to backend query params and omits empty values", () => {
    expect(
      searchStateToMessageParams({
        sender: " yavuz ",
        channel: "",
        q: " selam ",
        start: "2026-05-02T02:43",
        end: ""
      })
    ).toEqual({
      sender: "yavuz",
      q: "selam",
      start: "2026-05-02T02:43"
    });
  });

  it("keeps empty filters as latest-message search", () => {
    expect(searchStateToMessageParams(EMPTY_SEARCH_STATE)).toEqual({});
    expect(searchStateToUrlSearchParams(EMPTY_SEARCH_STATE).toString()).toBe("");
  });

  it("reads supported filters from URL state", () => {
    const state = readSearchState(
      new URLSearchParams("sender=yavuz&channel=hype&q=hello&start=2026-05-02T02:43")
    );

    expect(state).toMatchObject({
      sender: "yavuz",
      channel: "hype",
      q: "hello",
      start: "2026-05-02T02:43"
    });
  });

  it("creates active filter labels", () => {
    expect(
      getActiveFilters({
        ...EMPTY_SEARCH_STATE,
        sender: "yavuz",
        channel: "hype"
      })
    ).toEqual([
      { key: "sender", label: "Kullanıcı", value: "yavuz" },
      { key: "channel", label: "Kanal", value: "hype" }
    ]);
  });

  it("appends infinite-scroll pages without duplicating existing rows", () => {
    const first = messageFixture(1);
    const second = messageFixture(2);

    expect(appendUniqueMessages([first], [first, second]).map((message) => message.id)).toEqual([
      1,
      2
    ]);
  });
});

function messageFixture(id: number): Message {
  return {
    id,
    kick_message_id: `kick-${id}`,
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
