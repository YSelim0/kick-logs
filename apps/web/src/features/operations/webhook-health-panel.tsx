"use client";

import { Loader2, RefreshCcw, TriangleAlert, Webhook } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { getWebhookHealth, triggerWebhookSync } from "@/features/operations/api";
import type { ChannelSyncStatus, WebhookHealth } from "@/types/api";

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
          <ChannelSyncTable channels={health.channels} />
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

function ChannelSyncTable({ channels }: { channels: ChannelSyncStatus[] }) {
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
        <table className="w-full min-w-[600px] text-[13px]">
          <thead>
            <tr className="border-b border-border bg-elevated">
              <th className="px-3 py-2 text-left font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                Kanal
              </th>
              <th className="px-3 py-2 text-left font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                Broadcaster ID
              </th>
              <th className="px-3 py-2 text-left font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                Event Tip
              </th>
              <th className="px-3 py-2 text-left font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                Durum
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {channels.flatMap((ch) =>
              ch.subscriptions.length > 0
                ? ch.subscriptions.map((sub, i) => (
                    <tr
                      className="hover:bg-elevated/40"
                      key={`${ch.followed_channel_id}-${sub.event_type}`}
                    >
                      {i === 0 ? (
                        <td
                          className="px-3 py-2 font-medium text-foreground"
                          rowSpan={ch.subscriptions.length}
                        >
                          {ch.slug}
                        </td>
                      ) : null}
                      {i === 0 ? (
                        <td
                          className="px-3 py-2 font-mono text-[11px] text-muted-foreground"
                          rowSpan={ch.subscriptions.length}
                        >
                          {ch.broadcaster_user_id > 0 ? ch.broadcaster_user_id : "—"}
                        </td>
                      ) : null}
                      <td className="px-3 py-2 font-mono text-[11px] text-muted-foreground">
                        {sub.event_type.replace("channel.subscription.", "")}
                      </td>
                      <td className="px-3 py-2">
                        <SubStatusPill status={sub.status} error={sub.latest_sync_error} />
                      </td>
                    </tr>
                  ))
                : [
                    <tr className="hover:bg-elevated/40" key={ch.followed_channel_id}>
                      <td className="px-3 py-2 font-medium text-foreground">{ch.slug}</td>
                      <td className="px-3 py-2 font-mono text-[11px] text-muted-foreground">
                        {ch.broadcaster_user_id > 0 ? ch.broadcaster_user_id : "—"}
                      </td>
                      <td
                        className="px-3 py-2 font-mono text-[11px] text-muted-foreground"
                        colSpan={2}
                      >
                        <span className="text-muted-foreground">abonelik yok</span>
                      </td>
                    </tr>
                  ]
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function SubStatusPill({ status, error }: { status: string; error?: string | null }) {
  const color =
    status === "active"
      ? "bg-accent/20 text-accent"
      : status === "error"
        ? "bg-danger/20 text-danger"
        : "bg-elevated text-muted-foreground";

  const label = status === "active" ? "aktif" : status === "error" ? "hata" : status;

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-wider ${color}`}
      title={error ?? undefined}
    >
      {label}
    </span>
  );
}
