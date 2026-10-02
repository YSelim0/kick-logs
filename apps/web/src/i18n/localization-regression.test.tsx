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
    expect(document.title).toBe("Channels - KickLogs");
    expect(meta.content).toBe("Search and explore recorded Kick channels.");
    fireEvent.click(screen.getByText("Switch to de"));
    await waitFor(() => expect(document.title).toBe("Kanäle - KickLogs"));
    expect(meta.content).toBe("Suche und entdecke erfasste Kick-Kanäle.");
    route.pathname = "/users";
    view.rerender(
      <>
        <PageMetadata />
        <LocaleTestControls />
      </>
    );
    expect(document.title).toBe("Nutzer - KickLogs");
    route.pathname = "/search";
    view.rerender(
      <>
        <PageMetadata />
        <LocaleTestControls />
      </>
    );
    expect(document.title).toBe("Chat-Suche - KickLogs");
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
    expect(document.title).toBe("Anfrage senden - KickLogs");
    fireEvent.click(screen.getByText("Switch to tr"));
    await waitFor(() => expect(meta.content).toContain("Kick sohbet geçmişinde"));
    expect(document.title).toBe("Talep Gönder - KickLogs");
  });

  it.each(["en", "tr", "de"] as const)(
    "gives every route a distinct %s title without translating identities",
    (locale) => {
      const pages = [
        ["/", "Home", "Anasayfa", "Startseite"],
        ["/search", "Chat Search", "Sohbet Arama", "Chat-Suche"],
        ["/channels", "Channels", "Kanallar", "Kanäle"],
        ["/users", "Users", "Kullanıcılar", "Nutzer"],
        ["/prediction", "Prediction", "Tahmin", "Vorhersage"],
        ["/request", "Submit a Request", "Talep Gönder", "Anfrage senden"],
        ["/login", "Admin Sign In", "Yönetici Girişi", "Admin-Anmeldung"],
        [
          "/channels/Heaven",
          "Heaven Channel Profile",
          "Heaven Kanal Profili",
          "Heaven Kanalprofil"
        ],
        [
          "/users/example_user",
          "example_user User Profile",
          "example_user Kullanıcı Profili",
          "example_user Nutzerprofil"
        ],
        [
          "/prediction/hype",
          "hype Prediction Analysis",
          "hype Tahmin Analizi",
          "hype Vorhersageanalyse"
        ],
        ["/admin", "Operations - Admin", "Operasyonlar - Yönetim", "Betrieb - Admin"],
        ["/admin/operations", "Operations - Admin", "Operasyonlar - Yönetim", "Betrieb - Admin"],
        [
          "/admin/channels",
          "Followed Channels - Admin",
          "Takip Edilen Kanallar - Yönetim",
          "Verfolgte Kanäle - Admin"
        ],
        [
          "/admin/users",
          "Administrators - Admin",
          "Yöneticiler - Yönetim",
          "Administratoren - Admin"
        ],
        ["/admin/requests", "Requests - Admin", "Talepler - Yönetim", "Anfragen - Admin"],
        [
          "/admin/data",
          "Data Management - Admin",
          "Veri Yönetimi - Yönetim",
          "Datenverwaltung - Admin"
        ],
        ["/missing", "Page Not Found", "Sayfa Bulunamadı", "Seite nicht gefunden"],
        ["/channels/Heaven/missing", "Page Not Found", "Sayfa Bulunamadı", "Seite nicht gefunden"]
      ];
      route.pathname = "/";
      const view = renderWithLocale(<PageMetadata />, { locale });
      const column = { en: 1, tr: 2, de: 3 }[locale];
      for (const [pathname, ...titles] of pages) {
        route.pathname = pathname;
        view.rerender(<PageMetadata />);
        expect(document.title, pathname).toBe(`${titles[column - 1]} - KickLogs`);
      }
    }
  );

  it("decodes a profile slug once and updates it on same-section navigation", () => {
    route.pathname = "/users/example%5Fuser";
    const view = renderWithLocale(<PageMetadata />, { locale: "tr" });
    expect(document.title).toBe("example_user Kullanıcı Profili - KickLogs");
    route.pathname = "/users/Heaven";
    view.rerender(<PageMetadata />);
    expect(document.title).toBe("Heaven Kullanıcı Profili - KickLogs");
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
