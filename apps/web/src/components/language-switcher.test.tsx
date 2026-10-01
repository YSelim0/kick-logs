import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithLocale } from "@/test/render-with-locale";
import * as catalogs from "@/i18n/catalogs";
import { LanguageSwitcher } from "./language-switcher";
import { SiteHeader } from "./site-header";

vi.mock("next/navigation", () => ({ usePathname: () => "/search" }));

describe("LanguageSwitcher", () => {
  it("opens upward from a stable square flag and switches navigation without changing URLs", async () => {
    renderWithLocale(
      <>
        <SiteHeader />
        <LanguageSwitcher />
      </>,
      { locale: "en" }
    );
    const trigger = screen.getByRole("button", { name: "Change language (English)" });
    expect(trigger).toHaveClass("h-12", "w-12");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    fireEvent.click(trigger);
    expect(screen.getByRole("menu")).toHaveClass("bottom-full");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.getByRole("menuitemradio", { name: "English" })).toHaveAttribute(
      "aria-checked",
      "true"
    );
    fireEvent.click(screen.getByRole("menuitemradio", { name: "Deutsch" }));
    await screen.findByRole("button", { name: "Sprache ändern (Deutsch)" });
    expect(screen.getByRole("link", { name: "Kanäle" })).toHaveAttribute("href", "/channels");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    expect(document.cookie).toContain("kick_logs_locale=de");
  });

  it("supports keyboard selection, Escape/focus return, toggle and outside clicks", async () => {
    renderWithLocale(<LanguageSwitcher />, { locale: "en" });
    const trigger = screen.getByRole("button", { name: "Change language (English)" });
    fireEvent.keyDown(trigger, { key: "ArrowUp" });
    await waitFor(() =>
      expect(screen.getByRole("menuitemradio", { name: "English" })).toHaveFocus()
    );
    fireEvent.keyDown(document.activeElement!, { key: "ArrowDown" });
    expect(screen.getByRole("menuitemradio", { name: "Deutsch" })).toHaveFocus();
    fireEvent.keyDown(document.activeElement!, { key: "Home" });
    expect(screen.getByRole("menuitemradio", { name: "Türkçe" })).toHaveFocus();
    fireEvent.keyDown(document.activeElement!, { key: "End" });
    expect(screen.getByRole("menuitemradio", { name: "Deutsch" })).toHaveFocus();
    fireEvent.keyDown(document.activeElement!, { key: "Escape" });
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
    fireEvent.click(trigger);
    fireEvent.click(trigger);
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    fireEvent.click(trigger);
    fireEvent.pointerDown(document.body);
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("prevents duplicate loads while a language choice is pending", async () => {
    const messages = await catalogs.loadMessages("de", "public");
    let resolve!: (value: typeof messages) => void;
    const load = vi.spyOn(catalogs, "loadMessages").mockImplementation(
      () =>
        new Promise((done) => {
          resolve = done;
        })
    );
    renderWithLocale(<LanguageSwitcher />, { locale: "en" });
    fireEvent.click(screen.getByRole("button", { name: "Change language (English)" }));
    fireEvent.click(screen.getByRole("menuitemradio", { name: "Deutsch" }));
    expect(screen.getByRole("button", { name: "Change language (English)" })).toBeDisabled();
    expect(load).toHaveBeenCalledTimes(1);
    await act(async () => resolve(messages));
    expect(screen.getByRole("button", { name: "Sprache ändern (Deutsch)" })).toBeEnabled();
    load.mockRestore();
  });
});
