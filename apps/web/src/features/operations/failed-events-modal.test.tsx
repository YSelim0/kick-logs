import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithLocale } from "@/test/render-with-locale";
import { LocaleTestControls } from "@/test/locale-controls";
import { FailedEventsModal } from "./failed-events-modal";

const api = vi.hoisted(() => ({ getFailedEvents: vi.fn(), clearFailedEvents: vi.fn() }));
vi.mock("./api", () => api);
describe("localized failed event diagnostics", () => {
  it("does not present a load failure as an empty successful list", async () => {
    api.getFailedEvents.mockRejectedValue(new Error("private database message"));
    renderWithLocale(<FailedEventsModal open onOpenChange={() => {}} />, {
      locale: "en",
      scope: "admin"
    });
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not load failed events.");
    expect(screen.queryByText("No failed events.")).not.toBeInTheDocument();
    expect(screen.queryByText("private database message")).not.toBeInTheDocument();
  });
  beforeEach(() => {
    vi.resetAllMocks();
  });
  it("keeps raw diagnostics and translates the completed cleanup notice", async () => {
    api.getFailedEvents.mockResolvedValue({
      events: [
        {
          raw_event_id: "evt1",
          channel_slug: "Heaven",
          error_message: "RAW CH_CODE_123",
          attempts: 3,
          failed_at: "2026-10-02T00:00:00Z"
        }
      ],
      total: 1
    });
    api.clearFailedEvents.mockResolvedValue({ affected: 1 });
    renderWithLocale(
      <>
        <LocaleTestControls />
        <FailedEventsModal open onOpenChange={() => {}} />
      </>,
      { locale: "tr", scope: "admin" }
    );
    await screen.findByText("RAW CH_CODE_123");
    fireEvent.click(screen.getByRole("button", { name: "Switch to de" }));
    expect(await screen.findByText("Fehlgeschlagene Rohereignisse")).toBeInTheDocument();
    expect(screen.getByText("RAW CH_CODE_123")).toBeInTheDocument();
    expect(screen.getByText("Heaven")).toBeInTheDocument();
    expect(api.getFailedEvents).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole("button", { name: "Alle bereinigen" }));
    expect(await screen.findByText("1 fehlgeschlagener Versuch bereinigt.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Switch to en" }));
    expect(await screen.findByText("1 failed attempt cleared.")).toBeInTheDocument();
    expect(api.clearFailedEvents).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(api.getFailedEvents).toHaveBeenCalledTimes(2));
  });
  it.each([
    ["en", "No failed events."],
    ["tr", "Başarısız event yok."],
    ["de", "Keine fehlgeschlagenen Ereignisse."]
  ] as const)("localizes empty diagnostics in %s", async (locale, label) => {
    api.getFailedEvents.mockResolvedValue({ events: [], total: 0 });
    renderWithLocale(<FailedEventsModal open onOpenChange={() => {}} />, {
      locale,
      scope: "admin"
    });
    expect(await screen.findByText(label)).toBeInTheDocument();
  });
});
