import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SearchScreen } from "@/features/search/search-screen";
import { DEFAULT_MESSAGE_LIMIT } from "@/lib/constants";

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

  it("does not fetch messages on the first empty page load", async () => {
    render(<SearchScreen />);

    expect(
      await screen.findByText("Arama yapmak için yukarıdaki formu kullanın.")
    ).toBeInTheDocument();
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
    expect(screen.getByLabelText("Yanıtlar")).toBeChecked();
    expect(screen.getByLabelText("Emote içerenler")).toBeChecked();
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

  it("opens CSV export with the current submitted filters", async () => {
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    navigationMocks.query = "q=hello&reply_only=true";

    render(<SearchScreen />);

    await waitFor(() => expect(apiMocks.searchMessages).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole("button", { name: "CSV" }));

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
});
