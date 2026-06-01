import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { OperationsDashboard } from "@/features/operations/operations-dashboard";
import type { OperationsSummary } from "@/types/api";

const operationsApiMocks = vi.hoisted(() => ({
  getOperationsSummary: vi.fn()
}));

vi.mock("@/features/operations/api", () => ({
  getOperationsSummary: operationsApiMocks.getOperationsSummary
}));

describe("OperationsDashboard", () => {
  beforeEach(() => {
    operationsApiMocks.getOperationsSummary.mockReset();
  });

  it("shows a loading state while metrics are loading", () => {
    operationsApiMocks.getOperationsSummary.mockReturnValue(new Promise(() => undefined));

    render(<OperationsDashboard />);

    expect(screen.getByText("Operasyon metrikleri yükleniyor...")).toBeInTheDocument();
  });

  it("renders operational metrics and supports manual refresh", async () => {
    operationsApiMocks.getOperationsSummary.mockResolvedValue(summaryFixture());

    render(<OperationsDashboard />);

    expect(await screen.findByText("Canlı")).toBeInTheDocument();
    expect(screen.getByText("1 MB")).toBeInTheDocument();
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("15")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /yenile/i }));

    await waitFor(() => expect(operationsApiMocks.getOperationsSummary).toHaveBeenCalledTimes(2));
  });

  it("shows a calm warning for stale listener heartbeat", async () => {
    operationsApiMocks.getOperationsSummary.mockResolvedValue(
      summaryFixture({
        listener: {
          ...summaryFixture().listener,
          is_fresh: false,
          seconds_since_last_seen: 120
        }
      })
    );

    render(<OperationsDashboard />);

    expect(await screen.findByText(/Listener heartbeat bayat/i)).toBeInTheDocument();
    expect(screen.getByText("Bayat")).toBeInTheDocument();
  });

  it("shows a calm error state for failed raw events", async () => {
    operationsApiMocks.getOperationsSummary.mockResolvedValue(
      summaryFixture({
        raw_event_status_counts: {
          pending: 0,
          processing: 0,
          processed: 12,
          failed: 3,
          ignored: 0
        }
      })
    );

    render(<OperationsDashboard />);

    expect(
      await screen.findByText(/Retry edilebilir başarısız raw event var/i)
    ).toBeInTheDocument();
    expect(screen.getByText("İnceleme gerekli")).toBeInTheDocument();
  });

  it("shows an API error state", async () => {
    operationsApiMocks.getOperationsSummary.mockRejectedValue(new Error("API kapalı"));

    render(<OperationsDashboard />);

    expect(await screen.findByText("API kapalı")).toBeInTheDocument();
  });

  it("renders ingestion cards including queue backlog and breaker state", async () => {
    operationsApiMocks.getOperationsSummary.mockResolvedValue(
      summaryFixture({
        ingestion: {
          queue_depth: 1234,
          oldest_pending_age_seconds: 90,
          write_queue_depth: 50,
          write_queue_high_water_mark: 500,
          write_drop_count: 0,
          write_flush_count: 12,
          last_flush_size: 200,
          last_flush_millis: 35,
          clickhouse_insert_failures: 0,
          queue_enqueue_failures: 0,
          breaker_state: "closed",
          breaker_current_delay_ms: 0
        }
      })
    );

    render(<OperationsDashboard />);

    expect(await screen.findByText("1.234")).toBeInTheDocument();
    expect(screen.getByText("Aktif queue")).toBeInTheDocument();
    expect(screen.getByText("Kapalı")).toBeInTheDocument();
  });

  it("warns when ClickHouse circuit breaker is open", async () => {
    operationsApiMocks.getOperationsSummary.mockResolvedValue(
      summaryFixture({
        ingestion: {
          queue_depth: 4000,
          oldest_pending_age_seconds: 300,
          write_queue_depth: 100,
          write_queue_high_water_mark: 1000,
          write_drop_count: 12,
          write_flush_count: 50,
          last_flush_size: 0,
          last_flush_millis: 0,
          clickhouse_insert_failures: 6,
          queue_enqueue_failures: 0,
          breaker_state: "open",
          breaker_current_delay_ms: 5000
        }
      })
    );

    render(<OperationsDashboard />);

    expect(await screen.findByText(/circuit breaker açık/i)).toBeInTheDocument();
    expect(screen.getByText(/event düşürdü/i)).toBeInTheDocument();
  });
});

function summaryFixture(overrides: Partial<OperationsSummary> = {}): OperationsSummary {
  return {
    counts: {
      channels: 4,
      enabled_channels: 2,
      senders: 8,
      messages: 42,
      raw_events: 15
    },
    raw_event_status_counts: {
      pending: 1,
      processing: 0,
      processed: 14,
      failed: 0,
      ignored: 0
    },
    storage: {
      database_bytes: 1048576,
      tables: [
        {
          table_name: "chat_messages",
          total_bytes: 2048
        },
        {
          table_name: "raw_kick_events",
          total_bytes: 4096
        }
      ]
    },
    timestamps: {
      latest_message_at: "2026-05-13T09:15:00Z",
      latest_raw_event_received_at: "2026-05-13T09:14:00Z",
      latest_raw_event_processed_at: "2026-05-13T09:15:30Z",
      oldest_pending_raw_event_received_at: "2026-05-13T09:10:00Z"
    },
    listener: {
      service_name: "listener",
      last_seen_at: "2026-05-13T09:16:00Z",
      is_fresh: true,
      stale_after_seconds: 45,
      seconds_since_last_seen: 5
    },
    ingestion: {
      queue_depth: 0,
      oldest_pending_age_seconds: 0,
      write_queue_depth: 0,
      write_queue_high_water_mark: 0,
      write_drop_count: 0,
      write_flush_count: 0,
      last_flush_size: 0,
      last_flush_millis: 0,
      clickhouse_insert_failures: 0,
      queue_enqueue_failures: 0,
      breaker_state: "closed",
      breaker_current_delay_ms: 0
    },
    ...overrides
  };
}
