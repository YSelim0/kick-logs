import { describe, expect, it } from "vitest";

import {
  EMPTY_SEARCH_STATE,
  appendUniqueMessages,
  getActiveFilters,
  getDefaultSearchState,
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
        end: "2026-05-09T02:43"
      })
    ).toEqual({
      sender: "yavuz",
      q: "selam",
      start: new Date("2026-05-02T02:43").toISOString(),
      end: localDateTimeEndOfMinuteIso("2026-05-09T02:43")
    });
  });

  it("keeps URL date params in datetime-local format", () => {
    const params = searchStateToUrlSearchParams({
      sender: " yavuz ",
      channel: "",
      q: "",
      start: "2026-05-02T02:43",
      end: "2026-05-09T02:43"
    });

    expect(params.get("sender")).toBe("yavuz");
    expect(params.get("start")).toBe("2026-05-02T02:43");
    expect(params.get("end")).toBe("2026-05-09T02:43");
  });

  it("keeps empty filters as latest-message search", () => {
    expect(searchStateToMessageParams(EMPTY_SEARCH_STATE)).toEqual({});
    expect(searchStateToUrlSearchParams(EMPTY_SEARCH_STATE).toString()).toBe("");
  });

  it("defaults the date range to the previous seven days", () => {
    expect(getDefaultSearchState(new Date("2026-05-10T15:30:45"))).toMatchObject({
      start: "2026-05-03T15:30",
      end: "2026-05-10T15:30"
    });
  });

  it("fills missing URL date filters with the default search range", () => {
    const state = readSearchState(
      new URLSearchParams("sender=yavuz"),
      new Date("2026-05-10T15:30:45")
    );

    expect(state).toMatchObject({
      sender: "yavuz",
      start: "2026-05-03T15:30",
      end: "2026-05-10T15:30"
    });
  });

  it("reads supported filters from URL state", () => {
    const state = readSearchState(
      new URLSearchParams("sender=yavuz&channel=hype&q=hello&start=2026-05-02T02:43"),
      new Date("2026-05-10T15:30:45")
    );

    expect(state).toMatchObject({
      sender: "yavuz",
      channel: "hype",
      q: "hello",
      start: "2026-05-02T02:43",
      end: "2026-05-10T15:30"
    });
  });

  it("normalizes ISO URL date filters back to local input values", () => {
    const isoStart = new Date("2026-05-02T02:43").toISOString();
    const state = readSearchState(
      new URLSearchParams({ start: isoStart }),
      new Date("2026-05-10T15:30:45")
    );

    expect(state.start).toBe(toLocalMinuteValue(new Date(isoStart)));
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
      1, 2
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

function localDateTimeEndOfMinuteIso(value: string) {
  const date = new Date(value);
  date.setSeconds(59, 999);
  return date.toISOString();
}

function toLocalMinuteValue(date: Date) {
  const year = date.getFullYear();
  const month = padDatePart(date.getMonth() + 1);
  const day = padDatePart(date.getDate());
  const hours = padDatePart(date.getHours());
  const minutes = padDatePart(date.getMinutes());

  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

function padDatePart(value: number) {
  return String(value).padStart(2, "0");
}
