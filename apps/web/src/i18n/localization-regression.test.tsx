import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { loadMessages } from "./catalogs";
import { PageMetadata } from "./page-metadata";
import { LocaleTestControls } from "@/test/locale-controls";
import { renderWithLocale } from "@/test/render-with-locale";

const route = vi.hoisted(() => ({ pathname: "/channels" }));
vi.mock("next/navigation", () => ({ usePathname: () => route.pathname }));

afterEach(() => {
  document.querySelector('meta[name="description"]')?.remove();
});

describe("localization acceptance regressions", () => {
  it("updates route metadata in place and resets it after navigation", async () => {
    const meta = document.createElement("meta");
    meta.name = "description";
    document.head.append(meta);
    route.pathname = "/channels";
    const view = renderWithLocale(
      <>
        <PageMetadata />
        <LocaleTestControls />
      </>,
      { locale: "en" }
    );
    expect(document.title).toBe("Channels - Kick Logs");
    expect(meta.content).toBe("Search and explore recorded Kick channels.");
    fireEvent.click(screen.getByText("Switch to de"));
    await waitFor(() => expect(document.title).toBe("Kanäle - Kick Logs"));
    expect(meta.content).toBe("Suche und entdecke erfasste Kick-Kanäle.");
    route.pathname = "/users";
    view.rerender(
      <>
        <PageMetadata />
        <LocaleTestControls />
      </>
    );
    expect(document.title).toBe("Nutzer - Kick Logs");
    route.pathname = "/search";
    view.rerender(
      <>
        <PageMetadata />
        <LocaleTestControls />
      </>
    );
    expect(document.title).toBe("Kick Logs");
    expect(meta.content).toContain("Kick-Chatverlauf");
    meta.content = "stale server description";
    route.pathname = "/request";
    view.rerender(
      <>
        <PageMetadata />
        <LocaleTestControls />
      </>
    );
    expect(meta.content).toContain("Kick-Chatverlauf");
    fireEvent.click(screen.getByText("Switch to tr"));
    await waitFor(() => expect(meta.content).toContain("Kick sohbet geçmişinde"));
  });

  it.each(["en", "tr", "de"] as const)(
    "keeps %s admin catalogs out of public responses",
    async (locale) => {
      const publicMessages = await loadMessages(locale, "public");
      const adminMessages = await loadMessages(locale, "admin");
      for (const namespace of [
        "adminShell",
        "channelAdmin",
        "userAdmin",
        "operations",
        "dataManagement"
      ]) {
        expect(publicMessages).not.toHaveProperty(namespace);
        expect(adminMessages).toHaveProperty(namespace);
      }
    }
  );
});
