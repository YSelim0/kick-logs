"use client";

import {
  CalendarDays,
  Clock3,
  FileJson,
  FileText,
  Hash,
  MessageSquareReply,
  RotateCcw,
  Search,
  Smile,
  UserRound
} from "lucide-react";
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

      <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(280px,512px)_1fr]">
        <div>
          <div className="grid gap-4 sm:grid-cols-2">
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
          </div>

          <div className="mt-3 flex flex-wrap gap-2">
            {DATE_PRESETS.map((preset) => (
              <Button
                className="h-8 px-3 text-xs"
                key={preset.key}
                onClick={() => onDatePreset(preset.key)}
                type="button"
                variant="outline"
              >
                <Clock3 className="h-3.5 w-3.5 text-accent" />
                {preset.label}
              </Button>
            ))}
          </div>
        </div>

        <div className="grid gap-3">
          <div>
            <div className="mb-2 flex h-5 items-center gap-2 text-sm font-medium">
              <MessageSquareReply className="h-4 w-4 text-accent" />
              İçerik Tipi
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              <ToggleCheck
                checked={value.replyOnly}
                icon={<MessageSquareReply className="h-4 w-4" />}
                label="Yanıtlar"
                onChange={(checked) => onChange({ ...value, replyOnly: checked })}
              />
              <ToggleCheck
                checked={value.emoteOnly}
                icon={<Smile className="h-4 w-4" />}
                label="Emote içerenler"
                onChange={(checked) => onChange({ ...value, emoteOnly: checked })}
              />
            </div>
          </div>

          <div className="mb-2 flex h-5 items-center gap-2 text-sm font-medium">
            <Search className="h-4 w-4 text-accent" />
            İşlem
          </div>
          <div className="grid gap-3 sm:grid-cols-[96px_1fr_92px_92px]">
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
            <Button
              className="h-11"
              disabled={isLoading || !canExport}
              onClick={() => onExport("json")}
              type="button"
              variant="outline"
            >
              <FileJson className="h-4 w-4 text-accent" />
              JSON
            </Button>
            <Button
              className="h-11"
              disabled={isLoading || !canExport}
              onClick={() => onExport("csv")}
              type="button"
              variant="outline"
            >
              <FileText className="h-4 w-4 text-accent" />
              CSV
            </Button>
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
