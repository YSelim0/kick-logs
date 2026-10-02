"use client";

import { useTranslations } from "next-intl";
import { useUiFormat } from "@/i18n/use-ui-format";
import { getUiErrorKey, type UiErrorKey } from "@/i18n/errors";

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
  captured_raw_events: 0,
  recent_message_poll_captured: 0,
  recent_message_poll_errors: 0,
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
  const t = useTranslations("operations");
  const f = useUiFormat();
  const errors = useTranslations("common.errors");
  const [summary, setSummary] = useState<OperationsSummary | null>(null);
  const [error, setError] = useState<UiErrorKey | null>(null);
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
      setError(getUiErrorKey(caught, "adminRead"));
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
          <h2 className="text-[22px] font-semibold tracking-tight text-foreground">{t("title")}</h2>
          <p className="font-sans text-[13px] text-muted-foreground">{t("description")}</p>
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
          {t("refresh")}
        </Button>
      </div>

      {isLoading && !summary ? (
        <div className="rounded-md border border-border bg-elevated px-4 py-8 text-center text-[13px] text-muted-foreground">
          {t("loading")}
        </div>
      ) : null}

      {error ? (
        <div className="mb-4 rounded-md border border-danger bg-elevated px-3 py-2 text-[13px]">
          {errors(error)}
        </div>
      ) : null}

      {summary ? (
        <div className="flex flex-col gap-4">
          {!summary.listener.is_fresh ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message={t("listenerWarning")}
              tone="warning"
            />
          ) : null}
          {!summary.processor.is_fresh ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message={t("processorWarning")}
              tone="warning"
            />
          ) : null}
          {failedRawEvents > 0 ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message={t("failedWarning")}
              tone="danger"
            />
          ) : null}
          {isBreakerOpen ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message={t("breakerWarning", {
                delay: f.number(Math.round(ingestion.breaker_current_delay_ms))
              })}
              tone="danger"
            />
          ) : null}
          {ingestion.write_drop_count > 0 ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message={t("dropWarning", { count: f.number(ingestion.write_drop_count) })}
              tone="warning"
            />
          ) : null}
          {ingestion.stream_consumer_redelivered > 0 ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message={t("redeliveryWarning", {
                count: f.number(ingestion.stream_consumer_redelivered)
              })}
              tone="warning"
            />
          ) : null}
          {ingestion.stream_error ? (
            <OperationsNotice
              icon={<TriangleAlert className="h-4 w-4" />}
              message={t("streamError", { error: ingestion.stream_error })}
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

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <MetricCard
              detail={t("senders", { count: f.number(summary.counts.senders) })}
              icon={<MessageSquareText className="h-3.5 w-3.5" />}
              iconTone="accent"
              label={t("messages")}
              value={f.number(summary.counts.messages)}
            />
            <MetricCard
              detail={t("ackWaiting", { count: f.number(ingestion.stream_consumer_ack_pending) })}
              icon={<HardDrive className="h-3.5 w-3.5" />}
              iconTone="muted"
              label={t("backlog")}
              value={f.number(ingestion.queue_depth)}
            />
            <MetricCard
              detail={failedRawEvents > 0 ? t("review") : t("clean")}
              detailTone={failedRawEvents > 0 ? "danger" : "muted"}
              icon={<TriangleAlert className="h-3.5 w-3.5" />}
              iconTone={failedRawEvents > 0 ? "danger" : "muted"}
              label={t("failed")}
              onDetailClick={failedRawEvents > 0 ? () => setFailedModalOpen(true) : undefined}
              value={f.number(failedRawEvents)}
            />
            <MetricCard
              detail={t("tables", { count: f.number(summary.storage.tables.length) })}
              icon={<Database className="h-3.5 w-3.5" />}
              iconTone="muted"
              label={t("dbSize")}
              value={f.bytes(summary.storage.database_bytes)}
            />
          </div>

          <div className="rounded-lg border border-border bg-panel p-5">
            <div className="mb-4 flex items-center justify-between">
              <div className="flex flex-col gap-0.5">
                <span className="text-[14px] font-semibold text-foreground">{t("ingestion")}</span>
                <span className="font-mono text-[11px] text-faint">{t("ingestionDetail")}</span>
              </div>
              <div
                className={`flex items-center gap-1.5 rounded-full bg-elevated px-2.5 py-1 font-mono text-[10px] font-semibold tracking-[0.8px] ${
                  isBreakerOpen ? "text-danger" : "text-accent"
                }`}
              >
                <span
                  className={`h-1.5 w-1.5 rounded-full ${isBreakerOpen ? "bg-danger" : "bg-accent"}`}
                />
                {isBreakerOpen ? t("open") : t("closed")}
              </div>
            </div>
            <div className="overflow-x-auto rounded-md border border-border">
              <div className="flex min-w-[720px] divide-x divide-border">
                <IngestionCell
                  label={t("streamPending")}
                  value={f.number(ingestion.stream_consumer_pending)}
                />
                <IngestionCell
                  label={t("ackPending")}
                  value={f.number(ingestion.stream_consumer_ack_pending)}
                />
                <IngestionCell
                  label={t("redelivery")}
                  value={f.number(ingestion.stream_consumer_redelivered)}
                />
                <IngestionCell
                  label={t("oldest")}
                  value={
                    ingestion.stream_oldest_pending_age_seconds > 0
                      ? t("seconds", {
                          value: f.number(ingestion.stream_oldest_pending_age_seconds)
                        })
                      : "—"
                  }
                />
                <IngestionCell label={t("legacy")} value={f.number(ingestion.legacy_queue_depth)} />
                <IngestionCell
                  label={t("chFailures")}
                  value={f.number(ingestion.clickhouse_insert_failures)}
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
  const t = useTranslations("operations");
  const f = useUiFormat();
  return (
    <div className="flex min-w-0 items-center gap-2">
      <span
        className={`h-2 w-2 shrink-0 rounded-full ${heartbeat.is_fresh ? "bg-accent" : "bg-warning"}`}
      />
      <span className="min-w-0 break-words text-[13px] font-medium text-foreground">
        {t("heartbeat", {
          label,
          state: heartbeat.is_fresh ? t("fresh") : t("stale"),
          age:
            heartbeat.seconds_since_last_seen !== null
              ? t("signalAge", { seconds: f.number(heartbeat.seconds_since_last_seen) })
              : t("noSignal")
        })}
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
    <div className="flex min-w-0 flex-col gap-2.5 rounded-lg border border-border bg-panel p-4">
      <div className="flex items-center justify-between">
        <span className="font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
          {label}
        </span>
        <span className={iconClass}>{icon}</span>
      </div>
      <span className="break-words text-[22px] font-semibold leading-tight tracking-tight text-foreground">
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
      <span className="shrink-0">{icon}</span>
      <span className="text-foreground">{message}</span>
    </div>
  );
}

function getStatusCount(summary: OperationsSummary, status: string) {
  return summary.raw_event_status_counts[status] ?? 0;
}
