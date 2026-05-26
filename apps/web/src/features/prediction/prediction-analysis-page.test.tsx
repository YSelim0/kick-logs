import { act, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { PredictionAnalysisPage } from "@/features/prediction/prediction-analysis-page";
import { ApiClientError } from "@/lib/api-client";
import type { Prediction } from "@/types/api";

const apiMocks = vi.hoisted(() => ({
  getPrediction: vi.fn()
}));

vi.mock("next/image", () => ({
  default: ({ alt }: { alt: string }) => <span aria-label={alt} role="img" />
}));

vi.mock("@/features/prediction/api", () => apiMocks);

// Charts depend on recharts ResponsiveContainer which has no layout in jsdom; stub them.
vi.mock("@/features/prediction/prediction-distribution-chart", () => ({
  PredictionDistributionChart: () => <div data-testid="distribution-chart" />
}));
vi.mock("@/features/prediction/prediction-vote-return-chart", () => ({
  PredictionVoteReturnChart: () => <div data-testid="vote-return-chart" />
}));
vi.mock("@/features/prediction/prediction-top-users-chart", () => ({
  PredictionTopUsersChart: () => <div data-testid="top-users-chart" />
}));

describe("PredictionAnalysisPage", () => {
  beforeEach(() => {
    apiMocks.getPrediction.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the summary, state pill, and winner badge when data resolves", async () => {
    apiMocks.getPrediction.mockResolvedValue(predictionFixture());

    render(<PredictionAnalysisPage slug="nuriben" />);

    expect(await screen.findByText("mac bitis suresi")).toBeInTheDocument();
    expect(screen.getByText("Sonuçlandı")).toBeInTheDocument();
    expect(screen.getByText("KAZANAN")).toBeInTheDocument();
    expect(screen.getByTestId("distribution-chart")).toBeInTheDocument();
    expect(screen.getByTestId("top-users-chart")).toBeInTheDocument();
  });

  it("shows the not-found state on a 404", async () => {
    apiMocks.getPrediction.mockRejectedValue(new ApiClientError(404, { detail: "missing" }));

    render(<PredictionAnalysisPage slug="nuriben" />);

    await waitFor(() => expect(screen.getByText(/aktif tahmin bulunamadı/i)).toBeInTheDocument());
  });

  it("shows the error state on a non-404 failure", async () => {
    apiMocks.getPrediction.mockRejectedValue(new ApiClientError(502, { detail: "blocked" }));

    render(<PredictionAnalysisPage slug="nuriben" />);

    await waitFor(() => expect(screen.getByText(/tahmin verisi alınamadı/i)).toBeInTheDocument());
  });

  it("refreshes active predictions in the background without returning to loading", async () => {
    vi.useFakeTimers();
    apiMocks.getPrediction
      .mockResolvedValueOnce(predictionFixture({ state: "ACTIVE", totalPoints: 100 }))
      .mockResolvedValueOnce(predictionFixture({ state: "ACTIVE", totalPoints: 125 }));

    render(<PredictionAnalysisPage slug="nuriben" />);

    await flushPromises();
    expect(screen.getByText("Aktif")).toBeInTheDocument();
    expect(apiMocks.getPrediction).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });

    await flushPromises();
    expect(apiMocks.getPrediction).toHaveBeenCalledTimes(2);
    expect(screen.queryByText("tahmin verisi yükleniyor…")).not.toBeInTheDocument();
    expect(screen.getByText("125")).toBeInTheDocument();
  });

  it("keeps refreshing after a terminal state so later result transitions are visible", async () => {
    vi.useFakeTimers();
    apiMocks.getPrediction
      .mockResolvedValueOnce(predictionFixture({ state: "ACTIVE" }))
      .mockResolvedValueOnce(predictionFixture({ state: "RESOLVED" }))
      .mockResolvedValueOnce(predictionFixture({ state: "RESOLVED", totalPoints: 140 }));

    render(<PredictionAnalysisPage slug="nuriben" />);

    await flushPromises();
    expect(screen.getByText("Aktif")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });

    await flushPromises();
    expect(screen.getByText("Sonuçlandı")).toBeInTheDocument();
    expect(apiMocks.getPrediction).toHaveBeenCalledTimes(2);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });

    await flushPromises();
    expect(apiMocks.getPrediction).toHaveBeenCalledTimes(3);
    expect(screen.getByText("140")).toBeInTheDocument();
  });
});

function predictionFixture(overrides: Partial<Prediction> = {}): Prediction {
  return {
    id: "pred-1",
    channelId: 12440103,
    title: "mac bitis suresi",
    durationSeconds: 60,
    state: "RESOLVED",
    winningOutcomeId: "out-1",
    createdAt: "2026-05-17T05:09:11Z",
    lockedAt: "2026-05-17T05:10:12Z",
    updatedAt: "2026-05-17T05:13:10Z",
    totalPoints: 100,
    totalVotes: 4,
    outcomes: [
      {
        id: "out-1",
        title: "Evet",
        totalVoteAmount: 75,
        voteCount: 3,
        returnRate: 1.33,
        pointShare: 0.75,
        isWinner: true,
        topUsers: [{ id: 1, username: "alice", amount: 50 }]
      },
      {
        id: "out-2",
        title: "Hayir",
        totalVoteAmount: 25,
        voteCount: 1,
        returnRate: 4.0,
        pointShare: 0.25,
        isWinner: false,
        topUsers: [{ id: 2, username: "bob", amount: 25 }]
      }
    ],
    ...overrides
  };
}

async function flushPromises() {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}
