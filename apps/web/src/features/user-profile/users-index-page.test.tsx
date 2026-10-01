import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UsersIndexPage } from "@/features/user-profile/users-index-page";

const directoryMocks = vi.hoisted(() => ({
  getDirectoryUsers: vi.fn(),
  getDirectoryChannels: vi.fn()
}));
const analyticsMocks = vi.hoisted(() => ({ getTopSenders: vi.fn(), getTopChannels: vi.fn() }));
vi.mock("@/features/directory/api", () => directoryMocks);
vi.mock("@/features/analytics/api", () => analyticsMocks);
vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

describe("UsersIndexPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    directoryMocks.getDirectoryUsers.mockResolvedValue({
      items: [
        {
          id: 1,
          name: "example_user",
          slug: "example_user",
          profile_image_url: "https://example.com/avatar.png"
        }
      ],
      next_cursor: null
    });
  });

  it("searches the identity directory and preserves avatars and canonical profile links without analytics", async () => {
    const user = userEvent.setup();
    render(<UsersIndexPage />);
    await user.type(screen.getByRole("searchbox", { name: "Kullanıcı ara" }), "  example {Enter}");
    expect(await screen.findByRole("link", { name: /@example_user/ })).toHaveAttribute(
      "href",
      "/users/example-user"
    );
    const avatar = screen.getByRole("img", { name: "example_user" });
    expect(avatar).toHaveAttribute("src", "https://example.com/avatar.png");
    fireEvent.error(avatar);
    expect(screen.queryByRole("img", { name: "example_user" })).not.toBeInTheDocument();
    expect(directoryMocks.getDirectoryUsers).toHaveBeenCalledWith(
      { prefix: "example", limit: 50 },
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    );
    expect(directoryMocks.getDirectoryChannels).not.toHaveBeenCalled();
    expect(analyticsMocks.getTopSenders).not.toHaveBeenCalled();
    expect(screen.queryByText(/mesaj|son aktivite/i)).not.toBeInTheDocument();
  });

  it("preserves the fallback row when a stored identity has no link", async () => {
    directoryMocks.getDirectoryUsers.mockResolvedValue({
      items: [{ id: 2, name: "nolink", slug: "", profile_image_url: null }],
      next_cursor: null
    });
    const user = userEvent.setup();
    render(<UsersIndexPage />);
    await user.type(screen.getByRole("searchbox"), "nolink{Enter}");
    expect(await screen.findByText("@nolink")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /nolink/ })).not.toBeInTheDocument();
  });
});
