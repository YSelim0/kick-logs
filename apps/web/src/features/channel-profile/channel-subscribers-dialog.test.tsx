import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ChannelSubscribersDialog } from "./channel-subscribers-dialog";
import { renderWithLocale } from "@/test/render-with-locale";
import { LocaleTestControls } from "@/test/locale-controls";

const api = vi.hoisted(() => ({
  getChannelSubscribers: vi.fn(),
  buildChannelSubscribersExportUrl: vi.fn()
}));
vi.mock("./api", () => api);
const row = (id: number) => ({
  subscriber_kick_user_id: id,
  username: `example_user${id}`,
  slug: `example-user${id}`,
  is_gift: true,
  gifter_username: "Heaven",
  gifter_slug: "heaven",
  profile_image_url: "",
  started_at: "2026-10-01T00:00:00Z",
  expires_at: "2026-11-01T00:00:00Z"
});

describe("localized subscribers", () => {
  beforeEach(() => {
    api.getChannelSubscribers.mockReset();
  });
  it("preserves appended pages, source names and gift filters on language change", async () => {
    api.getChannelSubscribers
      .mockResolvedValueOnce({ items: [row(1)], count: 3 })
      .mockResolvedValueOnce({ items: [row(2)], count: 3 });
    renderWithLocale(
      <>
        <LocaleTestControls />
        <ChannelSubscribersDialog channelSlug="Heaven" mode="gifted" onOpenChange={vi.fn()} />
      </>,
      { locale: "en" }
    );
    expect(await screen.findByText("example_user1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));
    expect(await screen.findByText("example_user2")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Switch to de", hidden: true }));
    expect(await screen.findByRole("button", { name: "Mehr laden" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Schließen" })).toBeInTheDocument();
    expect(screen.getAllByText("Heaven")).toHaveLength(2);
    expect(api.getChannelSubscribers).toHaveBeenCalledTimes(2);
    expect(api.getChannelSubscribers).toHaveBeenLastCalledWith("Heaven", {
      limit: 50,
      offset: 1,
      gift_only: true
    });
  });
  it.each([
    [
      "en",
      "No active subscriptions have been recorded for this channel yet.",
      "The subscriber list could not be loaded.",
      "Loading subscribers..."
    ],
    [
      "tr",
      "Bu kanal için henüz aktif abonelik kaydı yok.",
      "Abone listesi şu anda alınamadı.",
      "Aboneler yükleniyor..."
    ],
    [
      "de",
      "Für diesen Kanal sind noch keine aktiven Abonnements erfasst.",
      "Die Abonnentenliste konnte nicht geladen werden.",
      "Abonnenten werden geladen..."
    ]
  ] as const)(
    "renders loading, empty and failure states in %s",
    async (locale, empty, error, loading) => {
      api.getChannelSubscribers.mockReturnValue(new Promise(() => {}));
      const pending = renderWithLocale(
        <ChannelSubscribersDialog channelSlug="Heaven" mode="all" onOpenChange={vi.fn()} />,
        { locale }
      );
      expect(screen.getByText(loading)).toBeInTheDocument();
      pending.unmount();
      api.getChannelSubscribers.mockResolvedValue({ items: [], count: 0 });
      const blank = renderWithLocale(
        <ChannelSubscribersDialog channelSlug="Heaven" mode="all" onOpenChange={vi.fn()} />,
        { locale }
      );
      expect(await screen.findByText(empty)).toBeInTheDocument();
      blank.unmount();
      api.getChannelSubscribers.mockRejectedValue(new Error("private"));
      renderWithLocale(
        <ChannelSubscribersDialog channelSlug="Heaven" mode="all" onOpenChange={vi.fn()} />,
        { locale }
      );
      await waitFor(() => expect(screen.getByText(error)).toBeInTheDocument());
      expect(screen.queryByText("private")).not.toBeInTheDocument();
    }
  );
});
