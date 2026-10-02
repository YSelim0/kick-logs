"use client";

import { useTranslations } from "next-intl";
import { useUiFormat } from "@/i18n/use-ui-format";

import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";

import { outcomeColor } from "@/features/prediction/format";
import type { PredictionOutcome } from "@/types/api";

const CHART_INITIAL_DIMENSION = { width: 640, height: 224 };

export function PredictionDistributionChart({ outcomes }: { outcomes: PredictionOutcome[] }) {
  const t = useTranslations("prediction");
  const f = useUiFormat();
  const data = outcomes.map((outcome, index) => ({
    name: outcome.title,
    value: outcome.totalVoteAmount,
    share: outcome.pointShare,
    color: outcomeColor(index)
  }));

  const hasData = data.some((entry) => entry.value > 0);
  if (!hasData) {
    return <p className="text-[13px] text-muted-foreground">{t("emptyDistribution")}</p>;
  }

  return (
    <div className="min-w-0 w-full">
      <div className="h-56 min-w-0 w-full">
        <ResponsiveContainer
          height="100%"
          initialDimension={CHART_INITIAL_DIMENSION}
          minWidth={0}
          width="100%"
        >
          <PieChart>
            <Pie
              cx="50%"
              cy="50%"
              data={data}
              dataKey="value"
              innerRadius={50}
              nameKey="name"
              outerRadius={80}
              paddingAngle={2}
              stroke="#0b0e0f"
            >
              {data.map((entry) => (
                <Cell fill={entry.color} key={entry.name} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: "#24272c",
                border: "1px solid #474f54",
                borderRadius: 6,
                fontSize: 12
              }}
              formatter={(value, _name, item) => {
                const payload = (item as { payload?: { share: number; name: string } }).payload;
                return [
                  t("distributionTooltip", {
                    points: f.compact(Number(value)),
                    share: f.percent(payload?.share ?? 0, 1)
                  }),
                  payload?.name ?? ""
                ];
              }}
              itemStyle={{ color: "#ffffff" }}
              labelStyle={{ color: "#9ca3af" }}
            />
          </PieChart>
        </ResponsiveContainer>
      </div>

      <ul className="mt-3 flex flex-wrap gap-x-4 gap-y-1">
        {data.map((entry) => (
          <li
            className="flex items-center gap-1.5 text-[12px] text-muted-foreground"
            key={entry.name}
          >
            <span
              aria-hidden
              className="inline-block h-2.5 w-2.5 rounded-sm"
              style={{ backgroundColor: entry.color }}
            />
            <span className="truncate text-foreground">{entry.name}</span>
            <span className="font-mono">{f.percent(entry.share, 1)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
