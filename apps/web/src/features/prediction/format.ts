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

// Categorical chart palette rooted in the v2 design tokens (the base palette has no blue/purple).
// accent, warning, text-secondary, danger, border-strong, accent-hover.
export const CHART_COLORS = [
  "#00e701",
  "#facc15",
  "#9ca3af",
  "#ff4d4f",
  "#474f54",
  "#00c701"
] as const;

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
