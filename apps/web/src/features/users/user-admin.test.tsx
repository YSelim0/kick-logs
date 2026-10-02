import { fireEvent, screen, waitFor } from "@testing-library/react";
import { LocaleTestControls } from "@/test/locale-controls";
import { createLocaleRenderer } from "@/test/render-with-locale";
const render = createLocaleRenderer("tr", "admin");
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
  it("keeps unsaved credentials and localizes role presentation only", async () => {
    userApiMocks.listAdminUsers.mockResolvedValue([adminFixture()]);
    render(
      <>
        <LocaleTestControls />
        <UserAdmin />
      </>
    );
    await screen.findByText("admin@kicklogs.local");
    fireEvent.change(screen.getByLabelText("E-POSTA"), {
      target: { value: "operator@example.com" }
    });
    fireEvent.change(screen.getByLabelText("GEÇİCİ PAROLA"), {
      target: { value: "not-a-real-password" }
    });
    fireEvent.click(screen.getByRole("button", { name: "Switch to en" }));
    expect(await screen.findByLabelText("TEMPORARY PASSWORD")).toHaveValue("not-a-real-password");
    expect(screen.getByLabelText("EMAIL")).toHaveValue("operator@example.com");
    expect(screen.getAllByText("Super admin").length).toBeGreaterThan(0);
    expect(userApiMocks.listAdminUsers).toHaveBeenCalledTimes(1);
    expect(userApiMocks.createAdminUser).not.toHaveBeenCalled();
  });
  beforeEach(() => {
    userApiMocks.createAdminUser.mockReset();
    userApiMocks.listAdminUsers.mockReset();
  });

  it("lists admin users without exposing secrets", async () => {
    userApiMocks.listAdminUsers.mockResolvedValue([adminFixture()]);

    render(<UserAdmin />);

    expect(await screen.findByText("admin@kicklogs.local")).toBeInTheDocument();
    expect(screen.getAllByText("Süper admin")).not.toHaveLength(0);
    expect(screen.getAllByText("Aktif")).not.toHaveLength(0);
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
