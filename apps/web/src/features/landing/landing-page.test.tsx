import { act, cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { HomepageResponse, HomepageSnapshot } from "@/features/landing/api";
import { LandingPage } from "@/features/landing/landing-page";

const homepageMocks = vi.hoisted(() => ({ getHomepage: vi.fn() }));

vi.mock("@/features/landing/api", () => homepageMocks);
vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

describe("LandingPage", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    homepageMocks.getHomepage.mockReset();
    homepageMocks.getHomepage.mockResolvedValue(snapshotFixture());
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("renders all five 14-day panels from one ready snapshot and stops polling", async () => {
    render(<LandingPage />);
    await flush();

    expect(screen.getByText("482")).toBeInTheDocument();
    expect(screen.getByText("Hype")).toBeInTheDocument();
    expect(screen.getByText("Yavuz")).toBeInTheDocument();
    expect(screen.getByText("KEKW")).toBeInTheDocument();
    expect(screen.getAllByText("Son 14 gün")).toHaveLength(5);
    expect(screen.getByRole("heading", { name: "Mesaj hacmi" })).toBeInTheDocument();
    expect(homepageMocks.getHomepage).toHaveBeenCalledExactlyOnceWith(expect.any(AbortSignal));
    await advance(180000);
    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(1);
    expect(vi.getTimerCount()).toBe(0);
  });

  it("keeps the hero, header, footer and profile navigation", async () => {
    render(<LandingPage />);
    await flush();

    expect(screen.getByRole("heading", { name: "Kick chat için kalıcı log." })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Arama başlat" })).toHaveAttribute("href", "/search");
    expect(screen.getByRole("link", { name: "Admin" })).toHaveAttribute("href", "/admin");
    expect(screen.getByRole("link", { name: /Hype/ })).toHaveAttribute("href", "/channels/hype");
    expect(screen.getByRole("link", { name: /Yavuz/ })).toHaveAttribute("href", "/users/yavuz");
    expect(
      screen
        .getAllByRole("link", { name: "GitHub" })
        .every((link) => link.getAttribute("href") === "https://github.com/YSelim0/kick-logs")
    ).toBe(true);
    expect(
      within(screen.getByRole("contentinfo")).getByRole("link", { name: "Talep" })
    ).toHaveAttribute("href", "/request");
  });

  it("shows motion-safe placeholders without zero values or empty hints while pending", () => {
    homepageMocks.getHomepage.mockReturnValue(new Promise(() => {}));
    const { container } = render(<LandingPage />);

    expect(screen.getByLabelText("TOPLAM MESAJ yükleniyor")).toBeInTheDocument();
    expect(screen.getByLabelText("Mesaj hacmi yükleniyor")).toBeInTheDocument();
    expect(screen.getAllByLabelText("Liste yükleniyor")).toHaveLength(3);
    expect(screen.queryByText("0")).not.toBeInTheDocument();
    expect(screen.queryByText(/verisi henüz yok/)).not.toBeInTheDocument();
    const pulses = container.querySelectorAll('[class*="animate-pulse"]');
    expect(pulses.length).toBeGreaterThan(0);
    for (const pulse of pulses) {
      expect(pulse).toHaveClass("motion-safe:animate-pulse", "motion-reduce:animate-none");
      expect(pulse).not.toHaveClass("animate-pulse");
    }
  });

  it("retries initializing after five seconds and stops after ready", async () => {
    homepageMocks.getHomepage.mockResolvedValueOnce({
      status: "initializing",
      retry_after_seconds: 5
    });
    render(<LandingPage />);
    await flush();

    expect(screen.queryByText("0")).not.toBeInTheDocument();
    await advance(4999);
    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(1);
    await advance(1);
    expect(screen.getByText("482")).toBeInTheDocument();
    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(2);
    await advance(180000);
    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(2);
  });

  it.each([
    { value: 1, delay: 5000 },
    { value: 12, delay: 12000 },
    { value: 90, delay: 30000 }
  ])("clamps retry_after_seconds=$value to the allowed delay", async ({ value, delay }) => {
    homepageMocks.getHomepage.mockResolvedValueOnce({
      status: "initializing",
      retry_after_seconds: value
    });
    render(<LandingPage />);
    await flush();
    await advance(delay - 1);
    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(1);
    await advance(1);
    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(2);
    expect(screen.getByText("482")).toBeInTheDocument();
  });

  it("cancels a scheduled retry on unmount", async () => {
    homepageMocks.getHomepage.mockResolvedValue({ status: "initializing", retry_after_seconds: 5 });
    const { unmount } = render(<LandingPage />);
    await flush();
    const signal = homepageMocks.getHomepage.mock.calls[0][0] as AbortSignal;
    unmount();

    expect(signal.aborted).toBe(true);
    expect(vi.getTimerCount()).toBe(0);
    await advance(180000);
    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(1);
  });

  it("aborts in-flight requests and ignores late initializing responses after unmount", async () => {
    let resolve!: (response: HomepageResponse) => void;
    homepageMocks.getHomepage.mockReturnValue(
      new Promise<HomepageResponse>((done) => {
        resolve = done;
      })
    );
    const { unmount } = render(<LandingPage />);
    const signal = homepageMocks.getHomepage.mock.calls[0][0] as AbortSignal;
    unmount();
    await act(async () => resolve({ status: "initializing", retry_after_seconds: 5 }));

    expect(signal.aborted).toBe(true);
    expect(vi.getTimerCount()).toBe(0);
    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(1);
  });

  it("stops after 24 attempts and offers manual retry without fake zeros", async () => {
    homepageMocks.getHomepage.mockResolvedValue({ status: "initializing", retry_after_seconds: 5 });
    render(<LandingPage />);
    await flush();
    await advance(115000);

    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(24);
    expect(screen.getByRole("alert")).toHaveTextContent(/alınamadı/);
    expect(screen.getByRole("button", { name: "Tekrar dene" })).toBeInTheDocument();
    expect(screen.queryByText("0")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("TOPLAM MESAJ yükleniyor")).not.toBeInTheDocument();
    await advance(60000);
    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(24);

    homepageMocks.getHomepage.mockResolvedValue(snapshotFixture());
    fireEvent.click(screen.getByRole("button", { name: "Tekrar dene" }));
    await flush();
    expect(screen.getByText("482")).toBeInTheDocument();
    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(25);
  });

  it("enforces the two-minute deadline even with longer retry delays", async () => {
    homepageMocks.getHomepage.mockResolvedValue({
      status: "initializing",
      retry_after_seconds: 30
    });
    render(<LandingPage />);
    await flush();
    await advance(120000);

    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(4);
    expect(screen.getByRole("button", { name: "Tekrar dene" })).toBeInTheDocument();
    expect(vi.getTimerCount()).toBe(0);
  });

  it("aborts a hung request at the deadline and ignores a late ready response", async () => {
    let resolve!: (response: HomepageResponse) => void;
    homepageMocks.getHomepage.mockReturnValue(
      new Promise<HomepageResponse>((done) => {
        resolve = done;
      })
    );
    render(<LandingPage />);
    const signal = homepageMocks.getHomepage.mock.calls[0][0] as AbortSignal;
    await advance(120000);
    await act(async () => resolve(snapshotFixture()));

    expect(signal.aborted).toBe(true);
    expect(screen.getByRole("button", { name: "Tekrar dene" })).toBeInTheDocument();
    expect(screen.queryByText("482")).not.toBeInTheDocument();
  });

  it("does not retry request failures automatically or fabricate empty data", async () => {
    homepageMocks.getHomepage.mockRejectedValue(new Error("unavailable"));
    render(<LandingPage />);
    await flush();
    await advance(180000);

    expect(screen.getByRole("alert")).toHaveTextContent(/alınamadı/);
    expect(screen.getByRole("button", { name: "Tekrar dene" })).toBeInTheDocument();
    expect(screen.queryByText("0")).not.toBeInTheDocument();
    expect(screen.queryByText(/verisi henüz yok/)).not.toBeInTheDocument();
    expect(homepageMocks.getHomepage).toHaveBeenCalledTimes(1);
  });

  it.each([false, true])("hides snapshot timestamps when stale is %s", async (stale) => {
    homepageMocks.getHomepage.mockResolvedValue({ ...snapshotFixture(), stale });
    const { container } = render(<LandingPage />);
    await flush();

    expect(screen.queryByText(/Veri aralığı \(UTC\)/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Güncelleme \(UTC\)/)).not.toBeInTheDocument();
    expect(container.querySelector("time")).toBeNull();
    expect(screen.getAllByText("Son 14 gün")).toHaveLength(5);
    if (stale) {
      expect(screen.getByRole("status")).toHaveTextContent(/güncel olmayabilir/);
    } else {
      expect(screen.queryByText(/güncel olmayabilir/)).not.toBeInTheDocument();
    }
    expect(screen.getByText("482")).toBeInTheDocument();
  });

  it("renders real empty values and retains the zero-day chart baseline in UTC", async () => {
    homepageMocks.getHomepage.mockResolvedValue({
      ...snapshotFixture(),
      overview: {
        total_messages: 0,
        total_senders: 0,
        total_channels: 0,
        total_emote_usages: 0,
        first_message_at: null,
        latest_message_at: null
      },
      message_volume: [{ bucket_start: "2026-05-14T23:00:00Z", message_count: 0 }],
      top_channels: [],
      top_senders: [],
      top_emotes: []
    });
    render(<LandingPage />);
    await flush();

    expect(
      within(screen.getByRole("region", { name: "Genel metrikler" })).getAllByText("0")
    ).toHaveLength(4);
    expect(screen.getByLabelText("14 May · 0 mesaj")).toHaveStyle({ height: "2%" });
    expect(screen.getByText("Kanal verisi henüz yok.")).toBeInTheDocument();
    expect(screen.getByText("Kullanıcı verisi henüz yok.")).toBeInTheDocument();
    expect(screen.getByText("Emote verisi henüz yok.")).toBeInTheDocument();
  });
});

async function flush() {
  await act(async () => {});
}

async function advance(milliseconds: number) {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(milliseconds);
  });
}

