import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useTranslations } from "next-intl";
import { useState } from "react";
import { renderToString } from "react-dom/server";
import { hydrateRoot } from "react-dom/client";
import { describe, expect, it, vi, afterEach } from "vitest";

import * as catalogs from "./catalogs";
import { LocaleProvider, useLocalePreference } from "./locale-provider";
import { saveLocalePreference } from "./preference";

function Example() {
  const t = useTranslations("common");
  const { locale, changeLocale } = useLocalePreference();
  const [value, setValue] = useState("");
  return (
    <>
      <span>{t("actions.close")}</span>
      <output>{locale}</output>
      <input aria-label="draft" value={value} onChange={(event) => setValue(event.target.value)} />
      <button onClick={() => void changeLocale("de")}>German</button>
      <button onClick={() => void changeLocale("tr")}>Turkish</button>
    </>
  );
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("locale provider", () => {
  it("hydrates real server markup without a recoverable mismatch", async () => {
    const messages = await catalogs.loadMessages("de", "public");
    const tree = (
      <LocaleProvider initialLocale="de" initialMessages={messages}>
        <Example />
      </LocaleProvider>
    );
    const container = document.createElement("div");
    container.innerHTML = renderToString(tree);
    document.body.append(container);
    const onRecoverableError = vi.fn();
    let root!: ReturnType<typeof hydrateRoot>;
    await act(async () => {
      root = hydrateRoot(container, tree, { onRecoverableError });
    });
    expect(container).toHaveTextContent("Schließen");
    expect(onRecoverableError).not.toHaveBeenCalled();
    await act(async () => root.unmount());
    container.remove();
  });

  it("retains the current locale if loading the new catalog fails", async () => {
    const messages = await catalogs.loadMessages("en", "public");
    vi.spyOn(catalogs, "loadMessages").mockRejectedValue(new Error("offline"));
    render(
      <LocaleProvider initialLocale="en" initialMessages={messages}>
        <Example />
      </LocaleProvider>
    );
    fireEvent.click(screen.getByText("German"));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("en"));
    expect(screen.getByText("Close")).toBeInTheDocument();
  });
  it("uses the same initial locale on the server and client, then keeps input state on switch", async () => {
    const messages = await catalogs.loadMessages("en", "public");
    const tree = (
      <LocaleProvider initialLocale="en" initialMessages={messages}>
        <Example />
      </LocaleProvider>
    );
    expect(renderToString(tree)).toContain("Close");
    render(tree);
    expect(screen.getByText("Close")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("draft"), { target: { value: "Heaven" } });
    fireEvent.click(screen.getByText("German"));
    await screen.findByText("Schließen");
    expect(screen.getByLabelText("draft")).toHaveValue("Heaven");
    expect(document.documentElement.lang).toBe("de");
    expect(document.cookie).toContain("kick_logs_locale=de");
  });

  it("keeps manual selection usable when cookie writes are blocked", async () => {
    const messages = await catalogs.loadMessages("en", "public");
    vi.spyOn(document, "cookie", "set").mockImplementation(() => {
      throw new Error("blocked");
    });
    expect(saveLocalePreference("de")).toBe(false);
    render(
      <LocaleProvider initialLocale="en" initialMessages={messages}>
        <Example />
      </LocaleProvider>
    );
    fireEvent.click(screen.getByText("Turkish"));
    await screen.findByText("Kapat");
    expect(screen.getByRole("status")).toHaveTextContent("tr");
  });

  it("ignores an older catalog response after a newer selection", async () => {
    const en = await catalogs.loadMessages("en", "public");
    const de = await catalogs.loadMessages("de", "public");
    const tr = await catalogs.loadMessages("tr", "public");
    let resolveGerman!: (value: typeof de) => void;
    vi.spyOn(catalogs, "loadMessages").mockImplementation((locale) =>
      locale === "de"
        ? new Promise((resolve) => {
            resolveGerman = resolve;
          })
        : Promise.resolve(tr)
    );
    render(
      <LocaleProvider initialLocale="en" initialMessages={en}>
        <Example />
      </LocaleProvider>
    );
    fireEvent.click(screen.getByText("German"));
    fireEvent.click(screen.getByText("Turkish"));
    await screen.findByText("Kapat");
    await act(async () => resolveGerman(de));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("tr"));
    expect(document.cookie).toContain("kick_logs_locale=tr");
  });
});
