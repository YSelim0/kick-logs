"use client";

import {
  CheckCircle2,
  CircleAlert,
  Loader2,
  RefreshCcw,
  TriangleAlert,
  Webhook
} from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog";
import { getWebhookHealth, triggerWebhookSync } from "@/features/operations/api";
import type { ChannelSyncStatus, EventSubStatus, WebhookHealth } from "@/types/api";

export function WebhookHealthPanel() {
  const [health, setHealth] = useState<WebhookHealth | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);

  const load = useCallback(async (mode: "initial" | "refresh" = "initial") => {
    if (mode === "refresh") setIsRefreshing(true);
    else setIsLoading(true);
    setError(null);
    try {
      setHealth(await getWebhookHealth());
    } catch {
      setError("Webhook durumu alınamadı.");
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  }, []);

  useEffect(() => {
    void load("initial");
  }, [load]);

  const handleSync = async () => {
    setIsSyncing(true);
    try {
      await triggerWebhookSync();
      await load("refresh");
    } catch {
      setError("Senkronizasyon tetiklenemedi.");
    } finally {
      setIsSyncing(false);
    }
  };

  return (
    <section className="rounded-lg border border-border bg-panel p-5">
      <div className="mb-5 flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-col gap-1">
          <h2 className="text-[22px] font-semibold tracking-tight text-foreground">Webhooks</h2>
          <p className="font-sans text-[13px] text-muted-foreground">
            Abonelik senkronizasyonu ve inbox durumu
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            disabled={isLoading || isRefreshing}
            onClick={() => void load("refresh")}
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
          <Button
            disabled={isLoading || isSyncing}
            onClick={() => void handleSync()}
            size="sm"
            type="button"
            variant="outline"
          >
            {isSyncing ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <Webhook className="h-3 w-3" />
            )}
            Senkronize Et
          </Button>
        </div>
      </div>

      {isLoading && !health ? (
        <div className="rounded-md border border-border bg-elevated px-4 py-8 text-center text-[13px] text-muted-foreground">
          Webhook durumu yükleniyor...
        </div>
      ) : null}

      {error ? (
        <div className="mb-4 rounded-md border border-danger bg-elevated px-3 py-2 text-[13px] text-danger">
          {error}
        </div>
      ) : null}

      {health ? (
        <div className="flex flex-col gap-4">
          <ConfigWarnings health={health} />
          <InboxCounts health={health} />
          <ChannelSyncTable channels={health.channels} eventTypes={health.configured_event_types} />
        </div>
      ) : null}
    </section>
  );
}

function ConfigWarnings({ health }: { health: WebhookHealth }) {
  const warnings: string[] = [];
  if (health.missing_client_credentials)
    warnings.push("Kick client credentials eksik — abonelik senkronizasyonu devre dışı.");
  if (health.missing_webhook_public_key)
    warnings.push("Webhook public key eksik — POST /webhooks/kick tüm istekleri reddediyor.");
  if (!health.webhook_sync_enabled)
    warnings.push("KICK_WEBHOOK_SYNC_ENABLED=false — senkronizasyon devre dışı.");

  if (warnings.length === 0) return null;
  return (
    <div className="flex flex-col gap-2">
      {warnings.map((w) => (
        <div
          className="flex items-center gap-2 rounded-md border border-warning bg-elevated px-3 py-2 text-[13px] text-warning"
          key={w}
        >
          <TriangleAlert className="h-4 w-4 shrink-0" />
          <span className="text-foreground">{w}</span>
        </div>
      ))}
    </div>
  );
}

