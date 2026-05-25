import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PredictionSearchPage } from "@/features/prediction/prediction-search-page";

const navigationMocks = vi.hoisted(() => ({
  push: vi.fn()
}));

vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: navigationMocks.push })
}));

describe("PredictionSearchPage", () => {
  beforeEach(() => {
    navigationMocks.push.mockReset();
  });

  it("renders the idle prompt and search form", () => {
    render(<PredictionSearchPage />);

    expect(screen.getByText("Tahmin verisi için kanal seçin")).toBeInTheDocument();
    expect(screen.getByRole("searchbox", { name: /kanal adı/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /göster/i })).toBeInTheDocument();
  });

  it("disables submit until the query has at least 2 characters", async () => {
    const user = userEvent.setup();
    render(<PredictionSearchPage />);

    const submit = screen.getByRole("button", { name: /göster/i });
    expect(submit).toBeDisabled();

    await user.type(screen.getByRole("searchbox", { name: /kanal adı/i }), "n");
    expect(submit).toBeDisabled();

    await user.type(screen.getByRole("searchbox", { name: /kanal adı/i }), "b");
    expect(submit).not.toBeDisabled();
  });

  it("navigates to the trimmed lowercased slug on submit", async () => {
    const user = userEvent.setup();
    render(<PredictionSearchPage />);

    await user.type(screen.getByRole("searchbox", { name: /kanal adı/i }), "  NuriBen  ");
    await user.click(screen.getByRole("button", { name: /göster/i }));

    expect(navigationMocks.push).toHaveBeenCalledWith("/prediction/nuriben");
  });

  it("navigates on Enter key", async () => {
    const user = userEvent.setup();
    render(<PredictionSearchPage />);

    await user.type(screen.getByRole("searchbox", { name: /kanal adı/i }), "hype{Enter}");

    expect(navigationMocks.push).toHaveBeenCalledWith("/prediction/hype");
  });
});
