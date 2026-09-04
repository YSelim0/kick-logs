"use client";

import { ChangeEvent, FormEvent, useMemo, useState } from "react";
import { FileUp, Loader2, TriangleAlert, Upload } from "lucide-react";

import { Button } from "@/components/ui/button";
import { confirmMessageImport, previewMessageImport } from "@/features/data-management/api";
import type { MessageImportPreview, MessageImportResult } from "@/types/api";

export function MessageImportPanel() {
  const [file, setFile] = useState<File | null>(null);
  const [limit, setLimit] = useState("20");
  const [preview, setPreview] = useState<MessageImportPreview | null>(null);
  const [confirmationText, setConfirmationText] = useState("");
  const [result, setResult] = useState<MessageImportResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [isConfirming, setIsConfirming] = useState(false);

  const parsedLimit = useMemo(() => {
    const trimmed = limit.trim();
    if (trimmed === "") return null;
    const value = Number(trimmed);
    return Number.isFinite(value) && value > 0 ? Math.floor(value) : null;
  }, [limit]);

  const canConfirm = Boolean(
    file &&
    preview?.can_execute &&
    confirmationText.trim() === preview?.confirmation_text &&
    !isConfirming
  );

  function selectFile(event: ChangeEvent<HTMLInputElement>) {
    setFile(event.target.files?.[0] ?? null);
    setPreview(null);
    setResult(null);
    setConfirmationText("");
    setError(null);
  }

  async function submitPreview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!file) {
      setError("Önce bir JSON export dosyası seçin.");
      return;
    }
    setIsPreviewing(true);
    setError(null);
    setResult(null);
    try {
      setPreview(await previewMessageImport(file, parsedLimit));
      setConfirmationText("");
    } catch {
      setError("Önizleme başarısız oldu.");
      setPreview(null);
    } finally {
      setIsPreviewing(false);
    }
  }

  async function submitConfirm() {
    if (!file || !preview) return;
    setIsConfirming(true);
    setError(null);
    try {
      setResult(await confirmMessageImport(file, parsedLimit, confirmationText.trim()));
      setPreview(null);
      setConfirmationText("");
    } catch {
      setError("İçe aktarma başarısız oldu.");
    } finally {
      setIsConfirming(false);
    }
  }

  return (
    <section className="rounded-lg border border-border bg-panel p-5">
      <div className="mb-1 flex items-center gap-2">
        <FileUp className="h-4 w-4 text-accent" />
        <h2 className="font-sans text-[15px] font-semibold text-foreground">Mesaj İçe Aktarma</h2>
      </div>
      <p className="mb-4 font-sans text-[12px] text-muted-foreground">
        JSON export dosyasından mesajları geri yükler. Kayıtlar{" "}
        <span className="font-mono text-foreground">kick_message_id</span> ile eşleşir; hâlihazırda
        var olan bir kayıt asla değiştirilmez, yalnızca atlanır.
      </p>

      {error ? (
        <div className="mb-4 rounded-md border border-danger bg-elevated px-3 py-2 text-[13px]">
          {error}
        </div>
      ) : null}

      <div className="flex flex-col gap-4">
        <form className="rounded-lg border border-border bg-elevated p-4" onSubmit={submitPreview}>
          <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_minmax(120px,160px)_auto]">
            <div className="flex flex-col gap-1.5">
              <label
                className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
                htmlFor="import-file"
              >
                EXPORT DOSYASI (JSON)
              </label>
              <input
                accept="application/json,.json"
                className="h-[38px] rounded-md border border-border-strong bg-panel px-3 py-1.5 font-sans text-[13px] text-foreground outline-none file:mr-3 file:rounded file:border-0 file:bg-elevated file:px-2 file:py-1 file:font-sans file:text-[12px] file:text-foreground focus:border-accent"
                id="import-file"
                onChange={selectFile}
                type="file"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <label
                className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
                htmlFor="import-limit"
              >
                LİMİT (BOŞ = TÜMÜ)
              </label>
              <input
                className="h-[38px] rounded-md border border-border-strong bg-panel px-3 font-sans text-[13px] text-foreground outline-none focus:border-accent placeholder:text-faint"
                id="import-limit"
                inputMode="numeric"
                onChange={(event) => setLimit(event.target.value)}
                placeholder="20"
                value={limit}
              />
            </div>

            <div className="flex items-end">
              <Button disabled={isPreviewing} type="submit" variant="outline">
                {isPreviewing ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <FileUp className="h-4 w-4 text-accent" />
                )}
                Dry-run
              </Button>
            </div>
          </div>
        </form>

        {preview ? (
          <div className="rounded-lg border border-warning bg-elevated p-4">
            <div className="mb-3 flex items-center gap-2">
              <TriangleAlert className="h-3.5 w-3.5 text-warning" />
              <span className="font-sans text-[13px] font-semibold text-foreground">
                İçe Aktarma Önizleme Sonucu
              </span>
            </div>
            <div className="mb-4 grid gap-3 md:grid-cols-4">
              <ImportStatCard label="EKLENECEK" value={formatNumber(preview.to_insert)} />
              <ImportStatCard label="ZATEN MEVCUT" value={formatNumber(preview.already_exists)} />
              <ImportStatCard
                label="DOSYA İÇİ TEKRAR"
                value={formatNumber(preview.duplicate_in_file)}
              />
              <ImportStatCard label="HATALI" value={formatNumber(preview.invalid)} />
            </div>
            <p className="mb-4 font-sans text-[12px] text-muted-foreground">
              Dosyadaki {formatNumber(preview.total_in_file)} kaydın{" "}
              {formatNumber(preview.records_read)} tanesi okundu.
            </p>

            {preview.invalid_reasons.length > 0 ? (
              <ul className="mb-4 flex flex-col gap-1">
                {preview.invalid_reasons.map((reason) => (
                  <li className="font-mono text-[11px] text-muted-foreground" key={reason.reason}>
                    {formatNumber(reason.count)} × {reason.reason} — {reason.example}
                  </li>
                ))}
              </ul>
            ) : null}

            {preview.reason ? (
              <p className="mb-4 font-sans text-[12px] text-muted-foreground">{preview.reason}</p>
            ) : null}

            {preview.can_execute ? (
              <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto]">
                <div className="flex flex-col gap-1.5">
                  <label
                    className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
                    htmlFor="import-confirm-text"
                  >
                    ONAY METNİ:{" "}
                    <span className="font-mono text-accent">{preview.confirmation_text}</span>
                  </label>
                  <input
                    className="h-[38px] rounded-md border border-border-strong bg-panel px-3 font-sans text-[13px] text-foreground outline-none focus:border-accent placeholder:text-faint"
                    id="import-confirm-text"
                    onChange={(event) => setConfirmationText(event.target.value)}
                    placeholder={preview.confirmation_text}
                    value={confirmationText}
                  />
                </div>
                <div className="flex items-end">
                  <Button disabled={!canConfirm} onClick={() => void submitConfirm()} type="button">
                    {isConfirming ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Upload className="h-4 w-4" />
                    )}
                    İçe aktar
                  </Button>
                </div>
              </div>
            ) : null}
          </div>
        ) : null}

        {result ? (
          <div className="rounded-lg border border-accent bg-elevated px-4 py-3 font-sans text-[13px] text-foreground">
            İçe aktarma tamamlandı: {formatNumber(result.written)} mesaj eklendi,{" "}
            {formatNumber(result.already_exists)} kayıt zaten mevcuttu ve değiştirilmedi.
          </div>
        ) : null}
      </div>
    </section>
  );
}

function ImportStatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border border-border bg-panel px-4 py-3">
      <span className="font-mono text-[10px] font-medium tracking-[0.8px] text-faint">{label}</span>
      <span className="truncate font-sans text-[18px] font-semibold text-foreground" title={value}>
        {value}
      </span>
    </div>
  );
}

function formatNumber(value: number) {
  return new Intl.NumberFormat("tr-TR").format(value);
}
