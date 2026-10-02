"use client";

import { useTranslations } from "next-intl";
import { useUiFormat } from "@/i18n/use-ui-format";

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

import { RETURN_RATE_COLOR, VOTE_COUNT_COLOR } from "@/features/prediction/format";
import type { PredictionOutcome } from "@/types/api";

const CHART_INITIAL_DIMENSION = { width: 640, height: 224 };

export function PredictionVoteReturnChart({ outcomes }: { outcomes: PredictionOutcome[] }) {
  const t = useTranslations("prediction");
  const f = useUiFormat();
  const data = outcomes.map((outcome) => ({
    name: outcome.title,
    voteCount: outcome.voteCount,
    returnRate: outcome.returnRate
  }));

  if (data.length === 0) {
    return <p className="text-[13px] text-muted-foreground">{t("emptyVotes")}</p>;
  }

  return (
    <div className="h-56 min-w-0 w-full">
      <ResponsiveContainer
        height="100%"
        initialDimension={CHART_INITIAL_DIMENSION}
        minWidth={0}
        width="100%"
      >
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
            tickFormatter={(value) => f.compact(Number(value))}
            tick={{ fill: "#9ca3af", fontSize: 11 }}
            tickLine={false}
            width={48}
          />
          <YAxis
            yAxisId="return"
            axisLine={false}
            orientation="right"
            tickFormatter={(value) => t("multiplier", { value: f.number(Number(value), 2) })}
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
            formatter={(value, name, item) =>
              item.dataKey === "returnRate"
                ? [t("multiplier", { value: f.number(Number(value), 2) }), name]
                : [f.compact(Number(value)), name]
            }
            itemStyle={{ color: "#ffffff" }}
            labelStyle={{ color: "#9ca3af" }}
          />
          <Legend wrapperStyle={{ fontSize: 12, color: "#9ca3af" }} />
          <Bar
            yAxisId="votes"
            dataKey="voteCount"
            fill={VOTE_COUNT_COLOR}
            name={t("voteCount")}
            radius={[3, 3, 0, 0]}
          />
          <Bar
            yAxisId="return"
            dataKey="returnRate"
            fill={RETURN_RATE_COLOR}
            name={t("returnRate")}
            radius={[3, 3, 0, 0]}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
