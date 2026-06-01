import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DataManagementPanel } from "@/features/data-management/data-management-panel";
import type { DataCleanupPreview, DataCleanupResult, DataManagementSummary } from "@/types/api";

const dataMocks = vi.hoisted(() => ({
  confirmDataCleanup: vi.fn(),
  getDataManagementSummary: vi.fn(),
  previewDataCleanup: vi.fn(),
  updateRetentionSettings: vi.fn()
}));

vi.mock("@/features/data-management/api", () => dataMocks);

describe("DataManagementPanel", () => {
  beforeEach(() => {
    dataMocks.confirmDataCleanup.mockReset();
    dataMocks.getDataManagementSummary.mockReset();
    dataMocks.previewDataCleanup.mockReset();
    dataMocks.updateRetentionSettings.mockReset();
    dataMocks.getDataManagementSummary.mockResolvedValue(summaryFixture());
    dataMocks.updateRetentionSettings.mockResolvedValue({
      message_retention_days: 30,
      raw_event_retention_days: 90,
      updated_at: "2026-05-15T10:00:00Z"
    });
    dataMocks.previewDataCleanup.mockResolvedValue(previewFixture());
    dataMocks.confirmDataCleanup.mockResolvedValue(resultFixture());
  });

  it("shows storage summary and current retention settings", async () => {
    render(<DataManagementPanel />);

    expect(await screen.findByRole("heading", { name: "Veri Yönetimi" })).toBeInTheDocument();
    expect(screen.getByText("Veritabanı")).toBeInTheDocument();
    expect(screen.getByText("chat_messages")).toBeInTheDocument();
    expect(screen.getByText("raw_kick_events")).toBeInTheDocument();
    expect(screen.getByText("raw_event_queue")).toBeInTheDocument();
    expect(screen.getByText("kick_webhook_events")).toBeInTheDocument();
    expect(screen.getByText("raw_event_claims")).toBeInTheDocument();
    expect(screen.getByLabelText("Mesajlar")).toHaveValue("forever");
    expect(screen.getByLabelText("Raw Eventler")).toHaveValue("90");
  });

  it("updates retention settings", async () => {
    render(<DataManagementPanel />);

    await screen.findByRole("heading", { name: "Veri Yönetimi" });
    fireEvent.change(screen.getByLabelText("Mesajlar"), { target: { value: "30" } });
    fireEvent.change(screen.getByLabelText("Raw Eventler"), { target: { value: "90" } });
    fireEvent.click(screen.getByRole("button", { name: /kaydet/i }));

    await waitFor(() =>
      expect(dataMocks.updateRetentionSettings).toHaveBeenCalledWith({
        message_retention_days: 30,
        raw_event_retention_days: 90
      })
    );
  });

  it("shows dry-run preview and blocks deletion until confirmation matches", async () => {
    render(<DataManagementPanel />);

    await screen.findByRole("heading", { name: "Veri Yönetimi" });
    fireEvent.click(screen.getByRole("button", { name: /dry-run/i }));

    expect(await screen.findByText("Cleanup Önizleme Sonucu")).toBeInTheDocument();
    expect(dataMocks.previewDataCleanup).toHaveBeenCalledWith({
      target: "old_messages",
      channel_slug: null,
      sender: null
    });
    expect(screen.getByText("DELETE OLD MESSAGES")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^sil$/i })).toBeDisabled();

    fireEvent.change(screen.getByPlaceholderText("DELETE OLD MESSAGES"), {
      target: { value: "DELETE" }
    });
    expect(screen.getByRole("button", { name: /^sil$/i })).toBeDisabled();

    fireEvent.change(screen.getByPlaceholderText("DELETE OLD MESSAGES"), {
      target: { value: "DELETE OLD MESSAGES" }
    });
    expect(screen.getByRole("button", { name: /^sil$/i })).toBeEnabled();
  });

  it("confirms cleanup and shows deleted counts", async () => {
    render(<DataManagementPanel />);

    await screen.findByRole("heading", { name: "Veri Yönetimi" });
    fireEvent.click(screen.getByRole("button", { name: /dry-run/i }));
    await screen.findByText("Cleanup Önizleme Sonucu");
    fireEvent.change(screen.getByPlaceholderText("DELETE OLD MESSAGES"), {
      target: { value: "DELETE OLD MESSAGES" }
    });
    fireEvent.click(screen.getByRole("button", { name: /^sil$/i }));

    await waitFor(() =>
      expect(dataMocks.confirmDataCleanup).toHaveBeenCalledWith({
        target: "old_messages",
        channel_slug: null,
        sender: null,
        confirmation_text: "DELETE OLD MESSAGES"
      })
    );
    expect(
      await screen.findByText(/Cleanup tamamlandı: 2 mesaj, 1 raw event silindi./)
    ).toBeInTheDocument();
  });

  it("shows API errors", async () => {
    dataMocks.getDataManagementSummary.mockRejectedValue(new Error("API down"));

    render(<DataManagementPanel />);

    expect(await screen.findByText("API down")).toBeInTheDocument();
  });
});

function summaryFixture(): DataManagementSummary {
  return {
    counts: {
      channels: 2,
      senders: 3,
      messages: 1200,
      raw_events: 1400
    },
    database_bytes: 1048576,
    retention_settings: {
      message_retention_days: null,
      raw_event_retention_days: 90,
      updated_at: "2026-05-15T09:00:00Z"
    },
    tables: [
      { table_name: "chat_messages", row_count: 1200, total_bytes: 512000 },
      { table_name: "raw_kick_events", row_count: 1400, total_bytes: 600000 },
      { table_name: "raw_event_queue", row_count: 4, total_bytes: 0 },
      { table_name: "kick_webhook_events", row_count: 8, total_bytes: 0 },
      { table_name: "raw_event_claims", row_count: 1200, total_bytes: 0 }
    ]
  };
}

function previewFixture(): DataCleanupPreview {
  return {
    target: "old_messages",
    affected: {
      messages: 2,
      raw_events: 0,
      total: 2
    },
    confirmation_text: "DELETE OLD MESSAGES",
    can_execute: true,
    cutoff_at: "2026-04-15T10:00:00Z",
    channel_slug: null,
    sender: null,
    retention_days: 30,
    reason: null
  };
}

function resultFixture(): DataCleanupResult {
  return {
    target: "old_messages",
    deleted: {
      messages: 2,
      raw_events: 1,
      total: 3
    },
    confirmation_text: "DELETE OLD MESSAGES",
    cutoff_at: "2026-04-15T10:00:00Z",
    channel_slug: null,
    sender: null,
    retention_days: 30
  };
}
