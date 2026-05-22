"use client";

import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { Loader2, RefreshCcw, Save, ShieldAlert, TriangleAlert, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  confirmDataCleanup,
  getDataManagementSummary,
  previewDataCleanup,
  updateRetentionSettings
} from "@/features/data-management/api";
import type {
  DataCleanupPreview,
  DataCleanupResult,
  DataCleanupTarget,
  DataManagementSummary,
  RetentionDays
} from "@/types/api";

type CleanupFormState = {
  target: DataCleanupTarget;
  channel_slug: string;
  sender: string;
};

const RETENTION_OPTIONS: Array<{ label: string; value: string }> = [
  { label: "Sonsuza kadar", value: "forever" },
  { label: "30 gün", value: "30" },
  { label: "90 gün", value: "90" }
];

const CLEANUP_TARGETS: Array<{ label: string; value: DataCleanupTarget }> = [
  { label: "Eski mesajlar", value: "old_messages" },
  { label: "Eski raw eventler", value: "old_raw_events" },
  { label: "Kanal", value: "channel" },
  { label: "Gönderen", value: "sender" }
];

export function DataManagementPanel() {
  const [summary, setSummary] = useState<DataManagementSummary | null>(null);
  const [messageRetention, setMessageRetention] = useState<RetentionDays>(null);
  const [rawEventRetention, setRawEventRetention] = useState<RetentionDays>(null);
  const [cleanupForm, setCleanupForm] = useState<CleanupFormState>({
    target: "old_messages",
    channel_slug: "",
    sender: ""
  });
  const [preview, setPreview] = useState<DataCleanupPreview | null>(null);
  const [confirmationText, setConfirmationText] = useState("");
  const [result, setResult] = useState<DataCleanupResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [isConfirming, setIsConfirming] = useState(false);

  const loadSummary = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const nextSummary = await getDataManagementSummary();
      setSummary(nextSummary);
      setMessageRetention(nextSummary.retention_settings.message_retention_days);
      setRawEventRetention(nextSummary.retention_settings.raw_event_retention_days);
    } catch (caught) {
      setError(resolveDataError(caught));
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadSummary();
  }, [loadSummary]);

  const cleanupRequest = useMemo(() => buildCleanupRequest(cleanupForm), [cleanupForm]);
  const canConfirm =
    preview?.can_execute === true &&
    confirmationText === preview.confirmation_text &&
    !isConfirming;

  async function saveRetention(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsSaving(true);
    setError(null);
    setResult(null);

    try {
      const settings = await updateRetentionSettings({
        message_retention_days: messageRetention,
        raw_event_retention_days: rawEventRetention
      });
      setSummary((current) => (current ? { ...current, retention_settings: settings } : current));
      setPreview(null);
      setConfirmationText("");
    } catch (caught) {
      setError(resolveDataError(caught));
    } finally {
      setIsSaving(false);
    }
  }

  async function submitPreview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsPreviewing(true);
    setError(null);
    setResult(null);
    setConfirmationText("");

    try {
      setPreview(await previewDataCleanup(cleanupRequest));
    } catch (caught) {
      setPreview(null);
      setError(resolveDataError(caught));
    } finally {
      setIsPreviewing(false);
    }
  }

  async function submitConfirm() {
    if (!preview || !canConfirm) return;

    setIsConfirming(true);
    setError(null);

    try {
      const nextResult = await confirmDataCleanup({
        ...cleanupRequest,
        confirmation_text: confirmationText
      });
      setResult(nextResult);
      setPreview(null);
      setConfirmationText("");
      await loadSummary();
    } catch (caught) {
      setError(resolveDataError(caught));
    } finally {
      setIsConfirming(false);
    }
  }

  return (
    <section className="rounded-lg border border-border bg-panel p-5">
      <div className="mb-5 flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-col gap-0.5">
          <h2 className="text-[14px] font-semibold text-foreground">Veri Yönetimi</h2>
          <span className="font-mono text-[11px] text-faint">
            Retention ayarları ve kontrollü cleanup
          </span>
        </div>
        <Button
          disabled={isLoading}
          onClick={() => void loadSummary()}
          size="sm"
          type="button"
          variant="outline"
        >
          {isLoading ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <RefreshCcw className="h-3 w-3" />
          )}
          Yenile
        </Button>
      </div>

      {isLoading && !summary ? (
        <div className="rounded-md border border-border bg-elevated px-4 py-8 text-center text-[13px] text-muted-foreground">
          Veri yönetimi bilgileri yükleniyor...
        </div>
      ) : null}

      {error ? (
        <div className="mb-4 rounded-md border border-danger bg-elevated px-3 py-2 text-[13px]">
          {error}
        </div>
      ) : null}

      {summary ? (
        <div className="flex flex-col gap-5">
          {/* Stats */}
          <div className="grid gap-3 md:grid-cols-3">
            <StatCard label="Veritabanı" value={formatBytes(summary.database_bytes)} />
            <StatCard label="Mesaj" value={formatNumber(summary.counts.messages)} />
            <StatCard label="Raw Event" value={formatNumber(summary.counts.raw_events)} />
          </div>

          {/* Tables */}
          <div className="overflow-x-auto rounded-lg border border-border">
            <div className="min-w-[360px]">
              <div className="flex items-center border-b border-border px-3 py-2">
                <span className="flex-1 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
                  TABLO
                </span>
                <span className="w-28 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
                  SATIR
                </span>
                <span className="w-24 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
                  BOYUT
                </span>
              </div>
              {summary.tables.map((table) => (
                <div
                  className="flex items-center border-b border-border px-3 py-2.5 last:border-b-0"
                  key={table.table_name}
                >
                  <span className="flex-1 truncate font-mono text-[12px] text-foreground">
                    {table.table_name}
                  </span>
                  <span className="w-28 font-mono text-[12px] text-muted-foreground">
                    {formatNumber(table.row_count)}
                  </span>
                  <span className="w-24 font-mono text-[12px] text-muted-foreground">
                    {formatBytes(table.total_bytes)}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Retention settings */}
          <form
            className="rounded-lg border border-border bg-elevated p-4"
            onSubmit={saveRetention}
          >
            <div className="mb-4 flex items-center gap-2">
              <Save className="h-3.5 w-3.5 text-accent" />
              <span className="font-sans text-[13px] font-semibold text-foreground">
                Retention Ayarları
              </span>
            </div>
            <div className="grid gap-3 md:grid-cols-[1fr_1fr_auto]">
              <RetentionSelect
                label="Mesajlar"
                onChange={setMessageRetention}
                value={messageRetention}
              />
              <RetentionSelect
                label="Raw Eventler"
                onChange={setRawEventRetention}
                value={rawEventRetention}
              />
              <div className="flex items-end">
                <Button disabled={isSaving} type="submit">
                  {isSaving ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4" />
                  )}
                  Kaydet
                </Button>
              </div>
            </div>
          </form>

          {/* Cleanup preview form */}
          <form
            className="rounded-lg border border-border bg-elevated p-4"
            onSubmit={submitPreview}
          >
            <div className="mb-4 flex items-center gap-2">
              <ShieldAlert className="h-3.5 w-3.5 text-accent" />
              <span className="font-sans text-[13px] font-semibold text-foreground">
                Cleanup Önizleme
              </span>
            </div>
            <div className="grid gap-3 md:grid-cols-[minmax(140px,180px)_minmax(0,1fr)_auto]">
              <div className="flex flex-col gap-1.5">
                <label
                  className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
                  htmlFor="cleanup-target"
                >
                  HEDEF
                </label>
                <select
                  className="h-[38px] rounded-md border border-border-strong bg-panel px-3 font-sans text-[13px] text-foreground outline-none focus:border-accent"
                  id="cleanup-target"
                  onChange={(e) =>
                    setCleanupForm((c) => ({ ...c, target: e.target.value as DataCleanupTarget }))
                  }
                  value={cleanupForm.target}
                >
                  {CLEANUP_TARGETS.map((t) => (
                    <option key={t.value} value={t.value}>
                      {t.label}
                    </option>
                  ))}
                </select>
              </div>

              {cleanupForm.target === "channel" ? (
                <div className="flex flex-col gap-1.5">
                  <label
                    className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
                    htmlFor="cleanup-channel"
                  >
                    KANAL SLUG
                  </label>
                  <input
                    className="h-[38px] rounded-md border border-border-strong bg-panel px-3 font-sans text-[13px] text-foreground outline-none focus:border-accent placeholder:text-faint"
                    id="cleanup-channel"
                    onChange={(e) =>
                      setCleanupForm((c) => ({ ...c, channel_slug: e.target.value }))
                    }
                    placeholder="örn. hype"
                    value={cleanupForm.channel_slug}
                  />
                </div>
              ) : cleanupForm.target === "sender" ? (
                <div className="flex flex-col gap-1.5">
                  <label
                    className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
                    htmlFor="cleanup-sender"
                  >
                    GÖNDEREN
                  </label>
                  <input
                    className="h-[38px] rounded-md border border-border-strong bg-panel px-3 font-sans text-[13px] text-foreground outline-none focus:border-accent placeholder:text-faint"
                    id="cleanup-sender"
                    onChange={(e) => setCleanupForm((c) => ({ ...c, sender: e.target.value }))}
                    placeholder="örn. yavuz"
                    value={cleanupForm.sender}
                  />
                </div>
              ) : (
                <div className="flex items-center rounded-md border border-border bg-panel px-3 py-2 font-sans text-[12px] text-muted-foreground">
                  Mevcut retention ayarını kullanır. Sonsuza kadar seçiliyse delete çalışmaz.
                </div>
              )}

              <div className="flex items-end">
                <Button disabled={isPreviewing} type="submit" variant="outline">
                  {isPreviewing ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Trash2 className="h-4 w-4 text-accent" />
                  )}
                  Dry-run
                </Button>
              </div>
            </div>
          </form>

          {/* Preview result */}
          {preview ? (
            <div className="rounded-lg border border-warning bg-elevated p-4">
              <div className="mb-3 flex items-center gap-2">
                <TriangleAlert className="h-3.5 w-3.5 text-warning" />
                <span className="font-sans text-[13px] font-semibold text-foreground">
                  Cleanup Önizleme Sonucu
                </span>
              </div>
              <div className="mb-4 grid gap-3 md:grid-cols-3">
                <StatCard label="MESAJ" value={formatNumber(preview.affected.messages)} />
                <StatCard label="RAW EVENT" value={formatNumber(preview.affected.raw_events)} />
                <StatCard label="TOPLAM" value={formatNumber(preview.affected.total)} />
              </div>
              {preview.reason ? (
                <p className="mb-4 font-sans text-[12px] text-muted-foreground">{preview.reason}</p>
              ) : null}
              <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto]">
                <div className="flex flex-col gap-1.5">
                  <label
                    className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
                    htmlFor="confirm-text"
                  >
                    ONAY METNİ:{" "}
                    <span className="font-mono text-accent">{preview.confirmation_text}</span>
                  </label>
                  <input
                    className="h-[38px] rounded-md border border-border-strong bg-panel px-3 font-sans text-[13px] text-foreground outline-none focus:border-accent placeholder:text-faint"
                    id="confirm-text"
                    onChange={(e) => setConfirmationText(e.target.value)}
                    placeholder={preview.confirmation_text}
                    value={confirmationText}
                  />
                </div>
                <div className="flex items-end">
                  <Button
                    className="border-danger text-danger hover:bg-danger/10"
                    disabled={!canConfirm}
                    onClick={() => void submitConfirm()}
                    type="button"
                    variant="outline"
                  >
                    {isConfirming ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Trash2 className="h-4 w-4" />
                    )}
                    Sil
                  </Button>
                </div>
              </div>
            </div>
          ) : null}

          {result ? (
            <div className="rounded-lg border border-accent bg-elevated px-4 py-3 font-sans text-[13px] text-foreground">
              Cleanup tamamlandı: {formatNumber(result.deleted.messages)} mesaj,{" "}
              {formatNumber(result.deleted.raw_events)} raw event silindi.
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border border-border bg-panel px-4 py-3">
      <span className="font-mono text-[10px] font-medium tracking-[0.8px] text-faint">{label}</span>
      <span className="truncate font-sans text-[18px] font-semibold text-foreground" title={value}>
        {value}
      </span>
    </div>
  );
}

function RetentionSelect({
  label,
  onChange,
  value
}: {
  label: string;
  onChange: (value: RetentionDays) => void;
  value: RetentionDays;
}) {
  const id = `retention-${label.toLowerCase().replace(/\s+/g, "-")}`;
  return (
    <div className="flex flex-col gap-1.5">
      <label
        className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
        htmlFor={id}
      >
        {label}
      </label>
      <select
        className="h-[38px] rounded-md border border-border-strong bg-panel px-3 font-sans text-[13px] text-foreground outline-none focus:border-accent"
        id={id}
        onChange={(e) => onChange(parseRetentionValue(e.target.value))}
        value={value ?? "forever"}
      >
        {RETENTION_OPTIONS.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}

function buildCleanupRequest(form: CleanupFormState) {
  return {
    target: form.target,
    channel_slug: form.target === "channel" ? form.channel_slug.trim() : null,
    sender: form.target === "sender" ? form.sender.trim() : null
  };
}

function parseRetentionValue(value: string): RetentionDays {
  if (value === "30") return 30;
  if (value === "90") return 90;
  return null;
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

function resolveDataError(error: unknown) {
  if (error instanceof Error) return error.message;
  return "Veri yönetimi işlemi tamamlanamadı.";
}
