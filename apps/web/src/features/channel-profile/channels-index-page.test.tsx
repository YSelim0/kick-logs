import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ChannelsIndexPage } from "@/features/channel-profile/channels-index-page";

const directoryMocks = vi.hoisted(() => ({
  getDirectoryChannels: vi.fn(),
  getDirectoryUsers: vi.fn()
}));
const analyticsMocks = vi.hoisted(() => ({ getTopChannels: vi.fn(), getTopSenders: vi.fn() }));
vi.mock("@/features/directory/api", () => directoryMocks);
vi.mock("@/features/analytics/api", () => analyticsMocks);
vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

describe("ChannelsIndexPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    directoryMocks.getDirectoryChannels.mockResolvedValue({
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

  it("searches the identity directory and preserves avatars and stored profile links without analytics", async () => {
    const user = userEvent.setup();
    render(<ChannelsIndexPage />);
    await user.type(screen.getByRole("searchbox", { name: "Kanal ara" }), "  example {Enter}");
    expect(await screen.findByRole("link", { name: /example_user/ })).toHaveAttribute(
      "href",
      "/channels/example_user"
    );
    const avatar = screen.getByRole("img", { name: "example_user" });
    expect(avatar).toHaveAttribute("src", "https://example.com/avatar.png");
    fireEvent.error(avatar);
    expect(screen.queryByRole("img", { name: "example_user" })).not.toBeInTheDocument();
    expect(directoryMocks.getDirectoryChannels).toHaveBeenCalledWith(
      { prefix: "example", limit: 50 },
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    );
    expect(directoryMocks.getDirectoryUsers).not.toHaveBeenCalled();
    expect(analyticsMocks.getTopChannels).not.toHaveBeenCalled();
    expect(screen.queryByText(/mesaj|son aktivite/i)).not.toBeInTheDocument();
  });

  it("preserves the fallback row when a stored identity has no link", async () => {
    directoryMocks.getDirectoryChannels.mockResolvedValue({
      items: [{ id: 2, name: "nolink", slug: "", profile_image_url: null }],
      next_cursor: null
    });
    const user = userEvent.setup();
    render(<ChannelsIndexPage />);
    await user.type(screen.getByRole("searchbox"), "nolink{Enter}");
    expect(await screen.findByText("nolink")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /nolink/ })).not.toBeInTheDocument();
  });
});
