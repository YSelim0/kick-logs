import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { RequestPage } from "@/features/requests/request-page";

const apiMocks = vi.hoisted(() => ({
  createUserRequest: vi.fn()
}));

vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

vi.mock("@/features/requests/api", () => ({
  createUserRequest: apiMocks.createUserRequest
}));

describe("RequestPage", () => {
  beforeEach(() => {
    apiMocks.createUserRequest.mockReset();
    apiMocks.createUserRequest.mockResolvedValue({ request_id: "req_123" });
  });

  it("renders the public request page with header request navigation", () => {
    render(<RequestPage />);

    expect(screen.getByRole("heading", { name: "Talep" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Talep" })).toHaveAttribute("href", "/request");
    expect(screen.getByLabelText("Kanal adı")).toBeInTheDocument();
    expect(screen.getByLabelText("Başlık")).toBeInTheDocument();
    expect(screen.getByLabelText("Mesaj")).toBeInTheDocument();
  });

  it("submits a channel request with the normalized payload keys expected by the API", async () => {
    const user = userEvent.setup();
    render(<RequestPage />);

    fireEvent.change(screen.getByLabelText("Kanal adı"), { target: { value: "kick.com/NuriBen" } });
    fireEvent.change(screen.getByLabelText("Başlık"), {
      target: { value: "Kanal takip edilsin" }
    });
    fireEvent.change(screen.getByLabelText("Mesaj"), {
      target: { value: "Bu kanal listede olursa iyi olur." }
    });
    fireEvent.change(screen.getByLabelText(/İletişim/), {
      target: { value: "mod@example.com" }
    });
    await user.click(screen.getByRole("button", { name: "Gönder" }));

    await waitFor(() => expect(apiMocks.createUserRequest).toHaveBeenCalledTimes(1));
    expect(apiMocks.createUserRequest).toHaveBeenCalledWith({
      type: "channel_request",
      title: "Kanal takip edilsin",
      message: "Bu kanal listede olursa iyi olur.",
      channel_slug: "kick.com/NuriBen",
      channel_display_name: "kick.com/NuriBen",
      contact: "mod@example.com",
      website: ""
    });
    expect(await screen.findByText("Talebin alındı.")).toBeInTheDocument();
    expect(screen.getByText("ID req_123")).toBeInTheDocument();
  });

  it("switches to feedback mode and omits channel fields from the request payload", async () => {
    const user = userEvent.setup();
    render(<RequestPage />);

    await user.click(screen.getByRole("button", { name: /Geri Bildirim/ }));

    expect(screen.queryByLabelText("Kanal adı")).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Başlık"), {
      target: { value: "Yeni filtre önerisi" }
    });
    fireEvent.change(screen.getByLabelText("Mesaj"), {
      target: { value: "Arama ekranına yeni bir filtre eklenebilir." }
    });
    await user.click(screen.getByRole("button", { name: "Gönder" }));

    await waitFor(() => expect(apiMocks.createUserRequest).toHaveBeenCalledTimes(1));
    expect(apiMocks.createUserRequest).toHaveBeenCalledWith({
      type: "feedback",
      title: "Yeni filtre önerisi",
      message: "Arama ekranına yeni bir filtre eklenebilir.",
      channel_slug: undefined,
      channel_display_name: undefined,
      contact: undefined,
      website: ""
    });
  });

  it("keeps submit disabled until required fields are ready", async () => {
    const user = userEvent.setup();
    render(<RequestPage />);

    const submit = screen.getByRole("button", { name: "Gönder" });
    expect(submit).toBeDisabled();

    await user.type(screen.getByLabelText("Kanal adı"), "levo");
    await user.type(screen.getByLabelText("Başlık"), "OK");
    await user.type(screen.getByLabelText("Mesaj"), "short enough");

    expect(submit).toBeDisabled();

    await user.clear(screen.getByLabelText("Başlık"));
    await user.type(screen.getByLabelText("Başlık"), "Kanal ekleme isteği");

    expect(submit).toBeEnabled();
  });
});
