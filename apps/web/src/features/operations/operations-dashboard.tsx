"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import {
  Activity,
  AlertTriangle,
  Clock3,
  Database,
  Gauge,
  HardDrive,
  Inbox,
  Loader2,
  MessageSquareText,
  RefreshCcw,
  Timer,
  Zap
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { getOperationsSummary } from "@/features/operations/api";
import type { IngestionHealth, OperationsSummary } from "@/types/api";

const EMPTY_INGESTION: IngestionHealth = {
  queue_depth: 0,
  oldest_pending_age_seconds: 0,
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

type MetricTone = "default" | "success" | "warning" | "danger";

export function OperationsDashboard() {
  const [summary, setSummary] = useState<OperationsSummary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);

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

  const cards = useMemo(() => (summary ? buildMetricCards(summary) : []), [summary]);
  const failedRawEvents = summary ? getStatusCount(summary, "failed") : 0;

  return (
    <section className="rounded-lg border border-border bg-black p-5">
      <div className="mb-5 flex flex-wrap items-start justify-between gap-4 border-b border-border pb-4">
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-md bg-kick-background text-primary">
            <Activity className="h-4 w-4" />
          </div>
          <div>
            <h2 className="text-base font-semibold">Operasyon Durumu</h2>
            <p className="text-xs text-muted-foreground">
              Listener, depolama ve raw event işleme sağlığı
            </p>
          </div>
        </div>

        <Button
          disabled={isLoading || isRefreshing}
          onClick={() => void loadSummary("refresh")}
          size="sm"
          type="button"
          variant="outline"
        >
          {isRefreshing ? (
            <Loader2 className="h-4 w-4 animate-spin text-accent" />
          ) : (
            <RefreshCcw className="h-4 w-4 text-accent" />
          )}
          Yenile
        </Button>
      </div>

      {isLoading && summary === null ? (
        <div className="rounded-md border border-border bg-kick-background px-4 py-8 text-center text-sm text-muted-foreground">
          Operasyon metrikleri yükleniyor...
        </div>
      ) : null}

      {error ? (
        <div className="mb-4 rounded-md border border-accent bg-kick-background px-3 py-2 text-sm">
          {error}
        </div>
      ) : null}

      {summary ? (
        <div className="space-y-4">
          {!summary.listener.is_fresh ? (
            <OperationsNotice
              icon={<AlertTriangle className="h-4 w-4" />}
              message="Listener heartbeat bayat. Listener çalışmıyor olabilir veya DB'ye yazamıyor olabilir."
              tone="warning"
            />
          ) : null}

          {failedRawEvents > 0 ? (
            <OperationsNotice
              icon={<AlertTriangle className="h-4 w-4" />}
              message="Başarısız raw event var. İşleme hatalarını backend loglarıyla incelemek gerekebilir."
              tone="danger"
            />
          ) : null}

          {(summary.ingestion ?? EMPTY_INGESTION).breaker_state === "open" ? (
            <OperationsNotice
              icon={<AlertTriangle className="h-4 w-4" />}
              message={`ClickHouse circuit breaker açık. Bekleme ${Math.round((summary.ingestion ?? EMPTY_INGESTION).breaker_current_delay_ms)} ms.`}
              tone="danger"
            />
          ) : null}

          {(summary.ingestion ?? EMPTY_INGESTION).write_drop_count > 0 ? (
            <OperationsNotice
              icon={<AlertTriangle className="h-4 w-4" />}
              message={`Buffered writer ${formatNumber((summary.ingestion ?? EMPTY_INGESTION).write_drop_count)} event düşürdü. Pusher trafiği buffer kapasitesini aştı.`}
              tone="warning"
            />
          ) : null}

          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {cards.map((card) => (
              <MetricCard key={card.label} {...card} />
            ))}
          </div>

          <div className="grid gap-3 text-xs text-muted-foreground md:grid-cols-3">
            <OperationsFact
              label="Son raw event"
              value={formatDate(summary.timestamps.latest_raw_event_received_at)}
            />
            <OperationsFact
              label="Son işlenen raw event"
              value={formatDate(summary.timestamps.latest_raw_event_processed_at)}
            />
            <OperationsFact
              label="En eski pending raw event"
              value={formatDate(summary.timestamps.oldest_pending_raw_event_received_at)}
            />
          </div>
        </div>
      ) : null}
    </section>
  );
}

