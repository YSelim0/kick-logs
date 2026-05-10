import { fireEvent, render, screen } from "@testing-library/react";
import { createElement } from "react";
import { describe, expect, it } from "vitest";

import { MessageContent, splitMessageContent } from "@/features/search/message-content";

describe("message content", () => {
  it("splits emote tokens at their inline positions", () => {
    expect(
      splitMessageContent("selam [emote:37226:KEKW] chat", [
        {
          id: "37226",
          name: "KEKW",
          token: "[emote:37226:KEKW]",
          image_url: "https://files.kick.com/emotes/37226/fullsize"
        }
      ])
    ).toEqual([
      { type: "text", value: "selam " },
      {
        type: "emote",
        id: "37226",
        name: "KEKW",
        token: "[emote:37226:KEKW]",
        imageUrl: "https://files.kick.com/emotes/37226/fullsize"
      },
      { type: "text", value: " chat" }
    ]);
  });

  it("renders emote fallback text when an image fails", () => {
    render(
      createElement(MessageContent, {
        content: "selam [emote:37226:KEKW]",
        emotes: [
          {
            id: "37226",
            name: "KEKW",
            token: "[emote:37226:KEKW]",
            image_url: "https://files.kick.com/emotes/37226/fullsize"
          }
        ]
      })
    );

    fireEvent.error(screen.getByRole("img", { name: "KEKW" }));

    expect(screen.getByText("KEKW")).toBeInTheDocument();
  });
});
