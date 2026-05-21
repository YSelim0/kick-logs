import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import AdminLayout from "@/app/admin/layout";
import type { AdminUser } from "@/types/api";

const navigationMocks = vi.hoisted(() => ({
  replace: vi.fn(),
  pathname: "/admin/operations"
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
  useRouter: () => ({ replace: navigationMocks.replace }),
  usePathname: () => navigationMocks.pathname,
  redirect: vi.fn()
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

describe("AdminLayout", () => {
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

    render(<AdminLayout>content</AdminLayout>);

    await waitFor(() => expect(navigationMocks.replace).toHaveBeenCalledWith("/login?next=/admin"));
  });

  it("shows the admin header with user email and logout", () => {
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

    render(<AdminLayout>page content</AdminLayout>);

    expect(screen.getByText("admin@kicklogs.local")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /kick logs/i })).toHaveAttribute("href", "/");
    expect(screen.getByRole("button", { name: /çıkış/i })).toBeInTheDocument();
    expect(screen.getByText("SUPER ADMIN")).toBeInTheDocument();
    expect(screen.getByText("page content")).toBeInTheDocument();
  });

  it("hides Users nav item from regular admins", () => {
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

    render(<AdminLayout>content</AdminLayout>);

    expect(screen.getByText("Operations")).toBeInTheDocument();
    expect(screen.getByText("Channels")).toBeInTheDocument();
    expect(screen.queryByText("Users")).not.toBeInTheDocument();
    expect(screen.getByText("Data")).toBeInTheDocument();
  });
});
