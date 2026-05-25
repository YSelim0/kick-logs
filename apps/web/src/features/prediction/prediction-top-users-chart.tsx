"use client";

import { Bar, BarChart, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

import { flattenTopUsers, formatCompactNumber, outcomeColor } from "@/features/prediction/format";
import type { PredictionOutcome } from "@/types/api";

const MAX_BARS = 12;

export function PredictionTopUsersChart({ outcomes }: { outcomes: PredictionOutcome[] }) {
  const bars = flattenTopUsers(outcomes).slice(0, MAX_BARS);

  if (bars.length === 0) {
    return <p className="text-[13px] text-muted-foreground">Üst kullanıcı verisi yok.</p>;
  }

  const height = Math.max(bars.length * 28, 120);

  return (
    <div className="w-full">
      <div className="w-full" style={{ height }}>
        <ResponsiveContainer height="100%" width="100%">
          <BarChart
            data={bars}
            layout="vertical"
            margin={{ top: 0, right: 16, bottom: 0, left: 8 }}
          >
            <XAxis hide type="number" />
            <YAxis
              axisLine={false}
              dataKey="username"
              tick={{ fill: "#9ca3af", fontSize: 11 }}
              tickLine={false}
              type="category"
              width={96}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: "#24272c",
                border: "1px solid #474f54",
                borderRadius: 6,
                fontSize: 12
              }}
              cursor={{ fill: "#24272c", opacity: 0.4 }}
              formatter={(value, _name, item) => {
                const payload = (item as { payload?: { outcomeTitle: string } }).payload;
                return [`${formatCompactNumber(Number(value))} puan`, payload?.outcomeTitle ?? ""];
              }}
              itemStyle={{ color: "#ffffff" }}
              labelStyle={{ color: "#9ca3af" }}
            />
            <Bar dataKey="amount" radius={[0, 3, 3, 0]}>
              {bars.map((bar, index) => (
                <Cell fill={outcomeColor(bar.outcomeIndex)} key={`${bar.username}-${index}`} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>

      <ul className="mt-3 flex flex-wrap gap-x-4 gap-y-1">
        {outcomes.map((outcome, index) => (
          <li
            className="flex items-center gap-1.5 text-[12px] text-muted-foreground"
            key={outcome.id}
          >
            <span
              aria-hidden
              className="inline-block h-2.5 w-2.5 rounded-sm"
              style={{ backgroundColor: outcomeColor(index) }}
            />
            <span className="truncate text-foreground">{outcome.title}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
