"use client";

import { useTranslations } from "next-intl";
import { useUiFormat } from "@/i18n/use-ui-format";

import { AlertTriangle, Loader2, Trash2 } from "lucide-react";
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
import { clearFailedEvents, getFailedEvents } from "@/features/operations/api";
import type { FailedRawEvent } from "@/types/api";

type ActionState = "idle" | "loading" | "success" | "error";

export function FailedEventsModal({
  open,
  onOpenChange,
  onActionComplete
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onActionComplete?: () => void;
}) {
  const t = useTranslations("failedEvents");
  const f = useUiFormat();
  const [events, setEvents] = useState<FailedRawEvent[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [clearState, setClearState] = useState<ActionState>("idle");
  const [clearedCount, setClearedCount] = useState<number | null>(null);
  const [loadFailed, setLoadFailed] = useState(false);

  const load = useCallback(async () => {
    setIsLoading(true);
    setLoadFailed(false);
    try {
      const res = await getFailedEvents();
      setEvents(res.events ?? []);
      setTotal(res.total ?? 0);
    } catch {
      setEvents([]);
      setLoadFailed(true);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) {
      void load();
      setClearState("idle");
      setClearedCount(null);
    }
  }, [open, load]);

  const handleClear = async () => {
    setClearState("loading");
    setClearedCount(null);
    try {
      const res = await clearFailedEvents();
      setClearState("success");
      setClearedCount(res.affected);
      onActionComplete?.();
      void load();
    } catch {
      setClearState("error");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[80vh] max-w-3xl flex-col overflow-hidden border-accent bg-black">
        <DialogClose onClose={() => onOpenChange(false)} />
        <div className="shrink-0 pr-6">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-foreground">
              <AlertTriangle className="h-4 w-4 text-accent" />
              {t("title")}
            </DialogTitle>
            <DialogDescription className="text-muted-foreground">
              {t("description")}
            </DialogDescription>
          </DialogHeader>
        </div>

        <div className="flex min-h-0 flex-col gap-4 overflow-y-auto">
          <p className="rounded-md border border-border bg-elevated px-3 py-2 text-xs text-muted-foreground">
            {t("explanation")}
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <Button
              disabled={clearState === "loading" || total === 0}
              onClick={() => void handleClear()}
              size="sm"
              variant="outline"
            >
              {clearState === "loading" ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Trash2 className="h-4 w-4 text-accent" />
              )}
              {t("clear")}
            </Button>
            {clearedCount !== null || clearState === "error" ? (
              <span
                className={`text-xs ${clearState === "error" ? "text-accent" : "text-primary"}`}
              >
                {clearState === "error"
                  ? t("clearError")
                  : t("cleared", { count: clearedCount ?? 0 })}
              </span>
            ) : null}
            <span className="ml-auto text-xs text-muted-foreground">
              {t("count", { count: total })}
            </span>
          </div>

          <div className="shrink-0 overflow-x-auto rounded-md border border-border">
            {isLoading ? (
              <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                {t("loading")}
              </div>
            ) : loadFailed ? (
              <p role="alert" className="p-4 text-sm text-danger">
                {t("loadError")}
              </p>
            ) : events.length === 0 ? (
              <div className="py-8 text-center text-sm text-muted-foreground">{t("empty")}</div>
            ) : (
              <table className="w-full min-w-[520px] text-xs">
                <thead className="sticky top-0 border-b border-border bg-kick-background">
                  <tr>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">
                      {t("channel")}
                    </th>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">
                      {t("error")}
                    </th>
                    <th className="px-3 py-2 text-center font-medium text-muted-foreground">
                      {t("attempts")}
                    </th>
                    <th className="px-3 py-2 text-right font-medium text-muted-foreground">
                      {t("lastError")}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {events.map((ev) => (
                    <tr
                      key={ev.raw_event_id}
                      className="border-b border-border/50 last:border-0 hover:bg-kick-background/60"
                    >
                      <td className="px-3 py-2 font-medium text-foreground">
                        {ev.channel_slug || "-"}
                      </td>
                      <td
                        className="max-w-xs truncate px-3 py-2 text-muted-foreground"
                        title={ev.error_message}
                      >
                        {ev.error_message || "-"}
                      </td>
                      <td className="px-3 py-2 text-center text-muted-foreground">
                        {f.number(ev.attempts)}
                      </td>
                      <td className="px-3 py-2 text-right text-muted-foreground">
                        {f.dateTime(ev.failed_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