function buildMetricCards(summary: OperationsSummary) {
  const failedRawEvents = getStatusCount(summary, "failed");
  const pendingRawEvents = getStatusCount(summary, "pending");
  const lastIngestAt =
    summary.timestamps.latest_message_at ??
    summary.timestamps.latest_raw_event_processed_at ??
    summary.timestamps.latest_raw_event_received_at;

  return [
    {
      label: "Listener",
      value: summary.listener.is_fresh ? "Canlı" : "Bayat",
      detail:
        summary.listener.seconds_since_last_seen === null
          ? "Heartbeat yok"
          : `${summary.listener.seconds_since_last_seen} sn önce`,
      icon: <Activity className="h-4 w-4" />,
      tone: summary.listener.is_fresh ? "success" : "warning"
    },
    {
      label: "Veritabanı",
      value: formatBytes(summary.storage.database_bytes),
      detail: `chat ${formatBytes(tableSize(summary, "chat_messages"))} / raw ${formatBytes(
        tableSize(summary, "raw_kick_events")
      )}`,
      icon: <Database className="h-4 w-4" />,
      tone: "default"
    },
    {
      label: "Mesaj",
      value: formatNumber(summary.counts.messages),
      detail: `${formatNumber(summary.counts.senders)} gönderici`,
      icon: <MessageSquareText className="h-4 w-4" />,
      tone: "default"
    },
    {
      label: "Raw Event",
      value: formatNumber(summary.counts.raw_events),
      detail: `${formatNumber(getStatusCount(summary, "processed"))} işlendi`,
      icon: <HardDrive className="h-4 w-4" />,
      tone: "default"
    },
    {
      label: "Failed Raw",
      value: formatNumber(failedRawEvents),
      detail: failedRawEvents > 0 ? "İnceleme gerekli" : "Temiz",
      icon: <AlertTriangle className="h-4 w-4" />,
      tone: failedRawEvents > 0 ? "danger" : "success"
    },
    {
      label: "Pending Raw",
      value: formatNumber(pendingRawEvents),
      detail: pendingRawEvents > 0 ? "Kuyrukta bekliyor" : "Bekleyen yok",
      icon: <Inbox className="h-4 w-4" />,
      tone: pendingRawEvents > 0 ? "warning" : "success"
    },
    {
      label: "Son Ingest",
      value: formatDate(lastIngestAt),
      detail: "En güncel mesaj/işleme",
      icon: <Clock3 className="h-4 w-4" />,
      tone: "default"
    },
    {
      label: "Queue Backlog",
      value: formatNumber((summary.ingestion ?? EMPTY_INGESTION).queue_depth),
      detail:
        (summary.ingestion ?? EMPTY_INGESTION).oldest_pending_age_seconds > 0
          ? `En eski ${formatDuration((summary.ingestion ?? EMPTY_INGESTION).oldest_pending_age_seconds)}`
          : "Bekleyen yok",
      icon: <Inbox className="h-4 w-4" />,
      tone: (summary.ingestion ?? EMPTY_INGESTION).queue_depth > 1000 ? "warning" : "default"
    },
    {
      label: "Writer Buffer",
      value: formatNumber((summary.ingestion ?? EMPTY_INGESTION).write_queue_depth),
      detail: `Tepe ${formatNumber((summary.ingestion ?? EMPTY_INGESTION).write_queue_high_water_mark)} / Drop ${formatNumber((summary.ingestion ?? EMPTY_INGESTION).write_drop_count)}`,
      icon: <Gauge className="h-4 w-4" />,
      tone: (summary.ingestion ?? EMPTY_INGESTION).write_drop_count > 0 ? "warning" : "default"
    },
    {
      label: "ClickHouse Breaker",
      value: (summary.ingestion ?? EMPTY_INGESTION).breaker_state === "open" ? "Açık" : "Kapalı",
      detail: `${formatNumber((summary.ingestion ?? EMPTY_INGESTION).clickhouse_insert_failures)} insert hatası`,
      icon: <Zap className="h-4 w-4" />,
      tone: (summary.ingestion ?? EMPTY_INGESTION).breaker_state === "open" ? "danger" : "success"
    },
    {
      label: "Son Flush",
      value: formatNumber((summary.ingestion ?? EMPTY_INGESTION).last_flush_size),
      detail:
        (summary.ingestion ?? EMPTY_INGESTION).last_flush_millis > 0
          ? `${formatNumber((summary.ingestion ?? EMPTY_INGESTION).last_flush_millis)} ms`
          : "Flush yok",
      icon: <Timer className="h-4 w-4" />,
      tone: "default"
    }
  ] satisfies Array<{
    label: string;
    value: string;
    detail: string;
    icon: ReactNode;
    tone: MetricTone;
  }>;
}

