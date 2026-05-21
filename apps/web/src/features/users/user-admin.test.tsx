import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UserAdmin } from "@/features/users/user-admin";
import type { AdminUser } from "@/types/api";

const userApiMocks = vi.hoisted(() => ({
  createAdminUser: vi.fn(),
  listAdminUsers: vi.fn()
}));

vi.mock("@/features/users/api", () => ({
  createAdminUser: userApiMocks.createAdminUser,
  listAdminUsers: userApiMocks.listAdminUsers
}));

describe("UserAdmin", () => {
  beforeEach(() => {
    userApiMocks.createAdminUser.mockReset();
    userApiMocks.listAdminUsers.mockReset();
  });

  it("lists admin users without exposing secrets", async () => {
    userApiMocks.listAdminUsers.mockResolvedValue([adminFixture()]);

    render(<UserAdmin />);

    expect(await screen.findByText("admin@kicklogs.local")).toBeInTheDocument();
    expect(screen.getAllByText("super_admin")).toHaveLength(1);
    expect(screen.getByText("Aktif")).toBeInTheDocument();
    expect(screen.queryByText(/password/i)).not.toBeInTheDocument();
  });

  it("creates a new admin user", async () => {
    const createdUser: AdminUser = {
      id: 2,
      email: "operator@kicklogs.local",
      role: "admin",
      is_active: true
    };

    userApiMocks.listAdminUsers.mockResolvedValue([adminFixture()]);
    userApiMocks.createAdminUser.mockResolvedValue(createdUser);

    render(<UserAdmin />);

    expect(await screen.findByText("admin@kicklogs.local")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("E-POSTA"), {
      target: { value: " operator@kicklogs.local " }
    });
    fireEvent.change(screen.getByLabelText("GEÇİCİ PAROLA"), {
      target: { value: "admin1234" }
    });
    fireEvent.click(screen.getByRole("button", { name: /oluştur/i }));

    await waitFor(() =>
      expect(userApiMocks.createAdminUser).toHaveBeenCalledWith({
        email: "operator@kicklogs.local",
        password: "admin1234"
      })
    );
    expect(await screen.findByText("operator@kicklogs.local")).toBeInTheDocument();
  });
});

function adminFixture(): AdminUser {
  return {
    id: 1,
    email: "admin@kicklogs.local",
    role: "super_admin",
    is_active: true
  };
}
