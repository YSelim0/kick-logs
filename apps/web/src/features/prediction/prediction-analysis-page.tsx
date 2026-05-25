"use client";

import { ArrowLeft, RefreshCw } from "lucide-react";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";

import { SiteHeader } from "@/components/site-header";
import { Button } from "@/components/ui/button";
import { getPrediction } from "@/features/prediction/api";
import { PredictionDistributionChart } from "@/features/prediction/prediction-distribution-chart";
import { PredictionTopUsersChart } from "@/features/prediction/prediction-top-users-chart";
import { PredictionVoteReturnChart } from "@/features/prediction/prediction-vote-return-chart";
import {
  formatCompactNumber,
  formatDateTime,
  formatMultiplier,
  formatPercent,
  outcomeColor,
  predictionStateBadge,
  type PredictionStateBadge
} from "@/features/prediction/format";
import { ApiClientError } from "@/lib/api-client";
import type { Prediction, PredictionOutcome } from "@/types/api";

type Status = "loading" | "ready" | "not-found" | "error";

export function PredictionAnalysisPage({ slug }: { slug: string }) {
  const [prediction, setPrediction] = useState<Prediction | null>(null);
  const [status, setStatus] = useState<Status>("loading");

  const load = useCallback(async () => {
    setStatus("loading");
    try {
      const next = await getPrediction(slug);
      setPrediction(next);
      setStatus("ready");
    } catch (caught) {
      setPrediction(null);
      setStatus(caught instanceof ApiClientError && caught.status === 404 ? "not-found" : "error");
    }
  }, [slug]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <main className="min-h-screen bg-page text-foreground">
      <SiteHeader activeRoute="prediction" />

      <div className="mx-auto max-w-[1280px] px-6 py-6">
        <header className="mb-5 flex items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <Link
              aria-label="Geri"
              className="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-elevated hover:text-foreground"
              href="/prediction"
            >
              <ArrowLeft className="h-4 w-4" />
            </Link>
            <p className="font-mono text-[13px]">
              <span className="text-accent">kick.com/</span>
              <span className="text-foreground">{slug}</span>
            </p>
          </div>
          <button
            aria-label="Yenile"
            className="inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-elevated hover:text-foreground disabled:opacity-50"
            disabled={status === "loading"}
            onClick={() => void load()}
            type="button"
          >
            <RefreshCw className={`h-4 w-4 ${status === "loading" ? "animate-spin" : ""}`} />
          </button>
        </header>

        {status === "loading" ? <LoadingState /> : null}
        {status === "not-found" ? <NotFoundState /> : null}
        {status === "error" ? <ErrorState onRetry={() => void load()} /> : null}
        {status === "ready" && prediction ? <PredictionContent prediction={prediction} /> : null}
      </div>
    </main>
  );
}

function LoadingState() {
  return (
    <div className="flex items-center gap-2 py-16 text-[13px] text-muted-foreground">
      <span aria-hidden className="inline-block h-1.5 w-1.5 animate-pulse rounded-full bg-accent" />
      <span className="font-mono">tahmin verisi yükleniyor…</span>
    </div>
  );
}

function NotFoundState() {
  return (
    <section className="rounded-lg border border-border bg-panel p-6">
      <p className="text-sm font-medium text-warning">Bu kanal için aktif tahmin bulunamadı.</p>
      <p className="mt-1 text-xs text-muted-foreground">
        Kanalın güncel bir tahmin oyunu olmayabilir.
      </p>
      <Button asChild className="mt-4" size="sm">
        <Link href="/prediction">Başka kanal dene</Link>
      </Button>
    </section>
  );
}

function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <section className="rounded-lg border border-danger/40 bg-danger/10 p-6">
      <p className="text-sm font-medium text-danger">Tahmin verisi alınamadı.</p>
      <p className="mt-1 text-xs text-muted-foreground">
        Kanal adını kontrol edin veya birazdan tekrar deneyin.
      </p>
      <Button className="mt-4" onClick={onRetry} size="sm" variant="outline">
        Tekrar dene
      </Button>
    </section>
  );
}

function PredictionContent({ prediction }: { prediction: Prediction }) {
  return (
    <div className="space-y-5">
      <SummaryCard prediction={prediction} />

      <section className="grid grid-cols-1 gap-5 md:grid-cols-2">
        {prediction.outcomes.map((outcome, index) => (
          <OutcomeCard color={outcomeColor(index)} key={outcome.id} outcome={outcome} />
        ))}
      </section>

      <section className="grid grid-cols-1 gap-5 lg:grid-cols-2">
        <Panel subtitle="puan payı" title="Puan dağılımı">
          <PredictionDistributionChart outcomes={prediction.outcomes} />
        </Panel>
        <Panel subtitle="oy / getiri" title="Oy ve getiri">
          <PredictionVoteReturnChart outcomes={prediction.outcomes} />
        </Panel>
      </section>

      <Panel subtitle="en yüksek bahisler" title="Top kullanıcılar">
        <PredictionTopUsersChart outcomes={prediction.outcomes} />
      </Panel>
    </div>
  );
}

