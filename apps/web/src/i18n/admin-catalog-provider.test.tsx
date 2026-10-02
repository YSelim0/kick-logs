import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { useEffect, useState } from "react";
import { renderToString } from "react-dom/server";
import { useTranslations } from "next-intl";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithLocale } from "@/test/render-with-locale";
import { LocaleTestControls } from "@/test/locale-controls";
import { LocaleProvider, useLocalePreference } from "./locale-provider";
import { loadMessages } from "./catalogs";
import * as catalogs from "./catalogs";
import { AdminCatalogProvider } from "./admin-catalog-provider";
import trAdmin from "./messages/tr/admin.json";
import deAdmin from "./messages/de/admin.json";

function Content() {
  const t = useTranslations("adminShell");
  return <h1>{t("channels")}</h1>;
}

describe("AdminCatalogProvider", () => {
  afterEach(() => vi.restoreAllMocks());

  it.each([false, true])(
    "preserves mounted admin drafts when a public switch finishes after navigation (failure: %s)",
    async (failFirst) => {
      const dePublic = await loadMessages("de", "public");
      let finishPublic!: (value: catalogs.CatalogMessages) => void;
      let finishAdmin!: (value: catalogs.CatalogMessages) => void;
      let rejectAdmin!: (reason: Error) => void;
      vi.spyOn(catalogs, "loadMessages").mockReturnValue(
        new Promise((resolve) => {
          finishPublic = resolve;
        })
      );
      const loader = vi.spyOn(catalogs, "loadAdminMessages").mockImplementation(
        () =>
          new Promise((resolve, reject) => {
            finishAdmin = resolve;
            rejectAdmin = reject;
          })
      );
      const mounted = vi.fn();
      const unmounted = vi.fn();
      function Draft() {
        const t = useTranslations("adminShell");
        const [draft, setDraft] = useState("");
        useEffect(() => {
          mounted();
          return () => {
            unmounted();
          };
        }, []);
        return (
          <>
            <h1>{t("channels")}</h1>
            <input
              aria-label="admin draft"
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
            />
          </>
        );
      }
      function Navigation() {
        const [admin, setAdmin] = useState(false);
        const { changeLocale } = useLocalePreference();
        return (
          <>
            <button onClick={() => void changeLocale("de", "public")}>Start public switch</button>
            <button onClick={() => setAdmin(true)}>Navigate admin</button>
            {admin && (
              <AdminCatalogProvider initialLocale="tr" initialMessages={trAdmin}>
                <Draft />
              </AdminCatalogProvider>
            )}
          </>
        );
      }
      renderWithLocale(<Navigation />, { locale: "tr" });
      fireEvent.click(screen.getByText("Start public switch"));
      fireEvent.click(screen.getByText("Navigate admin"));
      fireEvent.change(screen.getByLabelText("admin draft"), { target: { value: "unsaved note" } });
      await act(async () => finishPublic(dePublic));
      await waitFor(() => expect(loader).toHaveBeenCalledWith("de"));
      expect(screen.getByLabelText("admin draft")).toHaveValue("unsaved note");
      expect(unmounted).not.toHaveBeenCalled();
      if (failFirst) {
        await act(async () => rejectAdmin(new Error("offline")));
        expect(screen.getByLabelText("admin draft")).toHaveValue("unsaved note");
        fireEvent.click(screen.getByRole("button", { name: "Erneut versuchen" }));
        await waitFor(() => expect(loader).toHaveBeenCalledTimes(2));
      }
      await act(async () => finishAdmin(deAdmin));
      expect(await screen.findByRole("heading", { name: "Kanäle" })).toBeInTheDocument();
      expect(screen.getByLabelText("admin draft")).toHaveValue("unsaved note");
      expect(mounted).toHaveBeenCalledTimes(1);
      expect(unmounted).not.toHaveBeenCalled();
    }
  );
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
