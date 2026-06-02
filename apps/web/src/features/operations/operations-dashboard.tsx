"use client";

import { useCallback, useEffect, useState } from "react";
import type { ReactNode } from "react";
import {
  Database,
  HardDrive,
  Loader2,
  MessageSquareText,
  RefreshCcw,
  TriangleAlert
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { getOperationsSummary } from "@/features/operations/api";
import { FailedEventsModal } from "@/features/operations/failed-events-modal";
import type { IngestionHealth, OperationsSummary } from "@/types/api";

const EMPTY_INGESTION: IngestionHealth = {
  queue_depth: 0,
  oldest_pending_age_seconds: 0,
  legacy_queue_depth: 0,
  legacy_oldest_pending_age_seconds: 0,
  stream_messages: 0,
  stream_bytes: 0,
  stream_consumer_pending: 0,
  stream_consumer_ack_pending: 0,
  stream_consumer_redelivered: 0,
  stream_oldest_pending_age_seconds: 0,
  stream_latest_message_age_seconds: 0,
  stream_latest_consumer_update_time: null,
  stream_error: "",
  write_queue_depth: 0,
  write_queue_high_water_mark: 0,
  write_drop_count: 0,
  write_flush_count: 0,
  last_flush_size: 0,
  last_flush_millis: 0,
  clickhouse_insert_failures: 0,
  queue_enqueue_failures: 0,
  breaker_state: "closed",
  breaker_current_delay_ms: 0
};

export function OperationsDashboard() {
  const [summary, setSummary] = useState<OperationsSummary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [failedModalOpen, setFailedModalOpen] = useState(false);

  const loadSummary = useCallback(async (mode: "initial" | "refresh" = "initial") => {
    if (mode === "refresh") {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }
    setError(null);

    try {
      setSummary(await getOperationsSummary());
    } catch (caught) {
      setError(resolveOperationsError(caught));
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  }, []);

  useEffect(() => {
    void loadSummary("initial");
  }, [loadSummary]);

  const ingestion = summary?.ingestion ?? EMPTY_INGESTION;
  const failedRawEvents = summary ? getStatusCount(summary, "failed") : 0;
  const isBreakerOpen = ingestion.breaker_state === "open";

  return (
    <section className="rounded-lg border border-border bg-panel p-5">
      <div className="mb-5 flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-col gap-1">
          <h2 className="text-[22px] font-semibold tracking-tight text-foreground">Operations</h2>
          <p className="font-sans text-[13px] text-muted-foreground">
            Listener, processor, JetStream backlog, ClickHouse geçmişi, depolama özeti
          </p>
        </div>
        <Button
          disabled={isLoading || isRefreshing}
          onClick={() => void loadSummary("refresh")}
          size="sm"
          type="button"
          variant="outline"
        >
          {isRefreshing ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <RefreshCcw className="h-3 w-3" />
          )}
          Yenile
        </Button>
      </div>

      {isLoading && !summary ? (
        <div className="rounded-md border border-border bg-elevated px-4 py-8 text-center text-[13px] text-muted-foreground">
          Operasyon metrikleri yükleniyor...
        </div>
      ) : null}

      {error ? (
        <div className="mb-4 rounded-md border border-danger bg-elevated px-3 py-2 text-[13px]">
          {error}
        </div>
      ) : null}

      {summary ? (
        <div className="flex flex-col gap-4">
          {!summary.listener.is_fresh ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message="Listener heartbeat bayat. Listener çalışmıyor olabilir veya DB'ye yazamıyor olabilir."
              tone="warning"
            />
          ) : null}
          {!summary.processor.is_fresh ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message="Processor heartbeat bayat. JetStream backlog ClickHouse'a yazılamıyor olabilir."
              tone="warning"
            />
          ) : null}
          {failedRawEvents > 0 ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message="ClickHouse failed raw event attempt kaydı var. JetStream redelivery otomatik; backend loglarını incelemek gerekebilir."
              tone="danger"
            />
          ) : null}
          {isBreakerOpen ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message={`ClickHouse circuit breaker açık. Bekleme ${Math.round(ingestion.breaker_current_delay_ms)} ms.`}
              tone="danger"
            />
          ) : null}
          {ingestion.write_drop_count > 0 ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message={`Legacy buffered writer ${formatNumber(ingestion.write_drop_count)} event düşürdü. Bu metrik yeni JetStream hot path'te sıfır kalmalı.`}
              tone="warning"
            />
          ) : null}
          {ingestion.stream_consumer_redelivered > 0 ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message={`JetStream ${formatNumber(ingestion.stream_consumer_redelivered)} redelivery bildiriyor. Processor yavaşlaması veya geçici ClickHouse hatası olabilir.`}
              tone="warning"
            />
          ) : null}
          {ingestion.stream_error ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message={`JetStream metrikleri okunamadı: ${ingestion.stream_error}`}
              tone="warning"
            />
          ) : null}

          <div className="flex flex-wrap items-center gap-3 rounded-lg border border-border bg-panel px-4 py-3">
            <div className="grid flex-1 gap-2 md:grid-cols-2">
              <HeartbeatSummary label="Listener" heartbeat={summary.listener} />
              <HeartbeatSummary label="Processor" heartbeat={summary.processor} />
            </div>
            <span className="shrink-0 font-mono text-[11px] text-faint">
              {new Date().toISOString().slice(11, 19)} UTC
            </span>
          </div>

          <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
            <MetricCard
              detail={`${formatNumber(summary.counts.senders)} gönderici`}
              icon={<MessageSquareText className="h-3.5 w-3.5" />}
              iconTone="accent"
              label="MESAJ"
              value={formatNumber(summary.counts.messages)}
            />
            <MetricCard
              detail={`ack bekleyen ${formatNumber(ingestion.stream_consumer_ack_pending)}`}
              icon={<HardDrive className="h-3.5 w-3.5" />}
              iconTone="muted"
              label="JETSTREAM BACKLOG"
              value={formatNumber(ingestion.queue_depth)}
            />
            <MetricCard
              detail={failedRawEvents > 0 ? "İnceleme gerekli" : "Temiz"}
              detailTone={failedRawEvents > 0 ? "danger" : "muted"}
              icon={<TriangleAlert className="h-3.5 w-3.5" />}
              iconTone={failedRawEvents > 0 ? "danger" : "muted"}
              label="BAŞARISIZ RAW"
              onDetailClick={failedRawEvents > 0 ? () => setFailedModalOpen(true) : undefined}
              value={formatNumber(failedRawEvents)}
            />
            <MetricCard
              detail={`${summary.storage.tables.length} tablo`}
              icon={<Database className="h-3.5 w-3.5" />}
              iconTone="muted"
              label="DB BOYUTU"
              value={formatBytes(summary.storage.database_bytes)}
            />
          </div>

          <div className="rounded-lg border border-border bg-panel p-5">
            <div className="mb-4 flex items-center justify-between">
              <div className="flex flex-col gap-0.5">
                <span className="text-[14px] font-semibold text-foreground">Aktif Ingestion</span>
                <span className="font-mono text-[11px] text-faint">
                  JetStream consumer, processor breaker, legacy SQLite queue
                </span>
              </div>
              <div
                className={`flex items-center gap-1.5 rounded-full bg-elevated px-2.5 py-1 font-mono text-[10px] font-semibold tracking-[0.8px] ${
                  isBreakerOpen ? "text-danger" : "text-accent"
                }`}
              >
                <span
                  className={`h-1.5 w-1.5 rounded-full ${isBreakerOpen ? "bg-danger" : "bg-accent"}`}
                />
                {isBreakerOpen ? "Açık" : "Kapalı"}
              </div>
            </div>
            <div className="overflow-x-auto rounded-md border border-border">
              <div className="flex min-w-[720px] divide-x divide-border">
                <IngestionCell
                  label="Stream pending"
                  value={formatNumber(ingestion.stream_consumer_pending)}
                />
                <IngestionCell
                  label="Ack pending"
                  value={formatNumber(ingestion.stream_consumer_ack_pending)}
                />
                <IngestionCell
                  label="Redelivery"
                  value={formatNumber(ingestion.stream_consumer_redelivered)}
                />
                <IngestionCell
                  label="En eski"
                  value={
                    ingestion.stream_oldest_pending_age_seconds > 0
                      ? `${formatNumber(ingestion.stream_oldest_pending_age_seconds)}s`
                      : "—"
                  }
                />
                <IngestionCell
                  label="Legacy SQLite"
                  value={formatNumber(ingestion.legacy_queue_depth)}
                />
                <IngestionCell
                  label="CH failures"
                  value={formatNumber(ingestion.clickhouse_insert_failures)}
                />
              </div>
            </div>
          </div>
        </div>
      ) : null}

      <FailedEventsModal
        open={failedModalOpen}
        onOpenChange={setFailedModalOpen}
        onActionComplete={() => void loadSummary("refresh")}
      />
    </section>
  );
}

