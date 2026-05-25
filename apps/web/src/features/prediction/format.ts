import type { PredictionOutcome } from "@/types/api";

const COMPACT_FORMATTER = new Intl.NumberFormat("tr-TR", {
  notation: "compact",
  maximumFractionDigits: 1
});

const DATE_TIME_FORMATTER = new Intl.DateTimeFormat("tr-TR", {
  day: "2-digit",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit"
});

export function formatCompactNumber(value: number): string {
  return COMPACT_FORMATTER.format(value);
}

export function formatPercent(share: number): string {
  return `%${(share * 100).toFixed(1)}`;
}

export function formatMultiplier(returnRate: number): string {
  return `${returnRate.toFixed(2)}x`;
}

export function formatDateTime(value: string | null): string {
  if (!value) return "—";
  return DATE_TIME_FORMATTER.format(new Date(value));
}

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
  label: string;
  tone: PredictionStateTone;
};

export function predictionStateBadge(state: string): PredictionStateBadge {
  switch (state.toUpperCase()) {
    case "RESOLVED":
      return { label: "Sonuçlandı", tone: "accent" };
    case "LOCKED":
      return { label: "Kilitli", tone: "warning" };
    case "ACTIVE":
      return { label: "Aktif", tone: "neutral" };
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
