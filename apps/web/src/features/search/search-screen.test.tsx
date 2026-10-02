import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { createLocaleRenderer, renderWithLocale } from "@/test/render-with-locale";
import { LocaleTestControls } from "@/test/locale-controls";
const render = createLocaleRenderer("tr", "admin");
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SearchScreen } from "@/features/search/search-screen";
import { DEFAULT_MESSAGE_LIMIT } from "@/lib/constants";
import type { Message } from "@/types/api";

const navigationMocks = vi.hoisted(() => ({
  push: vi.fn(),
  query: ""
}));

const apiMocks = vi.hoisted(() => ({
  searchMessages: vi.fn()
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: navigationMocks.push }),
  useSearchParams: () => new URLSearchParams(navigationMocks.query)
}));

vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

vi.mock("@/features/search/api", async () => {
  const actual =
    await vi.importActual<typeof import("@/features/search/api")>("@/features/search/api");

  return {
    ...actual,
    searchMessages: apiMocks.searchMessages
  };
});

describe("SearchScreen", () => {
  beforeEach(() => {
    navigationMocks.push.mockReset();
    navigationMocks.query = "";
    apiMocks.searchMessages.mockReset();
    apiMocks.searchMessages.mockResolvedValue({ items: [], next_cursor: null });
  });

  it("preserves paginated results, drafts and submitted filters across locale changes", async () => {
    let intersect: IntersectionObserverCallback = () => {};
    vi.stubGlobal(
      "IntersectionObserver",
      class {
        constructor(callback: IntersectionObserverCallback) {
          intersect = callback;
        }
        observe() {}
        disconnect() {}
      }
    );
    navigationMocks.query = "q=hello&sender=yavuz";
    apiMocks.searchMessages
      .mockResolvedValueOnce({ items: [messageFixture(1, "hello Heaven")], next_cursor: "page2" })
      .mockResolvedValueOnce({
        items: [messageFixture(2, "hello unchanged")],
        next_cursor: "page3"
      });
    const view = render(
      <>
        <LocaleTestControls />
        <SearchScreen />
      </>
    );
    expect(await screen.findByText("Heaven", { exact: false })).toBeInTheDocument();
    await act(async () =>
      intersect(
        [{ isIntersecting: true }] as IntersectionObserverEntry[],
        {} as IntersectionObserver
      )
    );
    await waitFor(() => expect(apiMocks.searchMessages).toHaveBeenCalledTimes(2));
    fireEvent.change(screen.getByLabelText("Kullanıcı Adı"), { target: { value: "unsent_name" } });
    fireEvent.click(screen.getByRole("button", { name: "Switch to de" }));
    expect(await screen.findByLabelText("Benutzername")).toHaveValue("unsent_name");
    expect(screen.getByText("unchanged", { exact: false })).toBeInTheDocument();
    expect(apiMocks.searchMessages).toHaveBeenCalledTimes(2);
    expect(apiMocks.searchMessages.mock.calls[1][0]).toMatchObject({
      q: "hello",
      sender: "yavuz",
      cursor: "page2"
    });
    expect(navigationMocks.query).toBe("q=hello&sender=yavuz");
    expect(navigationMocks.push).not.toHaveBeenCalled();
    view.unmount();
    vi.unstubAllGlobals();
  });

  it("translates an existing error without retrying or exposing server details", async () => {
    navigationMocks.query = "q=hello";
    apiMocks.searchMessages.mockRejectedValue(new Error("private database details"));
    render(
      <>
        <LocaleTestControls />
        <SearchScreen />
      </>
    );
    expect(await screen.findByText("Mesajlar yüklenirken bir hata oluştu.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Switch to en" }));
    expect(await screen.findByText("Messages could not be loaded.")).toBeInTheDocument();
    expect(screen.queryByText("private database details")).not.toBeInTheDocument();
    expect(apiMocks.searchMessages).toHaveBeenCalledTimes(1);
  });

  it.each(["en", "tr", "de"] as const)(
    "keeps date and identity parameters locale-independent in %s",
    async (locale) => {
      navigationMocks.query =
        "sender=example_user&channel=Heaven&start=2026-05-02T02%3A43&end=2026-05-09T02%3A43";
      renderWithLocale(<SearchScreen />, { locale });
      const end = new Date("2026-05-09T02:43");
      end.setSeconds(59, 999);
      await waitFor(() =>
        expect(apiMocks.searchMessages).toHaveBeenCalledWith({
          sender: "example_user",
          channel: "Heaven",
          start: new Date("2026-05-02T02:43").toISOString(),
          end: end.toISOString(),
          limit: DEFAULT_MESSAGE_LIMIT
        })
      );
    }
  );

  it("does not fetch messages on the first empty page load", async () => {
    render(<SearchScreen />);

    expect(
      await screen.findByText("Arama yapmak için yukarıdaki formu kullanın.")
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /kick logs/i })).toHaveAttribute("href", "/");
    expect(apiMocks.searchMessages).not.toHaveBeenCalled();
  });

  it("loads results when the URL already contains search params", async () => {
    navigationMocks.query = "q=hello";

    render(<SearchScreen />);

    await waitFor(() => expect(apiMocks.searchMessages).toHaveBeenCalledTimes(1));
    expect(apiMocks.searchMessages).toHaveBeenCalledWith(
      expect.objectContaining({
        limit: DEFAULT_MESSAGE_LIMIT,
        q: "hello"
      })
    );
    expect(await screen.findByText("Bu filtrelerle mesaj bulunamadı.")).toBeInTheDocument();
  });

  it("deduplicates initial results before rendering message rows", async () => {
    navigationMocks.query = "q=filter";
    apiMocks.searchMessages.mockResolvedValue({
      items: [
        messageFixture(8333132417695925000, "first copy"),
        messageFixture(8333132417695925000, "second copy")
      ],
      next_cursor: null
    });

    render(<SearchScreen />);

    expect(await screen.findByText("first copy")).toBeInTheDocument();
    expect(screen.queryByText("second copy")).not.toBeInTheDocument();
  });

  it("sends date filters to the API as UTC ISO values", async () => {
    navigationMocks.query = "start=2026-05-02T02%3A43&end=2026-05-09T02%3A43";
    const endDate = new Date("2026-05-09T02:43");
    endDate.setSeconds(59, 999);

    render(<SearchScreen />);

    await waitFor(() =>
      expect(apiMocks.searchMessages).toHaveBeenCalledWith(
        expect.objectContaining({
          start: new Date("2026-05-02T02:43").toISOString(),
          end: endDate.toISOString()
        })
      )
    );
  });

  it("keeps reply-only and emote-only URL state shareable", async () => {
    navigationMocks.query = "q=hello&reply_only=true&emote_only=true";

    render(<SearchScreen />);

    await waitFor(() =>
      expect(apiMocks.searchMessages).toHaveBeenCalledWith(
        expect.objectContaining({
          q: "hello",
          reply_only: true,
          emote_only: true
        })
      )
    );
    expect(screen.getByLabelText("Sadece yanıtlar")).toBeChecked();
    expect(screen.getByLabelText("Sadece emote")).toBeChecked();
  });

  it("allows an explicit empty search without navigating", async () => {
    render(<SearchScreen />);

    const startInput = screen.getByLabelText("Başlangıç");
    const endInput = screen.getByLabelText("Bitiş");

    await waitFor(() => expect((startInput as HTMLInputElement).value).toContain("T"));
    fireEvent.change(startInput, { target: { value: "" } });
    fireEvent.change(endInput, { target: { value: "" } });
    fireEvent.click(screen.getByRole("button", { name: "Ara" }));

    await waitFor(() =>
      expect(apiMocks.searchMessages).toHaveBeenCalledWith({
        limit: DEFAULT_MESSAGE_LIMIT
      })
    );
    expect(navigationMocks.push).not.toHaveBeenCalled();
  });

  it("applies date presets from the compact quick range control", async () => {
    render(<SearchScreen />);

    const startInput = screen.getByLabelText("Başlangıç") as HTMLInputElement;

    await waitFor(() => expect(startInput.value).toContain("T"));
    const previousStart = startInput.value;
    fireEvent.change(screen.getByLabelText("Hızlı aralık"), { target: { value: "24h" } });

    await waitFor(() => expect(startInput.value).not.toBe(previousStart));
  });

  it("opens CSV export with the current submitted filters", async () => {
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    navigationMocks.query = "q=hello&reply_only=true";

    render(<SearchScreen />);

    await waitFor(() => expect(apiMocks.searchMessages).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole("button", { name: "Dışa aktar" }));
    fireEvent.click(screen.getByRole("button", { name: "CSV indir" }));

    const [url, target, features] = openSpy.mock.calls[0];
    expect(target).toBe("_blank");
    expect(features).toBe("noopener,noreferrer");

    const exportUrl = new URL(url as string);
    expect(exportUrl.pathname).toBe("/messages/export");
    expect(exportUrl.searchParams.get("format")).toBe("csv");
    expect(exportUrl.searchParams.get("q")).toBe("hello");
    expect(exportUrl.searchParams.get("reply_only")).toBe("true");

    openSpy.mockRestore();
  });

  it("closes the export menu when the user clicks outside", async () => {
    navigationMocks.query = "q=hello";

    render(<SearchScreen />);

    await waitFor(() => expect(apiMocks.searchMessages).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole("button", { name: "Dışa aktar" }));
    expect(screen.getByRole("button", { name: "CSV indir" })).toBeInTheDocument();

    fireEvent.mouseDown(document.body);

    await waitFor(() =>
      expect(screen.queryByRole("button", { name: "CSV indir" })).not.toBeInTheDocument()
    );
  });
});

function messageFixture(id: number, content: string): Message {
  return {
    id,
    kick_message_id: `kick-${id}`,
    chatroom_id: 10,
    content,
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