function SummaryCard({ prediction }: { prediction: Prediction }) {
  const badge = predictionStateBadge(prediction.state);

  return (
    <section className="rounded-lg border border-border bg-panel px-6 py-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <h1 className="text-[22px] font-semibold leading-tight text-foreground">
          {prediction.title}
        </h1>
        <StatePill badge={badge} />
      </div>

      <div className="mt-5 grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border md:grid-cols-4">
        <SummaryCell label="TOPLAM PUAN" value={formatCompactNumber(prediction.totalPoints)} />
        <SummaryCell label="TOPLAM OY" value={formatCompactNumber(prediction.totalVotes)} />
        <SummaryCell label="SÜRE" value={`${prediction.durationSeconds}s`} />
        <SummaryCell label="OLUŞTURULMA" value={formatDateTime(prediction.createdAt)} />
      </div>

      {prediction.lockedAt ? (
        <p className="mt-3 font-mono text-[11px] text-muted-foreground">
          kilitlenme:{" "}
          <span className="text-muted-foreground">{formatDateTime(prediction.lockedAt)}</span>
        </p>
      ) : null}
    </section>
  );
}

function StatePill({ badge }: { badge: PredictionStateBadge }) {
  const toneClass =
    badge.tone === "accent"
      ? "bg-elevated text-accent"
      : badge.tone === "warning"
        ? "bg-elevated text-warning"
        : "bg-elevated text-muted-foreground";

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 font-mono text-[10px] font-semibold uppercase tracking-wider ${toneClass}`}
    >
      {badge.label}
    </span>
  );
}

function SummaryCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-panel px-5 py-4">
      <div className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="mt-2 truncate font-sans text-[20px] font-semibold leading-none tracking-tight text-foreground">
        {value}
      </div>
    </div>
  );
}

function Panel({
  title,
  subtitle,
  children
}: {
  title: string;
  subtitle: string;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col rounded-lg border border-border bg-panel p-5">
      <header className="mb-4 flex flex-col gap-0.5">
        <h2 className="text-[14px] font-semibold leading-none text-foreground">{title}</h2>
        <p className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
          {subtitle}
        </p>
      </header>
      <div className="flex-1">{children}</div>
    </section>
  );
}

function OutcomeCard({ outcome, color }: { outcome: PredictionOutcome; color: string }) {
  return (
    <article
      className={`rounded-lg border bg-panel p-4 ${
        outcome.isWinner ? "border-accent" : "border-border"
      }`}
    >
      <header className="mb-3 flex items-center gap-2">
        <span
          aria-hidden
          className="inline-block h-2.5 w-2.5 shrink-0 rounded-sm"
          style={{ backgroundColor: color }}
        />
        <h3 className="flex-1 truncate text-[14px] font-medium text-foreground">{outcome.title}</h3>
        {outcome.isWinner ? (
          <span className="rounded-full bg-accent px-2 py-0.5 font-mono text-[9px] font-semibold uppercase tracking-wider text-text-on-accent">
            KAZANAN
          </span>
        ) : null}
      </header>

      <div className="grid grid-cols-3 gap-2">
        <OutcomeStat
          label="PUAN"
          sub={formatPercent(outcome.pointShare)}
          value={formatCompactNumber(outcome.totalVoteAmount)}
        />
        <OutcomeStat label="OY" value={formatCompactNumber(outcome.voteCount)} />
        <OutcomeStat label="GETİRİ" value={formatMultiplier(outcome.returnRate)} />
      </div>

      {outcome.topUsers.length > 0 ? (
        <ol className="mt-3 flex flex-col gap-1.5 border-t border-border pt-3">
          {outcome.topUsers.map((user, index) => (
            <li className="flex items-center gap-2 text-[13px]" key={user.id}>
              <span className="w-4 shrink-0 font-mono text-[10px] text-faint">{index + 1}</span>
              <span className="flex-1 truncate text-foreground">{user.username}</span>
              <span className="shrink-0 font-mono text-accent">
                {formatCompactNumber(user.amount)}
              </span>
            </li>
          ))}
        </ol>
      ) : null}
    </article>
  );
}

function OutcomeStat({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div>
      <div className="font-mono text-[9px] uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="mt-1 font-mono text-[13px] font-semibold text-foreground">{value}</div>
      {sub ? <div className="font-mono text-[10px] text-muted-foreground">{sub}</div> : null}
    </div>
  );
}
