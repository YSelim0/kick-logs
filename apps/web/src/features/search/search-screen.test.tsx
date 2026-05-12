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

vi.mock("@/features/search/api", () => ({
  searchMessages: apiMocks.searchMessages
}));

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
});
