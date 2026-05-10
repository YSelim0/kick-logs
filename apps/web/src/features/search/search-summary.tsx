"use client";

import { Filter, Radio, Search, Timer } from "lucide-react";
import type { ReactNode } from "react";

import {
  getActiveFilters,
  getLastMatchTime,
  getScopeText,
  type SearchFormState
} from "@/features/search/search-params";
import type { Message } from "@/types/api";

type SearchSummaryProps = {
  state: SearchFormState;
  messages: Message[];
  isLoading: boolean;
  error: string | null;
};

export function SearchSummary({ state, messages, isLoading, error }: SearchSummaryProps) {
  const activeFilters = getActiveFilters(state);

  return (
    <aside className="rounded-lg border border-border bg-black p-5">
      <div className="mb-4 flex items-center justify-between border-b border-border pb-4">
        <div className="flex items-center gap-2">
          <Search className="h-4 w-4 text-primary" />
          <h2 className="text-base font-semibold">Arama Özeti</h2>
        </div>
        <div className="flex items-center gap-2 rounded-md bg-kick-background px-3 py-1.5 text-xs text-muted-foreground">
          <span className="h-2 w-2 rounded-full bg-primary" />
          {error ? "Hata" : isLoading ? "Yükleniyor" : "Hazır"}
        </div>
      </div>

      <div className="space-y-3 text-sm">
        <SummaryLine icon={<Radio className="h-4 w-4" />} label="Kapsam" value={getScopeText(state)} />
        <SummaryLine icon={<Filter className="h-4 w-4" />} label="Yüklenen mesaj" value={String(messages.length)} />
        <SummaryLine icon={<Timer className="h-4 w-4" />} label="Son eşleşme" value={getLastMatchTime(messages)} />
      </div>

      <div className="mt-5 border-t border-border pt-4">
        <div className="mb-3 flex items-center gap-2 text-sm">
          <Filter className="h-4 w-4 text-primary" />
          Aktif filtreler
        </div>
        <div className="flex flex-wrap gap-2">
          {activeFilters.length ? (
            activeFilters.map((filter) => (
              <span
                className="rounded-md bg-kick-background px-2.5 py-1 text-xs text-foreground"
                key={filter.key}
              >
                <span className="text-primary">{filter.label}:</span> {filter.value}
              </span>
            ))
          ) : (
            <span className="rounded-md bg-kick-background px-2.5 py-1 text-xs text-muted-foreground">
              Filtre yok
            </span>
          )}
        </div>
      </div>
    </aside>
  );
}

function SummaryLine({
  icon,
  label,
  value
}: {
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="flex items-center gap-2 text-muted-foreground">
        <span className="text-accent">{icon}</span>
        {label}
      </div>
      <div className="truncate text-primary">{value}</div>
    </div>
  );
}
