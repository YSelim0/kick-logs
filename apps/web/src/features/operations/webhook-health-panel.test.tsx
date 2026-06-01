import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WebhookHealthPanel } from "@/features/operations/webhook-health-panel";
import type { WebhookHealth } from "@/types/api";

const opsMocks = vi.hoisted(() => ({
  getWebhookHealth: vi.fn(),
  triggerWebhookSync: vi.fn()
}));

vi.mock("@/features/operations/api", () => ({
  ...opsMocks,
  getOperationsSummary: vi.fn(),
  getFailedEvents: vi.fn(),
  retryFailedEvents: vi.fn(),
  clearFailedEvents: vi.fn()
}));

function healthFixture(overrides: Partial<WebhookHealth> = {}): WebhookHealth {
  return {
    configured_event_types: [
      "channel.subscription.new",
      "channel.subscription.renewal",
      "channel.subscription.gifts"
    ],
    missing_client_credentials: false,
    missing_webhook_public_key: false,
    webhook_sync_enabled: true,
    latest_webhook_received_at: "2026-06-01T12:00:00Z",
    inbox_counts: { pending: 2, processed: 100, failed: 0, ignored: 3 },
    channels: [
      {
        followed_channel_id: 1,
        slug: "hype",
        broadcaster_user_id: 9000,
        subscriptions: [
          {
            event_type: "channel.subscription.new",
            kick_subscription_id: "sub-123",
            status: "active",
            latest_sync_error: null,
            synced_at: "2026-06-01T10:00:00Z"
          },
          {
            event_type: "channel.subscription.renewal",
            kick_subscription_id: "sub-456",
            status: "active",
            latest_sync_error: null,
            synced_at: "2026-06-01T10:00:00Z"
          },
          {
            event_type: "channel.subscription.gifts",
            kick_subscription_id: "sub-789",
            status: "active",
            latest_sync_error: null,
            synced_at: "2026-06-01T10:00:00Z"
          }
        ]
      }
    ],
    ...overrides
  };
}

describe("WebhookHealthPanel", () => {
  beforeEach(() => {
    opsMocks.getWebhookHealth.mockReset();
    opsMocks.triggerWebhookSync.mockReset();
    opsMocks.getWebhookHealth.mockResolvedValue(healthFixture());
    opsMocks.triggerWebhookSync.mockResolvedValue({ status: "sync triggered" });
  });

  it("renders inbox counts", async () => {
    render(<WebhookHealthPanel />);
    await waitFor(() => expect(opsMocks.getWebhookHealth).toHaveBeenCalled());
    expect(await screen.findByText("Inbox")).toBeInTheDocument();
    expect(screen.getByText("100")).toBeInTheDocument();
  });

  it("shows channel sync summary and details", async () => {
    const user = userEvent.setup();
    render(<WebhookHealthPanel />);
    await waitFor(() => expect(opsMocks.getWebhookHealth).toHaveBeenCalled());
    expect(await screen.findByText("hype")).toBeInTheDocument();
    expect(screen.getByText("9000")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /aktif/i }));
    expect(screen.getByRole("heading", { name: /webhook detayı/i })).toBeInTheDocument();
    expect(screen.getByText("new")).toBeInTheDocument();
    expect(screen.getByText("renewal")).toBeInTheDocument();
    expect(screen.getByText("gifts")).toBeInTheDocument();
  });

  it("separates inactive subscriptions from errors", async () => {
    const user = userEvent.setup();
    opsMocks.getWebhookHealth.mockResolvedValue(
      healthFixture({
        channels: [
          {
            followed_channel_id: 1,
            slug: "hype",
            broadcaster_user_id: 9000,
            subscriptions: [
              {
                event_type: "channel.subscription.new",
                kick_subscription_id: "sub-123",
                status: "active",
                latest_sync_error: null,
                synced_at: "2026-06-01T10:00:00Z"
              }
            ]
          }
        ]
      })
    );
    render(<WebhookHealthPanel />);
    const inactiveButton = await screen.findByRole("button", { name: /aktif değil/i });
    await user.click(inactiveButton);
    expect(screen.getAllByText("aktif değil").length).toBeGreaterThanOrEqual(3);
  });

  it("summarizes sync errors by count", async () => {
    opsMocks.getWebhookHealth.mockResolvedValue(
      healthFixture({
        channels: [
          {
            followed_channel_id: 1,
            slug: "hype",
            broadcaster_user_id: 9000,
            subscriptions: [
              {
                event_type: "channel.subscription.new",
                kick_subscription_id: "",
                status: "deleted",
                latest_sync_error: "create event subscription returned status 429",
                synced_at: null
              },
              {
                event_type: "channel.subscription.renewal",
                kick_subscription_id: "sub-456",
                status: "active",
                latest_sync_error: null,
                synced_at: "2026-06-01T10:00:00Z"
              },
              {
                event_type: "channel.subscription.gifts",
                kick_subscription_id: "sub-789",
                status: "active",
                latest_sync_error: null,
                synced_at: "2026-06-01T10:00:00Z"
              }
            ]
          }
        ]
      })
    );
    render(<WebhookHealthPanel />);
    expect(await screen.findByRole("button", { name: /1 hata/i })).toBeInTheDocument();
  });

  it("shows config warnings when credentials missing", async () => {
    opsMocks.getWebhookHealth.mockResolvedValue(
      healthFixture({ missing_client_credentials: true, missing_webhook_public_key: true })
    );
    render(<WebhookHealthPanel />);
    expect(await screen.findByText(/client credentials eksik/i)).toBeInTheDocument();
    expect(screen.getByText(/public key eksik/i)).toBeInTheDocument();
  });

  it("shows failed inbox count in danger tone", async () => {
    opsMocks.getWebhookHealth.mockResolvedValue(
      healthFixture({ inbox_counts: { pending: 0, processed: 10, failed: 5, ignored: 0 } })
    );
    render(<WebhookHealthPanel />);
    await waitFor(() => expect(opsMocks.getWebhookHealth).toHaveBeenCalled());
    const failedCell = await screen.findByText("5");
    expect(failedCell).toHaveClass("text-danger");
  });

  it("triggers sync and refreshes on sync button click", async () => {
    const user = userEvent.setup();
    render(<WebhookHealthPanel />);
    await waitFor(() => expect(opsMocks.getWebhookHealth).toHaveBeenCalledTimes(1));
    const syncButton = await screen.findByRole("button", { name: /senkronize et/i });
    await user.click(syncButton);
    expect(opsMocks.triggerWebhookSync).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(opsMocks.getWebhookHealth).toHaveBeenCalledTimes(2));
  });

  it("shows error on fetch failure", async () => {
    opsMocks.getWebhookHealth.mockRejectedValue(new Error("network error"));
    render(<WebhookHealthPanel />);
    expect(await screen.findByText(/alınamadı/i)).toBeInTheDocument();
  });
});