function MetricCard({
  detail,
  icon,
  label,
  tone,
  value
}: {
  detail: string;
  icon: ReactNode;
  label: string;
  tone: MetricTone;
  value: string;
}) {
  return (
    <div className={`rounded-lg border bg-kick-background p-4 ${toneClass(tone)}`}>
      <div className="mb-3 flex items-center justify-between gap-3">
        <div className="text-xs font-medium text-muted-foreground">{label}</div>
        <div className="text-accent">{icon}</div>
      </div>
      <div className="truncate text-xl font-semibold text-foreground" title={value}>
        {value}
      </div>
      <div className="mt-1 truncate text-xs text-muted-foreground" title={detail}>
        {detail}
      </div>
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
      className={
        tone === "danger"
          ? "flex items-center gap-2 rounded-md border border-accent bg-kick-background px-3 py-2 text-sm"
          : "flex items-center gap-2 rounded-md border border-primary/70 bg-kick-background px-3 py-2 text-sm"
      }
    >
      <span className={tone === "danger" ? "text-accent" : "text-primary"}>{icon}</span>
      <span>{message}</span>
    </div>
  );
}

function OperationsFact({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-md border border-border bg-kick-background px-3 py-2">
      <div>{label}</div>
      <div className="truncate font-medium text-foreground" title={value}>
        {value}
      </div>
    </div>
  );
}

function getStatusCount(summary: OperationsSummary, status: string) {
  return summary.raw_event_status_counts[status] ?? 0;
}

function tableSize(summary: OperationsSummary, tableName: string) {
  return summary.storage.tables.find((table) => table.table_name === tableName)?.total_bytes ?? 0;
}

function toneClass(tone: MetricTone) {
  if (tone === "success") {
    return "border-primary/60";
  }

  if (tone === "warning") {
    return "border-primary";
  }

  if (tone === "danger") {
    return "border-accent";
  }

  return "border-border";
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("tr-TR").format(value);
}

function formatBytes(value: number) {
  if (value <= 0) {
    return "0 B";
  }

  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const amount = value / 1024 ** index;
  return `${new Intl.NumberFormat("tr-TR", {
    maximumFractionDigits: index === 0 ? 0 : 2
  }).format(amount)} ${units[index]}`;
}

function formatDuration(seconds: number) {
  if (seconds <= 0) {
    return "0 sn";
  }
  if (seconds < 60) {
    return `${seconds} sn`;
  }
  if (seconds < 3600) {
    return `${Math.floor(seconds / 60)} dk`;
  }
  if (seconds < 86400) {
    return `${Math.floor(seconds / 3600)} sa`;
  }
  return `${Math.floor(seconds / 86400)} gün`;
}

function formatDate(value: string | null) {
  if (!value) {
    return "-";
  }

  return new Intl.DateTimeFormat("tr-TR", {
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    month: "2-digit",
    year: "numeric"
  }).format(new Date(value));
}

function resolveOperationsError(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }

  return "Operasyon metrikleri alınamadı.";
}