function InboxCounts({ health }: { health: WebhookHealth }) {
  const counts = health.inbox_counts;
  const cells: { label: string; value: number; tone?: "danger" | "warning" }[] = [
    { label: "Pending", value: counts["pending"] ?? 0 },
    { label: "İşlendi", value: counts["processed"] ?? 0 },
    {
      label: "Başarısız",
      value: counts["failed"] ?? 0,
      tone: (counts["failed"] ?? 0) > 0 ? "danger" : undefined
    },
    { label: "Yoksayıldı", value: counts["ignored"] ?? 0 }
  ];

  return (
    <div className="rounded-lg border border-border bg-panel p-4">
      <div className="mb-3 flex items-center justify-between">
        <span className="text-[14px] font-semibold text-foreground">Inbox</span>
        {health.latest_webhook_received_at ? (
          <span className="font-mono text-[11px] text-muted-foreground">
            son webhook:{" "}
            {new Intl.DateTimeFormat("tr-TR", {
              day: "2-digit",
              month: "short",
              hour: "2-digit",
              minute: "2-digit"
            }).format(new Date(health.latest_webhook_received_at))}
          </span>
        ) : (
          <span className="font-mono text-[11px] text-muted-foreground">
            henüz webhook alınmadı
          </span>
        )}
      </div>
      <div className="overflow-x-auto rounded-md border border-border">
        <div className="flex min-w-[320px] divide-x divide-border">
          {cells.map((cell) => (
            <div className="flex flex-1 flex-col gap-1 bg-panel px-3.5 py-3" key={cell.label}>
              <span className="font-mono text-[10px] tracking-[0.5px] text-muted-foreground">
                {cell.label}
              </span>
              <span
                className={`text-[18px] font-semibold ${cell.tone === "danger" ? "text-danger" : "text-foreground"}`}
              >
                {new Intl.NumberFormat("tr-TR").format(cell.value)}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function ChannelSyncTable({
  channels,
  eventTypes
}: {
  channels: ChannelSyncStatus[];
  eventTypes: string[];
}) {
  const [selectedChannel, setSelectedChannel] = useState<ChannelSyncStatus | null>(null);

  if (channels.length === 0) {
    return (
      <div className="rounded-md border border-border bg-elevated px-4 py-4 text-[13px] text-muted-foreground">
        Takip edilen kanal yok.
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-border bg-panel p-4">
      <div className="mb-3">
        <span className="text-[14px] font-semibold text-foreground">Kanal Abonelikleri</span>
      </div>
      <div className="overflow-x-auto rounded-md border border-border">
        <table className="w-full min-w-[520px] text-[13px]">
          <thead>
            <tr className="border-b border-border bg-elevated">
              <th className="px-3 py-2 text-left font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                Kanal
              </th>
              <th className="px-3 py-2 text-left font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                Broadcaster ID
              </th>
              <th className="px-3 py-2 text-left font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                Webhook Durumu
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {channels.map((ch) => (
              <tr className="hover:bg-elevated/40" key={ch.followed_channel_id}>
                <td className="px-3 py-2 font-medium text-foreground">{ch.slug}</td>
                <td className="px-3 py-2 font-mono text-[11px] text-muted-foreground">
                  {ch.broadcaster_user_id > 0 ? ch.broadcaster_user_id : "—"}
                </td>
                <td className="px-3 py-2">
                  <ChannelSyncSummaryButton
                    channel={ch}
                    eventTypes={eventTypes}
                    onClick={() => setSelectedChannel(ch)}
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <ChannelSyncDetailsDialog
        channel={selectedChannel}
        eventTypes={eventTypes}
        onClose={() => setSelectedChannel(null)}
      />
    </div>
  );
}

function ChannelSyncSummaryButton({
  channel,
  eventTypes,
  onClick
}: {
  channel: ChannelSyncStatus;
  eventTypes: string[];
  onClick: () => void;
}) {
  const state = getChannelSyncState(channel, eventTypes);
  const isHealthy = state.kind === "active";
  const isInactive = state.kind === "inactive";

  return (
    <button
      className={`inline-flex min-w-[78px] items-center justify-center gap-1.5 rounded-md border px-2.5 py-1 font-mono text-[10px] font-semibold uppercase tracking-wider transition hover:border-border-strong ${
        isHealthy
          ? "border-accent/40 bg-accent/10 text-accent"
          : isInactive
            ? "border-warning/40 bg-warning/10 text-warning"
            : "border-danger/50 bg-danger/10 text-danger"
      }`}
      onClick={onClick}
      type="button"
    >
      {isHealthy ? (
        <CheckCircle2 className="h-3 w-3" />
      ) : isInactive ? (
        <TriangleAlert className="h-3 w-3" />
      ) : (
        <CircleAlert className="h-3 w-3" />
      )}
      {state.label}
    </button>
  );
}

function ChannelSyncDetailsDialog({
  channel,
  eventTypes,
  onClose
}: {
  channel: ChannelSyncStatus | null;
  eventTypes: string[];
  onClose: () => void;
}) {
  if (!channel) return null;

  const rows = buildEventRows(channel, eventTypes);
  const state = getChannelSyncState(channel, eventTypes);

  return (
    <Dialog open={Boolean(channel)} onOpenChange={(open) => (!open ? onClose() : undefined)}>
      <DialogContent className="max-h-[85vh] max-w-2xl overflow-y-auto border-border bg-panel p-0 text-foreground shadow-none">
        <DialogClose onClose={onClose} />
        <div className="border-b border-border px-5 py-4">
          <DialogHeader>
            <DialogTitle className="text-[18px]">Webhook Detayı</DialogTitle>
            <DialogDescription className="text-[12px] text-muted-foreground">
              Kanal bilgileri ve beklenen Kick abonelik event durumları
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-2 sm:grid-cols-3">
            <DetailCell label="KANAL" value={channel.slug} />
            <DetailCell
              label="BROADCASTER ID"
              value={channel.broadcaster_user_id > 0 ? String(channel.broadcaster_user_id) : "—"}
            />
            <DetailCell
              label="GENEL DURUM"
              value={state.kind === "active" ? "aktif" : state.label.toLocaleLowerCase("tr-TR")}
            />
          </div>
        </div>

        <div className="px-5 py-4">
          <div className="overflow-hidden rounded-md border border-border">
            {rows.map((row) => (
              <div
                className="border-b border-border bg-panel px-3 py-3 last:border-b-0"
                key={row.eventType}
              >
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-mono text-[11px] text-muted-foreground">
                      {formatEventType(row.eventType)}
                    </span>
                    <SubStatusPill
                      error={row.subscription?.latest_sync_error}
                      missing={!row.subscription}
                      status={row.subscription?.status ?? "missing"}
                    />
                  </div>
                  <div className="mt-2 grid gap-1 font-mono text-[10px] text-muted-foreground sm:grid-cols-2">
                    <span title={row.subscription?.kick_subscription_id || undefined}>
                      ID: {row.subscription?.kick_subscription_id || "—"}
                    </span>
                    <span>Sync: {formatDateTime(row.subscription?.synced_at)}</span>
                  </div>
                  {row.subscription?.latest_sync_error ? (
                    <p className="mt-2 rounded-md border border-danger/40 bg-danger/10 px-2 py-1 text-[12px] text-danger">
                      {row.subscription.latest_sync_error}
                    </p>
                  ) : null}
                </div>
              </div>
            ))}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function DetailCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-border bg-elevated px-3 py-2">
      <span className="block font-mono text-[10px] tracking-[0.5px] text-muted-foreground">
        {label}
      </span>
      <span className="mt-1 block truncate text-[13px] font-medium text-foreground" title={value}>
        {value}
      </span>
    </div>
  );
}

function SubStatusPill({
  status,
  error,
  missing = false
}: {
  status: string;
  error?: string | null;
  missing?: boolean;
}) {
  const hasError = Boolean(error) || status === "error";
  const isActive = status === "active" && !hasError;
  const color = isActive
    ? "bg-accent/20 text-accent"
    : hasError
      ? "bg-danger/20 text-danger"
      : "bg-warning/10 text-warning";

  const label = isActive
    ? "aktif"
    : hasError
      ? "hata"
      : missing || status === "deleted" || status === "missing"
        ? "aktif değil"
        : status;

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-wider ${color}`}
      title={error ?? undefined}
    >
      {label}
    </span>
  );
}

function buildEventRows(channel: ChannelSyncStatus, eventTypes: string[]) {
  const expectedEventTypes =
    eventTypes.length > 0
      ? eventTypes
      : channel.subscriptions.map((subscription) => subscription.event_type);

  return expectedEventTypes.map((eventType) => ({
    eventType,
    subscription: channel.subscriptions.find(
      (subscription) => subscription.event_type === eventType
    )
  }));
}

function getChannelSyncState(channel: ChannelSyncStatus, eventTypes: string[]) {
  const rows = buildEventRows(channel, eventTypes);
  const errorCount = rows.filter((row) => isSubscriptionError(row.subscription)).length;
  const inactiveCount = rows.filter(
    (row) => !isSubscriptionError(row.subscription) && isSubscriptionInactive(row.subscription)
  ).length;

  if (errorCount > 0) {
    return { kind: "error" as const, label: `${errorCount} Hata` };
  }

  if (inactiveCount > 0) {
    return { kind: "inactive" as const, label: "Aktif değil" };
  }

  return { kind: "active" as const, label: "aktif" };
}

function isSubscriptionError(subscription?: EventSubStatus) {
  return Boolean(subscription?.latest_sync_error) || subscription?.status === "error";
}

function isSubscriptionInactive(subscription?: EventSubStatus) {
  return !subscription || subscription.status !== "active";
}

function formatEventType(eventType: string) {
  return eventType.replace("channel.subscription.", "");
}

function formatDateTime(value?: string | null) {
  if (!value) return "—";
  return new Intl.DateTimeFormat("tr-TR", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}
