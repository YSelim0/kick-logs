"use client";

import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis
} from "recharts";

import {
  formatCompactNumber,
  formatMultiplier,
  RETURN_RATE_COLOR,
  VOTE_COUNT_COLOR
} from "@/features/prediction/format";
import type { PredictionOutcome } from "@/types/api";

export function PredictionVoteReturnChart({ outcomes }: { outcomes: PredictionOutcome[] }) {
  const data = outcomes.map((outcome) => ({
    name: outcome.title,
    voteCount: outcome.voteCount,
    returnRate: outcome.returnRate
  }));

  if (data.length === 0) {
    return <p className="text-[13px] text-muted-foreground">Oy ve oran verisi yok.</p>;
  }

  return (
    <div className="h-56 w-full">
      <ResponsiveContainer height="100%" width="100%">
        <BarChart data={data} margin={{ top: 8, right: 0, bottom: 0, left: -16 }}>
          <CartesianGrid stroke="#24272c" vertical={false} />
          <XAxis
            axisLine={{ stroke: "#24272c" }}
            dataKey="name"
            tick={{ fill: "#9ca3af", fontSize: 11 }}
            tickLine={false}
          />
          <YAxis
            yAxisId="votes"
            axisLine={false}
            tickFormatter={(value) => formatCompactNumber(Number(value))}
            tick={{ fill: "#9ca3af", fontSize: 11 }}
            tickLine={false}
            width={48}
          />
          <YAxis
            yAxisId="return"
            axisLine={false}
            orientation="right"
            tickFormatter={(value) => formatMultiplier(Number(value))}
            tick={{ fill: "#9ca3af", fontSize: 11 }}
            tickLine={false}
            width={48}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: "#24272c",
              border: "1px solid #474f54",
              borderRadius: 6,
              fontSize: 12
            }}
            cursor={{ fill: "#24272c", opacity: 0.4 }}
            formatter={(value, name) =>
              name === "Getiri oranı"
                ? [formatMultiplier(Number(value)), name]
                : [formatCompactNumber(Number(value)), name]
            }
            itemStyle={{ color: "#ffffff" }}
            labelStyle={{ color: "#9ca3af" }}
          />
          <Legend wrapperStyle={{ fontSize: 12, color: "#9ca3af" }} />
          <Bar
            yAxisId="votes"
            dataKey="voteCount"
            fill={VOTE_COUNT_COLOR}
            name="Oy sayısı"
            radius={[3, 3, 0, 0]}
          />
          <Bar
            yAxisId="return"
            dataKey="returnRate"
            fill={RETURN_RATE_COLOR}
            name="Getiri oranı"
            radius={[3, 3, 0, 0]}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
