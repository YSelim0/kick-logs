import { act, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ChannelsIndexPage } from "@/features/channel-profile/channels-index-page";
import { UsersIndexPage } from "@/features/user-profile/users-index-page";

const mocks = vi.hoisted(() => ({ getDirectoryUsers: vi.fn(), getDirectoryChannels: vi.fn() }));
vi.mock("@/features/directory/api", () => mocks);
vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

function page(name = "alpha", next: string | null = null, id = 1) {
  return { items: [{ id, name, slug: name, profile_image_url: null }], next_cursor: next };
}

function deferred() {
  let resolve!: (value: ReturnType<typeof page>) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<ReturnType<typeof page>>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}

describe.each([
  { kind: "users", Component: UsersIndexPage, api: mocks.getDirectoryUsers },
  { kind: "channels", Component: ChannelsIndexPage, api: mocks.getDirectoryChannels }
])("$kind directory interaction", ({ kind, Component, api }) => {
  beforeEach(() => {
    vi.resetAllMocks();
    api.mockResolvedValue(page());
  });

  it("makes no idle or typing requests, even after a debounce interval", () => {
    vi.useFakeTimers();
    try {
      render(<Component />);
      act(() => vi.advanceTimersByTime(10000));
      expect(api).not.toHaveBeenCalled();
      expect(screen.queryByRole("link", { name: "Talep gönder" })).not.toBeInTheDocument();
      fireEvent.change(screen.getByRole("searchbox"), { target: { value: "alpha" } });
      act(() => vi.advanceTimersByTime(10000));
      expect(api).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  it("gates empty, short, overlong and control-character prefixes", () => {
    render(<Component />);
    const input = screen.getByRole("searchbox");
    const submit = screen.getByRole("button", { name: "Ara" });
    for (const value of ["", " ", " a ", "a".repeat(161), "ab\u0000"]) {
      fireEvent.change(input, { target: { value } });
      expect(submit).toBeDisabled();
      fireEvent.submit(input.closest("form")!);
      expect(api).not.toHaveBeenCalled();
    }
    fireEvent.change(input, { target: { value: " ab " } });
    expect(submit).toBeEnabled();
  });

  it("shows loading and prevents duplicate submissions of the in-flight prefix", async () => {
    const pending = deferred();
    api.mockReturnValue(pending.promise);
    render(<Component />);
    const input = screen.getByRole("searchbox");
    fireEvent.change(input, { target: { value: "alpha" } });
    fireEvent.submit(input.closest("form")!);
    expect(screen.getByRole("button", { name: /Aranıyor/ })).toBeDisabled();
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Talep gönder" })).not.toBeInTheDocument();
    fireEvent.submit(input.closest("form")!);
    expect(api).toHaveBeenCalledTimes(1);
    await act(async () => pending.resolve(page()));
    expect(screen.getByRole("link", { name: /alpha/ })).toBeInTheDocument();
  });

  it("quotes the submitted prefix in empty results, not unsent input", async () => {
    api.mockResolvedValue({ items: [], next_cursor: null });
    const user = userEvent.setup();
    render(<Component />);
    const input = screen.getByRole("searchbox");
    await user.type(input, "missing{Enter}");
    expect(await screen.findByText(/"missing".*bulunamadı/)).toBeInTheDocument();
    if (kind === "channels") {
      expect(screen.getByRole("link", { name: "Talep gönder" })).toHaveAttribute(
        "href",
        "/request"
      );
      expect(screen.getByText(/en geç 6 saat içinde incelenir/)).toBeInTheDocument();
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
      await user.keyboard("{Escape}");
      expect(screen.getByRole("link", { name: "Talep gönder" })).toBeInTheDocument();
    } else {
      expect(screen.queryByRole("link", { name: "Talep gönder" })).not.toBeInTheDocument();
    }
    await user.clear(input);
    await user.type(input, "unsent");
    expect(screen.getByText(/"missing".*bulunamadı/)).toBeInTheDocument();
    expect(api).toHaveBeenCalledTimes(1);
  });

  it("shows a recoverable first-page error and clears it after resubmit", async () => {
    api.mockRejectedValueOnce(new Error("offline"));
    const user = userEvent.setup();
    render(<Component />);
    await user.type(screen.getByRole("searchbox"), "alpha{Enter}");
    expect(await screen.findByRole("alert")).toHaveTextContent(/Sonuçlar alınamadı/);
    expect(screen.queryByRole("link", { name: "Talep gönder" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Ara" }));
    expect(await screen.findByRole("link", { name: /alpha/ })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("clears the empty-result request prompt when a new search starts", async () => {
    const pending = deferred();
    api
      .mockResolvedValueOnce({ items: [], next_cursor: null })
      .mockReturnValueOnce(pending.promise);
    const user = userEvent.setup();
    render(<Component />);
    const input = screen.getByRole("searchbox");
    await user.type(input, "missing{Enter}");
    await screen.findByText(/"missing".*bulunamadı/);
    if (kind === "channels") {
      expect(screen.getByRole("link", { name: "Talep gönder" })).toBeInTheDocument();
    }
    await user.clear(input);
    await user.type(input, "alpha{Enter}");
    expect(screen.queryByText(/"missing".*bulunamadı/)).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Talep gönder" })).not.toBeInTheDocument();
    await act(async () => pending.resolve(page()));
    expect(screen.getByRole("link", { name: /alpha/ })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Talep gönder" })).not.toBeInTheDocument();
  });

  it("appends cursor pages for the submitted prefix while retaining previous rows on retry", async () => {
    api
      .mockResolvedValueOnce(page("alpha", "page-two"))
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(page("alpha-two", null, 2));
    const user = userEvent.setup();
    render(<Component />);
    const input = screen.getByRole("searchbox");
    await user.type(input, "alpha{Enter}");
    await screen.findByRole("link", { name: /alpha/ });
    await user.clear(input);
    await user.type(input, "unsent");
    await user.click(screen.getByRole("button", { name: "Daha fazla yükle" }));
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /alpha/ })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Daha fazla yükle" }));
    expect(await screen.findByRole("link", { name: /alpha-two/ })).toBeInTheDocument();
    expect(screen.getAllByRole("link", { name: /alpha/ })).toHaveLength(2);
    expect(api).toHaveBeenLastCalledWith(
      { prefix: "alpha", limit: 50, after: "page-two" },
      expect.anything()
    );
    expect(screen.queryByRole("button", { name: "Daha fazla yükle" })).not.toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("keeps existing rows when the final page is empty and prevents duplicate append requests", async () => {
    const pending = deferred();
    api.mockResolvedValueOnce(page("alpha", "page-two")).mockReturnValueOnce(pending.promise);
    const user = userEvent.setup();
    render(<Component />);
    await user.type(screen.getByRole("searchbox"), "alpha{Enter}");
    const more = await screen.findByRole("button", { name: "Daha fazla yükle" });
    await user.click(more);
    fireEvent.click(more);
    expect(api).toHaveBeenCalledTimes(2);
    expect(more).toBeDisabled();
    await act(async () => pending.resolve({ items: [], next_cursor: null }));
    expect(screen.getByRole("link", { name: /alpha/ })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Talep gönder" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Daha fazla yükle" })).not.toBeInTheDocument();
  });

  it.each(["resolve", "reject"] as const)(
    "ignores stale first-page %s after a newer submitted search",
    async (outcome) => {
      const old = deferred();
      api.mockReturnValueOnce(old.promise).mockResolvedValueOnce(page("beta"));
      render(<Component />);
      const input = screen.getByRole("searchbox");
      fireEvent.change(input, { target: { value: "alpha" } });
      fireEvent.submit(input.closest("form")!);
      const signal = api.mock.calls[0][1].signal as AbortSignal;
      fireEvent.change(input, { target: { value: "beta" } });
      fireEvent.submit(input.closest("form")!);
      expect(await screen.findByRole("link", { name: /beta/ })).toBeInTheDocument();
      expect(signal.aborted).toBe(true);
      await act(async () => {
        if (outcome === "resolve") old.resolve(page());
        else old.reject(new Error("old"));
      });
      expect(screen.getByRole("link", { name: /beta/ })).toBeInTheDocument();
      expect(screen.queryByRole("link", { name: /alpha/ })).not.toBeInTheDocument();
      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    }
  );

  it.each(["resolve", "reject"] as const)(
    "ignores stale append %s after a replacement search",
    async (outcome) => {
      const old = deferred();
      api
        .mockResolvedValueOnce(page("alpha", "page-two"))
        .mockReturnValueOnce(old.promise)
        .mockResolvedValueOnce(page("beta"));
      const user = userEvent.setup();
      render(<Component />);
      const input = screen.getByRole("searchbox");
      await user.type(input, "alpha{Enter}");
      await user.click(await screen.findByRole("button", { name: "Daha fazla yükle" }));
      await user.clear(input);
      await user.type(input, "beta{Enter}");
      expect(await screen.findByRole("link", { name: /beta/ })).toBeInTheDocument();
      await act(async () => {
        if (outcome === "resolve") old.resolve(page("alpha-two", null, 2));
        else old.reject(new Error("old"));
      });
      expect(screen.getByRole("link", { name: /beta/ })).toBeInTheDocument();
      expect(screen.queryByRole("link", { name: /alpha/ })).not.toBeInTheDocument();
      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    }
  );

  it("aborts an in-flight request on unmount", async () => {
    const pending = deferred();
    api.mockReturnValue(pending.promise);
    const { unmount } = render(<Component />);
    const input = screen.getByRole("searchbox");
    fireEvent.change(input, { target: { value: "alpha" } });
    fireEvent.submit(input.closest("form")!);
    const signal = api.mock.calls[0][1].signal as AbortSignal;
    unmount();
    expect(signal.aborted).toBe(true);
    await act(async () => pending.resolve(page()));
    expect(api).toHaveBeenCalledTimes(1);
  });
});
