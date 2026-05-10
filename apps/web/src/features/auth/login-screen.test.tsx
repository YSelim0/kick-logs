import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LoginScreen } from "@/features/auth/login-screen";
import { ApiClientError } from "@/lib/api-client";

const navigationMocks = vi.hoisted(() => ({
  query: "",
  replace: vi.fn()
}));

const apiMocks = vi.hoisted(() => ({
  login: vi.fn()
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: navigationMocks.replace }),
  useSearchParams: () => new URLSearchParams(navigationMocks.query)
}));

vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

vi.mock("@/features/auth/api", () => ({
  login: apiMocks.login
}));

describe("LoginScreen", () => {
  beforeEach(() => {
    navigationMocks.query = "";
    navigationMocks.replace.mockReset();
    apiMocks.login.mockReset();
  });

  it("logs in and redirects to admin", async () => {
    apiMocks.login.mockResolvedValue({
      user: {
        id: 1,
        email: "admin@kicklogs.local",
        role: "super_admin",
        is_active: true
      }
    });

    render(<LoginScreen />);

    fireEvent.change(screen.getByLabelText("Parola"), {
      target: { value: "admin123" }
    });
    fireEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    await waitFor(() =>
      expect(apiMocks.login).toHaveBeenCalledWith({
        email: "admin@kicklogs.local",
        password: "admin123"
      })
    );
    expect(navigationMocks.replace).toHaveBeenCalledWith("/admin");
  });

  it("uses the safe next path after login", async () => {
    navigationMocks.query = "next=/admin";
    apiMocks.login.mockResolvedValue({
      user: {
        id: 1,
        email: "admin@kicklogs.local",
        role: "admin",
        is_active: true
      }
    });

    render(<LoginScreen />);

    fireEvent.change(screen.getByLabelText("Parola"), {
      target: { value: "admin123" }
    });
    fireEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    await waitFor(() => expect(navigationMocks.replace).toHaveBeenCalledWith("/admin"));
  });

  it("shows a compact credential error", async () => {
    apiMocks.login.mockRejectedValue(new ApiClientError(401, { detail: "Invalid credentials." }));

    render(<LoginScreen />);

    fireEvent.change(screen.getByLabelText("Parola"), {
      target: { value: "wrong" }
    });
    fireEvent.click(screen.getByRole("button", { name: /giriş yap/i }));

    expect(await screen.findByText("E-posta veya parola hatalı.")).toBeInTheDocument();
    expect(navigationMocks.replace).not.toHaveBeenCalled();
  });
});
