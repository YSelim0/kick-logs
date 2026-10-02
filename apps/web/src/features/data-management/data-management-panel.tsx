"use client";

import { useTranslations } from "next-intl";
import { useUiFormat } from "@/i18n/use-ui-format";
import { getUiErrorKey, type UiErrorKey } from "@/i18n/errors";

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

const RETENTION_OPTIONS = ["forever", "30", "90"] as const;

const CLEANUP_TARGETS: DataCleanupTarget[] = [
  "old_messages",
  "old_raw_events",
  "channel",
  "sender"
];

export function DataManagementPanel() {
  const t = useTranslations("dataManagement");
  const f = useUiFormat();
  const errors = useTranslations("common.errors");
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
  const [error, setError] = useState<UiErrorKey | null>(null);
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
      setError(getUiErrorKey(caught, "cleanup"));
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
      setError(getUiErrorKey(caught, "cleanup"));
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
      setError(getUiErrorKey(caught, "cleanup"));
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
      setError(getUiErrorKey(caught, "cleanup"));
    } finally {
      setIsConfirming(false);
    }
  }

  return (
    <section className="rounded-lg border border-border bg-panel p-5">
      <div className="mb-5 flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-col gap-0.5">
          <h2 className="text-[14px] font-semibold text-foreground">{t("title")}</h2>
          <span className="font-mono text-[11px] text-faint">{t("description")}</span>
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
        <div className="flex flex-col gap-5">
          {/* Stats */}
          <div className="grid gap-3 md:grid-cols-3">
            <StatCard label={t("database")} value={f.bytes(summary.database_bytes)} />
            <StatCard label={t("messages")} value={f.number(summary.counts.messages)} />
            <StatCard label={t("rawEvents")} value={f.number(summary.counts.raw_events)} />
          </div>

          {/* Tables */}
          <div className="overflow-x-auto rounded-lg border border-border">
            <div className="min-w-[360px]">
              <div className="flex items-center border-b border-border px-3 py-2">
                <span className="flex-1 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
                  {t("table")}
                </span>
                <span className="w-28 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
                  {t("rows")}
                </span>
                <span className="w-24 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
                  {t("size")}
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
                    {f.number(table.row_count)}
                  </span>
                  <span className="w-24 font-mono text-[12px] text-muted-foreground">
                    {f.bytes(table.total_bytes)}
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
                {t("retention")}
              </span>
            </div>
            <div className="grid gap-3 md:grid-cols-[1fr_1fr_auto]">
              <RetentionSelect
                label={t("messageRetention")}
                onChange={setMessageRetention}
                value={messageRetention}
              />
              <RetentionSelect
                label={t("rawRetention")}
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
                  {t("save")}
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
                {t("previewForm")}
              </span>
            </div>
            <div className="grid gap-3 md:grid-cols-[minmax(140px,180px)_minmax(0,1fr)_auto]">
              <div className="flex flex-col gap-1.5">
                <label
                  className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
                  htmlFor="cleanup-target"
                >
                  {t("target")}
                </label>
                <select
                  className="h-[38px] rounded-md border border-border-strong bg-panel px-3 font-sans text-[13px] text-foreground outline-none focus:border-accent"
                  id="cleanup-target"
                  onChange={(e) =>
                    setCleanupForm((c) => ({ ...c, target: e.target.value as DataCleanupTarget }))
                  }
                  value={cleanupForm.target}
                >
                  {CLEANUP_TARGETS.map((value) => (
                    <option key={value} value={value}>
                      {t(`targets.${value}`)}
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
                    {t("channelSlug")}
                  </label>
                  <input
                    className="h-[38px] rounded-md border border-border-strong bg-panel px-3 font-sans text-[13px] text-foreground outline-none focus:border-accent placeholder:text-faint"
                    id="cleanup-channel"
                    onChange={(e) =>
                      setCleanupForm((c) => ({ ...c, channel_slug: e.target.value }))
                    }
                    placeholder={t("channelPlaceholder")}
                    value={cleanupForm.channel_slug}
                  />
                </div>
              ) : cleanupForm.target === "sender" ? (
                <div className="flex flex-col gap-1.5">
                  <label
                    className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
                    htmlFor="cleanup-sender"
                  >
                    {t("sender")}
                  </label>
                  <input
                    className="h-[38px] rounded-md border border-border-strong bg-panel px-3 font-sans text-[13px] text-foreground outline-none focus:border-accent placeholder:text-faint"
                    id="cleanup-sender"
                    onChange={(e) => setCleanupForm((c) => ({ ...c, sender: e.target.value }))}
                    placeholder={t("senderPlaceholder")}
                    value={cleanupForm.sender}
                  />
                </div>
              ) : (
                <div className="flex items-center rounded-md border border-border bg-panel px-3 py-2 font-sans text-[12px] text-muted-foreground">
                  {t("retentionHint")}
                </div>
              )}

              <div className="flex items-end">
                <Button disabled={isPreviewing} type="submit" variant="outline">
                  {isPreviewing ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Trash2 className="h-4 w-4 text-accent" />
                  )}
                  {t("dryRun")}
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
                  {t("previewTitle")}
                </span>
              </div>
              <div className="mb-4 grid gap-3 md:grid-cols-3">
                <StatCard
                  label={t("previewMessages")}
                  value={f.number(preview.affected.messages)}
                />
                <StatCard label={t("previewRaw")} value={f.number(preview.affected.raw_events)} />
                <StatCard label={t("total")} value={f.number(preview.affected.total)} />
              </div>
              {preview.reason ? (
                <p className="mb-4 font-sans text-[12px] text-muted-foreground">
                  {preview.reason === "Retention is set to keep forever."
                    ? t("foreverReason")
                    : preview.reason}
                </p>
              ) : null}
              <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto]">
                <div className="flex flex-col gap-1.5">
                  <label
                    className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
                    htmlFor="confirm-text"
                  >
                    {t("confirmation")}{" "}
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
                    {t("delete")}
                  </Button>
                </div>
              </div>
            </div>
          ) : null}

          {result ? (
            <div className="rounded-lg border border-accent bg-elevated px-4 py-3 font-sans text-[13px] text-foreground">
              {t("complete", {
                messages: f.number(result.deleted.messages),
                rawEvents: f.number(result.deleted.raw_events)
              })}
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
  const t = useTranslations("dataManagement");
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
        {RETENTION_OPTIONS.map((value) => (
          <option key={value} value={value}>
            {value === "forever" ? t("forever") : t("days", { days: Number(value) })}
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