function HeartbeatSummary({
  heartbeat,
  label
}: {
  heartbeat: OperationsSummary["listener"];
  label: string;
}) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <span
        className={`h-2 w-2 shrink-0 rounded-full ${heartbeat.is_fresh ? "bg-accent" : "bg-warning"}`}
      />
      <span className="min-w-0 truncate text-[13px] font-medium text-foreground">
        {label}: {heartbeat.is_fresh ? "Canlı" : "Bayat"}
        {" · son sinyal "}
        {heartbeat.seconds_since_last_seen !== null
          ? `${heartbeat.seconds_since_last_seen}s`
          : "yok"}
        {" önce"}
      </span>
    </div>
  );
}

function MetricCard({
  detail,
  detailTone = "muted",
  icon,
  iconTone = "muted",
  label,
  onDetailClick,
  value
}: {
  detail: string;
  detailTone?: "muted" | "danger";
  icon: ReactNode;
  iconTone?: "accent" | "muted" | "danger";
  label: string;
  onDetailClick?: () => void;
  value: string;
}) {
  const iconClass =
    iconTone === "accent"
      ? "text-accent"
      : iconTone === "danger"
        ? "text-danger"
        : "text-muted-foreground";
  const detailClass = detailTone === "danger" ? "text-danger" : "text-muted-foreground";

  return (
    <div className="flex flex-col gap-2.5 rounded-lg border border-border bg-panel p-4">
      <div className="flex items-center justify-between">
        <span className="font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
          {label}
        </span>
        <span className={iconClass}>{icon}</span>
      </div>
      <span className="text-[26px] font-semibold leading-none tracking-tight text-foreground">
        {value}
      </span>
      {onDetailClick ? (
        <button
          className={`text-left text-[11px] underline-offset-2 hover:underline ${detailClass}`}
          onClick={onDetailClick}
          type="button"
        >
          {detail}
        </button>
      ) : (
        <span className={`text-[11px] ${detailClass}`}>{detail}</span>
      )}
    </div>
  );
}

function IngestionCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-1 flex-col gap-1 bg-panel px-3.5 py-3">
      <span className="font-mono text-[10px] tracking-[0.5px] text-faint">{label}</span>
      <span className="text-[18px] font-semibold text-foreground">{value}</span>
    </div>
  );
}

function OperationsNotice({
  icon,
  message,
  tone
}: {
  icon: ReactNode;
  message: string;
  tone: "warning" | "danger";
}) {
  return (
    <div
      className={`flex items-center gap-2 rounded-md border bg-elevated px-3 py-2 text-[13px] ${
        tone === "danger" ? "border-danger text-danger" : "border-warning text-warning"
      }`}
    >
      {icon}
      <span className="text-foreground">{message}</span>
    </div>
  );
}

function getStatusCount(summary: OperationsSummary, status: string) {
  return summary.raw_event_status_counts[status] ?? 0;
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("tr-TR").format(value);
}

function formatBytes(value: number) {
  if (value <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const amount = value / 1024 ** index;
  return `${new Intl.NumberFormat("tr-TR", { maximumFractionDigits: index === 0 ? 0 : 2 }).format(amount)} ${units[index]}`;
}

function resolveOperationsError(error: unknown) {
  if (error instanceof Error) return error.message;
  return "Operasyon metrikleri alınamadı.";
}
