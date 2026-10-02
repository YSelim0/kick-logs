import type { PredictionOutcome } from "@/types/api";

// Categorical chart palette rooted in the app palette. The first two colors are intentionally
// high-contrast for two-outcome predictions.
export const CHART_COLORS = [
  "#22C55E",
  "#C084FC",
  "#FFFFFF",
  "#FF005C",
  "#474f54",
  "#26001B"
] as const;

export const VOTE_COUNT_COLOR = CHART_COLORS[0];
export const RETURN_RATE_COLOR = CHART_COLORS[1];

export function outcomeColor(index: number): string {
  return CHART_COLORS[index % CHART_COLORS.length];
}

type PredictionStateTone = "accent" | "warning" | "neutral";

export type PredictionStateBadge = {
  key?: "active" | "locked" | "resolved" | "cancelled";
  label?: string;
  tone: PredictionStateTone;
};

export function predictionStateBadge(state: string): PredictionStateBadge {
  switch (state.toUpperCase()) {
    case "RESOLVED":
      return { key: "resolved", tone: "accent" };
    case "LOCKED":
      return { key: "locked", tone: "warning" };
    case "CANCELED":
    case "CANCELLED":
      return { key: "cancelled", tone: "warning" };
    case "ACTIVE":
      return { key: "active", tone: "neutral" };
    default:
      return { label: state || "—", tone: "neutral" };
  }
}

export type TopUserBar = {
  username: string;
  amount: number;
  outcomeIndex: number;
  outcomeTitle: string;
};

// Flatten the per-outcome top users into a single ranked list for the horizontal bar chart.
export function flattenTopUsers(outcomes: PredictionOutcome[]): TopUserBar[] {
  const bars: TopUserBar[] = [];
  outcomes.forEach((outcome, outcomeIndex) => {
    outcome.topUsers.forEach((user) => {
      bars.push({
        username: user.username,
        amount: user.amount,
        outcomeIndex,
        outcomeTitle: outcome.title
      });
    });
  });
  return bars.sort((a, b) => b.amount - a.amount);
}
