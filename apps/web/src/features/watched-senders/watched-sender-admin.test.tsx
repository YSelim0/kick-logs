import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WatchedSenderAdmin } from "@/features/watched-senders/watched-sender-admin";
import type { WatchedSender } from "@/types/api";

const watchedSenderApiMocks = vi.hoisted(() => ({
  addWatchedSender: vi.fn(),
  listWatchedSenders: vi.fn(),
  removeWatchedSender: vi.fn(),
  getNotificationSettings: vi.fn(),
  updateNotificationSettings: vi.fn()
}));

vi.mock("@/features/watched-senders/api", () => ({
  addWatchedSender: watchedSenderApiMocks.addWatchedSender,
  listWatchedSenders: watchedSenderApiMocks.listWatchedSenders,
  removeWatchedSender: watchedSenderApiMocks.removeWatchedSender,
  getNotificationSettings: watchedSenderApiMocks.getNotificationSettings,
  updateNotificationSettings: watchedSenderApiMocks.updateNotificationSettings
}));

describe("WatchedSenderAdmin", () => {
  beforeEach(() => {
    watchedSenderApiMocks.addWatchedSender.mockReset();
    watchedSenderApiMocks.listWatchedSenders.mockReset();
    watchedSenderApiMocks.removeWatchedSender.mockReset();
    watchedSenderApiMocks.getNotificationSettings.mockReset();
    watchedSenderApiMocks.updateNotificationSettings.mockReset();
    watchedSenderApiMocks.getNotificationSettings.mockResolvedValue({ cooldown_seconds: 600 });
  });

  it("lists watched senders", async () => {
    watchedSenderApiMocks.listWatchedSenders.mockResolvedValue([senderFixture()]);

    render(<WatchedSenderAdmin />);

    expect(await screen.findAllByText("@nuriben")).not.toHaveLength(0);
  });

  it("shows an empty state when no senders are watched", async () => {
    watchedSenderApiMocks.listWatchedSenders.mockResolvedValue([]);

    render(<WatchedSenderAdmin />);

    expect(await screen.findByText("Henüz izlenen kullanıcı yok.")).toBeInTheDocument();
  });

  it("adds a watched sender by username", async () => {
    const sender = senderFixture();
    watchedSenderApiMocks.listWatchedSenders.mockResolvedValue([]);
    watchedSenderApiMocks.addWatchedSender.mockResolvedValue(sender);

    render(<WatchedSenderAdmin />);

    expect(await screen.findByText("Henüz izlenen kullanıcı yok.")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Kick kullanıcı adı"), {
      target: { value: " nuriben " }
    });
    fireEvent.click(screen.getByRole("button", { name: /ekle/i }));

    await waitFor(() =>
      expect(watchedSenderApiMocks.addWatchedSender).toHaveBeenCalledWith({
        username: "nuriben"
      })
    );
    expect(await screen.findAllByText("@nuriben")).not.toHaveLength(0);
  });

  it("removes a watched sender", async () => {
    watchedSenderApiMocks.listWatchedSenders.mockResolvedValue([senderFixture()]);
    watchedSenderApiMocks.removeWatchedSender.mockResolvedValue({ status: "ok" });

    render(<WatchedSenderAdmin />);

    expect(await screen.findAllByText("@nuriben")).not.toHaveLength(0);
    fireEvent.click(screen.getAllByRole("button", { name: /kaldır/i })[0]);

    await waitFor(() => expect(watchedSenderApiMocks.removeWatchedSender).toHaveBeenCalledWith(1));
    await waitFor(() => expect(screen.queryAllByText("@nuriben")).toHaveLength(0));
  });

  it("shows the current cooldown in minutes", async () => {
    watchedSenderApiMocks.listWatchedSenders.mockResolvedValue([]);
    watchedSenderApiMocks.getNotificationSettings.mockResolvedValue({ cooldown_seconds: 600 });

    render(<WatchedSenderAdmin />);

    expect(await screen.findByLabelText("Bekleme süresi (dakika)")).toHaveValue(10);
  });

  it("saves a new cooldown value in minutes", async () => {
    watchedSenderApiMocks.listWatchedSenders.mockResolvedValue([]);
    watchedSenderApiMocks.getNotificationSettings.mockResolvedValue({ cooldown_seconds: 600 });
    watchedSenderApiMocks.updateNotificationSettings.mockResolvedValue({ cooldown_seconds: 300 });

    render(<WatchedSenderAdmin />);

    const input = await screen.findByLabelText("Bekleme süresi (dakika)");
    fireEvent.change(input, { target: { value: "5" } });
    fireEvent.click(screen.getByRole("button", { name: /kaydet/i }));

    await waitFor(() =>
      expect(watchedSenderApiMocks.updateNotificationSettings).toHaveBeenCalledWith({
        cooldown_seconds: 300
      })
    );
  });
});

function senderFixture(): WatchedSender {
  return {
    id: 1,
    username: "nuriben",
    created_at: "2026-09-04T00:00:00.000Z"
  };
}
