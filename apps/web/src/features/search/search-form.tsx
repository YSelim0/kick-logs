"use client";

import {
  CalendarDays,
  Clock3,
  Download,
  FileJson,
  FileText,
  Hash,
  MessageSquareReply,
  RotateCcw,
  Search,
  Smile,
  UserRound
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DATE_PRESETS,
  type DatePresetKey,
  type SearchFormState
} from "@/features/search/search-params";
import type { MessageExportFormat } from "@/types/api";

type SearchFormProps = {
  value: SearchFormState;
  isLoading: boolean;
  canExport: boolean;
  onChange: (value: SearchFormState) => void;
  onDatePreset: (preset: DatePresetKey) => void;
  onExport: (format: MessageExportFormat) => void;
  onReset: () => void;
  onSubmit: () => void;
};

export function SearchForm({
  value,
  isLoading,
  canExport,
  onChange,
  onDatePreset,
  onExport,
  onReset,
  onSubmit
}: SearchFormProps) {
  const [isExportMenuOpen, setIsExportMenuOpen] = useState(false);
  const exportMenuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isExportMenuOpen) {
      return;
    }

    function closeOnOutsideClick(event: MouseEvent | TouchEvent) {
      const target = event.target;

      if (target instanceof Node && !exportMenuRef.current?.contains(target)) {
        setIsExportMenuOpen(false);
      }
    }

    document.addEventListener("mousedown", closeOnOutsideClick);
    document.addEventListener("touchstart", closeOnOutsideClick);

    return () => {
      document.removeEventListener("mousedown", closeOnOutsideClick);
      document.removeEventListener("touchstart", closeOnOutsideClick);
    };
  }, [isExportMenuOpen]);

  function handleExport(format: MessageExportFormat) {
    setIsExportMenuOpen(false);
    onExport(format);
  }

  return (
    <form
      className="rounded-lg border border-border bg-black p-4"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
    >
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-md bg-kick-background text-primary">
            <Search className="h-4 w-4" />
          </div>
          <div>
            <h2 className="text-base font-semibold">Arama Filtreleri</h2>
            <p className="text-xs text-muted-foreground">
              Gönderen, kanal, kelime ve tarih aralığı
            </p>
          </div>
        </div>
        <div className="rounded-md bg-kick-background px-3 py-1.5 text-xs text-primary">
          7 filtre
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-[minmax(280px,390px)_1fr]">
        <Field icon={<Search className="h-4 w-4" />} id="q" label="Aramak istediğiniz Kelime">
          <Input
            id="q"
            maxLength={500}
            onChange={(event) => onChange({ ...value, q: event.target.value })}
            placeholder="örn. selam, KEKW, duyuru"
            value={value.q}
          />
        </Field>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field icon={<UserRound className="h-4 w-4" />} id="sender" label="Kullanıcı Adı">
            <Input
              id="sender"
              maxLength={160}
              onChange={(event) => onChange({ ...value, sender: event.target.value })}
              placeholder="tüm kullanıcılar"
              value={value.sender}
            />
          </Field>

          <Field icon={<Hash className="h-4 w-4" />} id="channel" label="Kanal Adı">
            <Input
              id="channel"
              maxLength={160}
              onChange={(event) => onChange({ ...value, channel: event.target.value })}
              placeholder="tüm kanallar"
              value={value.channel}
            />
          </Field>
        </div>
      </div>

      <div className="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_160px]">
        <Field icon={<CalendarDays className="h-4 w-4" />} id="start" label="Başlangıç">
          <Input
            id="start"
            onChange={(event) => onChange({ ...value, start: event.target.value })}
            type="datetime-local"
            value={value.start}
          />
        </Field>

        <Field icon={<CalendarDays className="h-4 w-4" />} id="end" label="Bitiş">
          <Input
            id="end"
            onChange={(event) => onChange({ ...value, end: event.target.value })}
            type="datetime-local"
            value={value.end}
          />
        </Field>

        <Field icon={<Clock3 className="h-4 w-4" />} id="datePreset" label="Hızlı aralık">
          <select
            className="flex h-11 w-full rounded-md border border-border bg-kick-background px-3 py-2 text-sm text-foreground outline-none transition-colors focus:border-primary focus:ring-2 focus:ring-primary/25"
            defaultValue=""
            id="datePreset"
            onChange={(event) => {
              if (!event.target.value) {
                return;
              }

              onDatePreset(event.target.value as DatePresetKey);
              event.target.value = "";
            }}
          >
            <option value="">Seç</option>
            {DATE_PRESETS.map((preset) => (
              <option key={preset.key} value={preset.key}>
                Son {preset.label}
              </option>
            ))}
          </select>
        </Field>
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(280px,1fr)_minmax(280px,420px)]">
        <div>
          <div className="mb-2 flex h-5 items-center gap-2 text-sm font-medium">
            <MessageSquareReply className="h-4 w-4 text-accent" />
            Sonuç Türü
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <ToggleCheck
              checked={value.replyOnly}
              icon={<MessageSquareReply className="h-4 w-4" />}
              label="Sadece yanıtlar"
              onChange={(checked) => onChange({ ...value, replyOnly: checked })}
            />
            <ToggleCheck
              checked={value.emoteOnly}
              icon={<Smile className="h-4 w-4" />}
              label="Sadece emote"
              onChange={(checked) => onChange({ ...value, emoteOnly: checked })}
            />
          </div>
        </div>

        <div>
          <div className="mb-2 flex h-5 items-center gap-2 text-sm font-medium">
            <Search className="h-4 w-4 text-accent" />
            İşlem
          </div>
          <div className="grid gap-3 sm:grid-cols-[96px_1fr_44px]">
            <Button
              className="h-11"
              disabled={isLoading}
              onClick={onReset}
              type="button"
              variant="outline"
            >
              <RotateCcw className="h-4 w-4 text-accent" />
              Sıfırla
            </Button>
            <Button className="h-11" disabled={isLoading} type="submit">
              <Search className="h-4 w-4" />
              Ara
            </Button>
            <div className="relative" ref={exportMenuRef}>
              <Button
                aria-expanded={isExportMenuOpen}
                aria-label="Dışa aktar"
                className="h-11 w-full px-0"
                disabled={isLoading || !canExport}
                onClick={() => setIsExportMenuOpen((current) => !current)}
                title="Dışa aktar"
                type="button"
                variant="outline"
              >
                <Download className="h-4 w-4 text-accent" />
              </Button>

              {isExportMenuOpen ? (
                <div className="absolute right-0 z-20 mt-2 grid min-w-[150px] gap-2 rounded-md border border-border bg-black p-2 shadow-xl">
                  <Button
                    className="h-9 justify-start px-3 text-xs"
                    onClick={() => handleExport("json")}
                    type="button"
                    variant="ghost"
                  >
                    <FileJson className="h-4 w-4 text-accent" />
                    JSON indir
                  </Button>
                  <Button
                    className="h-9 justify-start px-3 text-xs"
                    onClick={() => handleExport("csv")}
                    type="button"
                    variant="ghost"
                  >
                    <FileText className="h-4 w-4 text-accent" />
                    CSV indir
                  </Button>
                </div>
              ) : null}
            </div>
          </div>
        </div>
      </div>
    </form>
  );
}

function ToggleCheck({
  checked,
  icon,
  label,
  onChange
}: {
  checked: boolean;
  icon: ReactNode;
  label: string;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex h-11 cursor-pointer items-center gap-3 rounded-md border border-border bg-kick-background px-3 text-sm text-foreground transition-colors hover:border-primary">
      <input
        checked={checked}
        className="h-4 w-4 accent-primary"
        onChange={(event) => onChange(event.target.checked)}
        type="checkbox"
      />
      <span className="text-accent">{icon}</span>
      <span className="truncate">{label}</span>
    </label>
  );
}

function Field({
  children,
  icon,
  id,
  label
}: {
  children: ReactNode;
  icon: ReactNode;
  id: string;
  label: string;
}) {
  return (
    <div>
      <label className="mb-2 flex h-5 items-center gap-2 text-sm font-medium" htmlFor={id}>
        <span className="text-accent">{icon}</span>
        {label}
      </label>
      {children}
    </div>
  );
}
