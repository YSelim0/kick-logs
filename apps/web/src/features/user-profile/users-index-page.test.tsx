import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UsersIndexPage } from "@/features/user-profile/users-index-page";
import type { TopSendersResponse } from "@/types/api";

const analyticsMocks = vi.hoisted(() => ({
  getTopSenders: vi.fn()
}));

vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

vi.mock("@/features/analytics/api", () => analyticsMocks);

describe("UsersIndexPage", () => {
  beforeEach(() => {
    analyticsMocks.getTopSenders.mockReset();
    analyticsMocks.getTopSenders.mockResolvedValue(usersFixture());
  });

  it("renders empty idle prompt on initial load without calling the API", () => {
    render(<UsersIndexPage />);

    expect(screen.getByText("Kullanıcı bulmak için arama yapın")).toBeInTheDocument();
    expect(analyticsMocks.getTopSenders).not.toHaveBeenCalled();
  });

  it("renders search input and submit button", () => {
    render(<UsersIndexPage />);

    expect(screen.getByRole("searchbox", { name: /kullanıcı ara/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^ara$/i })).toBeInTheDocument();
  });

  it("does not call the API while the user is typing", async () => {
    const user = userEvent.setup();
    render(<UsersIndexPage />);

    await user.type(screen.getByRole("searchbox", { name: /kullanıcı ara/i }), "alpha");

    expect(analyticsMocks.getTopSenders).not.toHaveBeenCalled();
  });

  it("submit button is disabled until query has at least 2 characters", async () => {
    const user = userEvent.setup();
    render(<UsersIndexPage />);

    const submit = screen.getByRole("button", { name: /^ara$/i });
    expect(submit).toBeDisabled();

    await user.type(screen.getByRole("searchbox", { name: /kullanıcı ara/i }), "a");
    expect(submit).toBeDisabled();

    await user.type(screen.getByRole("searchbox", { name: /kullanıcı ara/i }), "l");
    expect(submit).not.toBeDisabled();
  });

  it("shows user results after clicking Ara button", async () => {
    const user = userEvent.setup();
    render(<UsersIndexPage />);

    const input = screen.getByRole("searchbox", { name: /kullanıcı ara/i });
    await user.type(input, "alpha");
    await user.click(screen.getByRole("button", { name: /^ara$/i }));

    await waitFor(() =>
      expect(analyticsMocks.getTopSenders).toHaveBeenCalledWith(
        expect.objectContaining({ q: "alpha", limit: 20 })
      )
    );

    expect(await screen.findByText("@alpha")).toBeInTheDocument();
    expect(screen.getByText("@beta")).toBeInTheDocument();
  });

  it("triggers search on Enter key in the input", async () => {
    const user = userEvent.setup();
    render(<UsersIndexPage />);

    const input = screen.getByRole("searchbox", { name: /kullanıcı ara/i });
    await user.type(input, "alpha{Enter}");

    await waitFor(() =>
      expect(analyticsMocks.getTopSenders).toHaveBeenCalledWith(
        expect.objectContaining({ q: "alpha", limit: 20 })
      )
    );
  });

  it("shows empty state when API returns no results", async () => {
    analyticsMocks.getTopSenders.mockResolvedValue({ items: [] });

    const user = userEvent.setup();
    render(<UsersIndexPage />);

    await user.type(screen.getByRole("searchbox", { name: /kullanıcı ara/i }), "xyz");
    await user.click(screen.getByRole("button", { name: /^ara$/i }));

    await waitFor(() =>
      expect(screen.getByText(/"xyz" için kullanıcı bulunamadı\./)).toBeInTheDocument()
    );
  });

  it("shows error state when API fails", async () => {
    analyticsMocks.getTopSenders.mockRejectedValue(new Error("network error"));

    const user = userEvent.setup();
    render(<UsersIndexPage />);

    await user.type(screen.getByRole("searchbox", { name: /kullanıcı ara/i }), "err");
    await user.click(screen.getByRole("button", { name: /^ara$/i }));

    await waitFor(() => expect(screen.getByText(/sonuçlar alınamadı/i)).toBeInTheDocument());
  });

  it("each user row links to the user profile page with _ to - slug conversion", async () => {
    analyticsMocks.getTopSenders.mockResolvedValue({
      items: [
        {
          sender_id: 99,
          kick_user_id: 99,
          username: "example_user",
          slug: "example_user",
          profile_image_url: null,
          message_count: 5,
          first_message_at: "2026-05-01T10:00:00Z",
          latest_message_at: "2026-05-14T09:30:00Z"
        }
      ]
    });

    const user = userEvent.setup();
    render(<UsersIndexPage />);

    await user.type(screen.getByRole("searchbox", { name: /kullanıcı ara/i }), "example");
    await user.click(screen.getByRole("button", { name: /^ara$/i }));

    await waitFor(() => expect(screen.getByText("@example_user")).toBeInTheDocument());

    // slug converts _ to - per Kick profile URL convention
    const links = screen.getAllByRole("link");
    const profileLink = links.find((l) => l.getAttribute("href") === "/users/example-user");
    expect(profileLink).toBeDefined();
  });

  it("user rows with no slug render without a user profile link", async () => {
    analyticsMocks.getTopSenders.mockResolvedValue({
      items: [
        {
          sender_id: 100,
          kick_user_id: 100,
          username: "nolink",
          slug: "",
          profile_image_url: null,
          message_count: 1,
          first_message_at: "2026-05-01T10:00:00Z",
          latest_message_at: "2026-05-14T09:30:00Z"
        }
      ]
    });

    const user = userEvent.setup();
    render(<UsersIndexPage />);

    await user.type(screen.getByRole("searchbox", { name: /kullanıcı ara/i }), "nolink");
    await user.click(screen.getByRole("button", { name: /^ara$/i }));

    await waitFor(() => expect(screen.getByText("@nolink")).toBeInTheDocument());
    const links = screen.getAllByRole("link");
    const userProfileLink = links.find((l) => l.getAttribute("href")?.startsWith("/users/"));
    expect(userProfileLink).toBeUndefined();
  });
});

function usersFixture(): TopSendersResponse {
  return {
    items: [
      {
        sender_id: 1,
        kick_user_id: 10,
        username: "alpha",
        slug: "alpha",
        profile_image_url: null,
        message_count: 1200,
        first_message_at: "2026-05-01T10:00:00Z",
        latest_message_at: "2026-05-14T09:30:00Z"
      },
      {
        sender_id: 2,
        kick_user_id: 20,
        username: "beta",
        slug: "beta",
        profile_image_url: null,
        message_count: 600,
        first_message_at: "2026-05-02T10:00:00Z",
        latest_message_at: "2026-05-13T08:00:00Z"
      }
    ]
  };
}
