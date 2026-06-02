"use client";

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
  const [events, setEvents] = useState<FailedRawEvent[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [clearState, setClearState] = useState<ActionState>("idle");
  const [actionMessage, setActionMessage] = useState<string | null>(null);

  const load = useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await getFailedEvents();
      setEvents(res.events ?? []);
      setTotal(res.total ?? 0);
    } catch {
      setEvents([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) {
      void load();
      setClearState("idle");
      setActionMessage(null);
    }
  }, [open, load]);

  const handleClear = async () => {
    setClearState("loading");
    setActionMessage(null);
    try {
      const res = await clearFailedEvents();
      setClearState("success");
      setActionMessage(`${res.affected} başarısız attempt temizlendi.`);
      onActionComplete?.();
      void load();
    } catch {
      setClearState("error");
      setActionMessage("Temizleme başarısız.");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[80vh] max-w-3xl overflow-hidden border-accent bg-black">
        <DialogClose onClose={() => onOpenChange(false)} />
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-foreground">
            <AlertTriangle className="h-4 w-4 text-accent" />
            Başarısız Raw Eventler
          </DialogTitle>
          <DialogDescription className="text-muted-foreground">
            ClickHouse diagnostic failed raw event kayıtları. Terminal ignored eventler bu listede
            gösterilmez.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4 overflow-hidden">
          <p className="rounded-md border border-border bg-elevated px-3 py-2 text-xs text-muted-foreground">
            JetStream failed eventleri otomatik redelivery ile tekrar işler. Bu liste yalnızca
            ClickHouse üzerinde kalan diagnostic failed attempt kayıtlarını gösterir.
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
              Tümünü Temizle
            </Button>
            {actionMessage ? (
              <span
                className={`text-xs ${clearState === "error" ? "text-accent" : "text-primary"}`}
              >
                {actionMessage}
              </span>
            ) : null}
            <span className="ml-auto text-xs text-muted-foreground">{total} kayıt</span>
          </div>

          <div className="overflow-y-auto rounded-md border border-border">
            {isLoading ? (
              <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Yükleniyor...
              </div>
            ) : events.length === 0 ? (
              <div className="py-8 text-center text-sm text-muted-foreground">
                Başarısız event yok.
              </div>
            ) : (
              <table className="w-full text-xs">
                <thead className="sticky top-0 border-b border-border bg-kick-background">
                  <tr>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">Kanal</th>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">Hata</th>
                    <th className="px-3 py-2 text-center font-medium text-muted-foreground">
                      Deneme
                    </th>
                    <th className="px-3 py-2 text-right font-medium text-muted-foreground">
                      Son Hata
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
                      <td className="px-3 py-2 text-center text-muted-foreground">{ev.attempts}</td>
                      <td className="px-3 py-2 text-right text-muted-foreground">
                        {formatShortDate(ev.failed_at)}
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

function formatShortDate(value: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("tr-TR", {
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    month: "2-digit"
  }).format(new Date(value));
}
