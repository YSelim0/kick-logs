import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderToString } from "react-dom/server";
import { useTranslations } from "next-intl";
import { describe, expect, it, vi } from "vitest";
import { renderWithLocale } from "@/test/render-with-locale";
import { LocaleTestControls } from "@/test/locale-controls";
import { LocaleProvider } from "./locale-provider";
import { loadMessages } from "./catalogs";
import { AdminCatalogProvider } from "./admin-catalog-provider";
import trAdmin from "./messages/tr/admin.json";
import deAdmin from "./messages/de/admin.json";

function Content() {
  const t = useTranslations("adminShell");
  return <h1>{t("channels")}</h1>;
}

describe("AdminCatalogProvider", () => {
  it("renders direct admin SSR in the request locale", async () => {
    const html = renderToString(
      <LocaleProvider initialLocale="de" initialMessages={await loadMessages("de", "public")}>
        <AdminCatalogProvider initialLocale="de" initialMessages={deAdmin}>
          <Content />
        </AdminCatalogProvider>
      </LocaleProvider>
    );
    expect(html).toContain("Kanäle");
    expect(html).not.toContain("Kanallar");
  });

  it("keeps a session-selected language despite stale server props and blocked cookies", async () => {
    const cookie = vi.spyOn(document, "cookie", "set").mockImplementation(() => {
      throw new Error("blocked");
    });
    const { rerender } = renderWithLocale(<LocaleTestControls />, { locale: "tr" });
    fireEvent.click(screen.getByRole("button", { name: "Switch to de" }));
    await waitFor(() => expect(document.documentElement.lang).toBe("de"));
    rerender(
      <AdminCatalogProvider initialLocale="tr" initialMessages={trAdmin}>
        <Content />
      </AdminCatalogProvider>
    );
    expect(await screen.findByRole("heading", { name: "Kanäle" })).toBeInTheDocument();
    expect(screen.queryByText("Kanallar")).not.toBeInTheDocument();
    cookie.mockRestore();
  });

  it("loads admin messages when navigating from a public session with different server locale", async () => {
    renderWithLocale(
      <AdminCatalogProvider initialLocale="tr" initialMessages={trAdmin}>
        <Content />
      </AdminCatalogProvider>,
      { locale: "de" }
    );
    expect(await screen.findByRole("heading", { name: "Kanäle" })).toBeInTheDocument();
    expect(screen.queryByText("Kanallar")).not.toBeInTheDocument();
  });
});
