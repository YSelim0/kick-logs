import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithLocale } from "@/test/render-with-locale";
import { PredictionDistributionChart } from "./prediction-distribution-chart";
import { PredictionVoteReturnChart } from "./prediction-vote-return-chart";
import { PredictionTopUsersChart } from "./prediction-top-users-chart";

describe("localized prediction chart empty states", () => {
  it("renders German copy without depending on source outcome labels", () => {
    renderWithLocale(
      <>
        <PredictionDistributionChart outcomes={[]} />
        <PredictionVoteReturnChart outcomes={[]} />
        <PredictionTopUsersChart outcomes={[]} />
      </>,
      { locale: "de" }
    );
    expect(screen.getByText("Keine Daten zur Punkteverteilung.")).toBeInTheDocument();
    expect(screen.getByText("Keine Stimmen- oder Quotendaten.")).toBeInTheDocument();
    expect(screen.getByText("Keine Daten zu Top-Nutzern.")).toBeInTheDocument();
  });
});
