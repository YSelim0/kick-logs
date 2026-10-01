import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ProfileLoading } from "@/components/profile-loading";

describe("ProfileLoading", () => {
  it.each([
    { kind: "channel" as const, label: "Kanal profili", cells: 6, shape: "rounded-md" },
    { kind: "user" as const, label: "Kullan\u0131c\u0131 profili", cells: 4, shape: "rounded-full" }
  ])("renders an accessible, non-interactive $kind skeleton", ({ kind, label, cells, shape }) => {
    render(<ProfileLoading kind={kind} />);

    const status = screen.getByRole("status");
    expect(status).toHaveTextContent(`${label} y\u00fckleniyor...`);
    expect(status).toHaveAttribute("aria-live", "polite");
    const region = screen.getByRole("region", { name: label });
    expect(region).toHaveAttribute("aria-busy", "true");
    expect(region).not.toContainElement(status);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();

    const placeholders = within(region).getByTestId("profile-placeholders");
    expect(placeholders).toHaveAttribute("aria-hidden", "true");
    expect(within(placeholders).getByTestId("profile-avatar")).toHaveClass(
      "h-[72px]",
      "w-[72px]",
      "shrink-0",
      shape
    );
    expect(within(placeholders).getByTestId("profile-metrics").children).toHaveLength(cells);
    expect(within(placeholders).getByTestId("profile-analytics").children).toHaveLength(3);
    expect(within(placeholders).getByTestId("profile-messages").children).toHaveLength(4);
  });

  it("only animates when motion is safe and retains static placeholders for reduced motion", () => {
    const { container } = render(<ProfileLoading kind="user" />);
    const animated = Array.from(container.querySelectorAll("[class]")).filter((element) =>
      element.getAttribute("class")?.includes("animate-")
    );

    expect(animated.length).toBeGreaterThan(1);
    for (const element of animated) {
      expect(element.getAttribute("class")).toMatch(/motion-safe:animate-(pulse|spin)/);
      expect(element).toHaveClass("motion-reduce:animate-none");
      expect(element).not.toHaveClass("animate-pulse", "animate-spin");
    }
    expect(screen.getByTestId("profile-avatar")).toHaveClass("bg-elevated");
  });
});