function snapshotFixture(): HomepageSnapshot {
  return {
    status: "ready",
    as_of: "2026-05-15T00:05:00Z",
    start: "2026-05-01T00:00:00Z",
    end: "2026-05-15T00:00:00Z",
    timezone: "UTC",
    stale: false,
    overview: {
      total_messages: 482,
      total_senders: 76,
      total_channels: 5,
      total_emote_usages: 314,
      first_message_at: "2026-05-01T10:00:00Z",
      latest_message_at: "2026-05-14T09:30:00Z"
    },
    message_volume: [
      { bucket_start: "2026-05-13T00:00:00Z", message_count: 40 },
      { bucket_start: "2026-05-14T00:00:00Z", message_count: 60 }
    ],
    top_channels: [
      {
        channel_id: 1,
        slug: "hype",
        display_name: "Hype",
        profile_image_url: null,
        banner_image_url: null,
        message_count: 400,
        first_message_at: "2026-05-01T10:00:00Z",
        latest_message_at: "2026-05-14T09:30:00Z"
      }
    ],
    top_senders: [
      {
        sender_id: 1,
        kick_user_id: 10,
        username: "Yavuz",
        slug: "yavuz",
        profile_image_url: null,
        message_count: 120,
        first_message_at: "2026-05-01T10:00:00Z",
        latest_message_at: "2026-05-14T09:30:00Z"
      }
    ],
    top_emotes: [
      {
        id: "37226",
        name: "KEKW",
        token: "[emote:37226:KEKW]",
        image_url: "https://files.kick.com/emotes/37226/fullsize",
        usage_count: 99,
        message_count: 80
      }
    ]
  };
}
