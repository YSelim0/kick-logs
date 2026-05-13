import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AdminDashboard } from "@/features/admin/admin-dashboard";
import type { AdminUser } from "@/types/api";

const navigationMocks = vi.hoisted(() => ({
  replace: vi.fn()
}));

const authMocks = vi.hoisted(() => ({
  logout: vi.fn(),
  state: {
    error: null as string | null,
    refresh: vi.fn(),
    setUser: vi.fn(),
    status: "loading",
    user: null as AdminUser | null
  }
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: navigationMocks.replace })
}));

vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

vi.mock("@/features/auth/api", () => ({
  logout: authMocks.logout
}));

vi.mock("@/features/auth/use-auth", () => ({
  useCurrentUser: () => authMocks.state
}));

vi.mock("@/features/channels/channel-admin", () => ({
  ChannelAdmin: () => <section>Channel admin</section>
}));

vi.mock("@/features/operations/operations-dashboard", () => ({
  OperationsDashboard: () => <section>Operations dashboard</section>
}));

vi.mock("@/features/users/user-admin", () => ({
  UserAdmin: () => <section>Admin users</section>
}));

describe("AdminDashboard", () => {
  beforeEach(() => {
    navigationMocks.replace.mockReset();
    authMocks.logout.mockReset();
    authMocks.state = {
      error: null,
      refresh: vi.fn(),
      setUser: vi.fn(),
      status: "loading",
      user: null
    };
  });

  it("redirects unauthenticated users to login", async () => {
    authMocks.state = {
      ...authMocks.state,
      status: "unauthenticated"
    };

    render(<AdminDashboard />);

    await waitFor(() => expect(navigationMocks.replace).toHaveBeenCalledWith("/login?next=/admin"));
  });

  it("shows the current admin session and logs out", async () => {
    authMocks.logout.mockResolvedValue({ status: "ok" });
    authMocks.state = {
      ...authMocks.state,
      status: "authenticated",
      user: {
        id: 1,
        email: "admin@kicklogs.local",
        role: "super_admin",
        is_active: true
      }
    };

    render(<AdminDashboard />);

    expect(screen.getAllByText("admin@kicklogs.local")).toHaveLength(2);
    expect(screen.getByText("Operations dashboard")).toBeInTheDocument();
    expect(screen.getByText("Admin users")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /çıkış/i }));

    await waitFor(() => expect(authMocks.logout).toHaveBeenCalledTimes(1));
    expect(navigationMocks.replace).toHaveBeenCalledWith("/login");
  });

  it("hides user management from regular admins", () => {
    authMocks.state = {
      ...authMocks.state,
      status: "authenticated",
      user: {
        id: 2,
        email: "operator@kicklogs.local",
        role: "admin",
        is_active: true
      }
    };

    render(<AdminDashboard />);

    expect(screen.getByText("Channel admin")).toBeInTheDocument();
    expect(screen.queryByText("Admin users")).not.toBeInTheDocument();
  });
});
