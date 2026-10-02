import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { NotFoundScreen as NotFound } from "@/components/not-found-screen";
import { renderWithLocale } from "@/test/render-with-locale";
import { KickProfileLink } from "./kick-profile-link";
import { ProfileLoading } from "./profile-loading";
import { DialogClose } from "./ui/dialog";

describe("localized shared chrome", () => {
  it("translates a 404 without changing the destination routes", () => {
    renderWithLocale(<NotFound />, { locale: "de" });
    expect(screen.getByRole("heading", { name: "Seite nicht gefunden." })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Zur Suche" })).toHaveAttribute("href", "/search");
  });
  it("translates profile controls and keeps the Kick slug intact", () => {
    renderWithLocale(
      <>
        <KickProfileLink href="https://kick.com/Heaven" />
        <ProfileLoading kind="user" />
        <DialogClose onClose={() => {}} />
      </>,
      { locale: "en" }
    );
    expect(screen.getByRole("link", { name: "Visit Kick account" })).toHaveAttribute(
      "href",
      "https://kick.com/Heaven"
    );
    expect(screen.getByRole("status")).toHaveTextContent("Loading user profile...");
    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();
  });
});
