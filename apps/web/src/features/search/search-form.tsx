"use client";

import { CalendarDays, Download, FileJson, FileText } from "lucide-react";
import { useTranslations } from "next-intl";
import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DATE_PRESETS,
  type DatePresetKey,
  type SearchFormState
} from "@/features/search/search-params";
import { cn } from "@/lib/utils";
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
  const t = useTranslations("search");
  const actions = useTranslations("common.actions");
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
      className="flex flex-col gap-3.5 rounded-lg border border-border bg-panel p-5"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
    >
      <div className="grid gap-3 md:grid-cols-3">
        <Field id="sender" label={t("sender")}>
          <Input
            id="sender"
            maxLength={160}
            onChange={(event) => onChange({ ...value, sender: event.target.value })}
            placeholder={t("senderPlaceholder")}
            value={value.sender}
          />
        </Field>
        <Field id="channel" label={t("channel")}>
          <Input
            id="channel"
            maxLength={160}
            onChange={(event) => onChange({ ...value, channel: event.target.value })}
            placeholder={t("channelPlaceholder")}
            value={value.channel}
          />
        </Field>
        <Field id="q" label={t("content")}>
          <Input
            id="q"
            maxLength={500}
            onChange={(event) => onChange({ ...value, q: event.target.value })}
            placeholder={t("contentPlaceholder")}
            value={value.q}
          />
        </Field>
      </div>

      <div className="grid gap-3 md:grid-cols-3">
        <Field id="start" label={t("start")} icon={<CalendarDays className="h-3.5 w-3.5" />}>
          <Input
            id="start"
            onChange={(event) => onChange({ ...value, start: event.target.value })}
            type="datetime-local"
            value={value.start}
          />
        </Field>
        <Field id="end" label={t("end")} icon={<CalendarDays className="h-3.5 w-3.5" />}>
          <Input
            id="end"
            onChange={(event) => onChange({ ...value, end: event.target.value })}
            type="datetime-local"
            value={value.end}
          />
        </Field>
        <Field id="datePreset" label={t("quickRange")}>
          <select
            className="flex h-10 w-full rounded-md border border-border bg-elevated px-3 py-2 text-[13px] text-foreground outline-none transition-colors focus:border-border-strong"
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
            <option value="">{t("chooseRange")}</option>
            {DATE_PRESETS.map((preset) => (
              <option key={preset.key} value={preset.key}>
                {t(`presets.${preset.key}`)}
              </option>
            ))}
          </select>
        </Field>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <TogglePill
          checked={value.replyOnly}
          label={t("replyOnly")}
          onChange={(checked) => onChange({ ...value, replyOnly: checked })}
        />
        <TogglePill
          checked={value.emoteOnly}
          label={t("emoteOnly")}
          onChange={(checked) => onChange({ ...value, emoteOnly: checked })}
        />
        <div className="flex flex-1 justify-end gap-2">
          <div className="relative" ref={exportMenuRef}>
            <button
              aria-expanded={isExportMenuOpen}
              aria-label={t("export")}
              className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border bg-elevated text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
              disabled={isLoading || !canExport}
              onClick={() => setIsExportMenuOpen((current) => !current)}
              title={t("export")}
              type="button"
            >
              <Download className="h-4 w-4" />
            </button>

            {isExportMenuOpen ? (
              <div className="absolute right-0 z-20 mt-2 grid min-w-[160px] gap-1 rounded-md border border-border bg-panel p-1.5 shadow-lg">
                <button
                  className="inline-flex h-8 items-center gap-2 rounded-sm px-2 text-[13px] text-foreground hover:bg-elevated"
                  onClick={() => handleExport("json")}
                  type="button"
                >
                  <FileJson className="h-4 w-4 text-muted-foreground" />
                  {t("downloadFormat", { format: "JSON" })}
                </button>
                <button
                  className="inline-flex h-8 items-center gap-2 rounded-sm px-2 text-[13px] text-foreground hover:bg-elevated"
                  onClick={() => handleExport("csv")}
                  type="button"
                >
                  <FileText className="h-4 w-4 text-muted-foreground" />
                  {t("downloadFormat", { format: "CSV" })}
                </button>
              </div>
            ) : null}
          </div>
          <Button
            className="h-9 border-border-strong"
            disabled={isLoading}
            onClick={onReset}
            size="sm"
            type="button"
            variant="outline"
          >
            {actions("reset")}
          </Button>
          <Button
            className="h-9 bg-accent px-4 text-accent-foreground hover:bg-accent-hover"
            disabled={isLoading}
            size="sm"
            type="submit"
          >
            {actions("search")}
          </Button>
        </div>
      </div>
    </form>
  );
}

function TogglePill({
  checked,
  label,
  onChange
}: {
  checked: boolean;
  label: string;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label
      className={cn(
        "inline-flex h-9 cursor-pointer items-center gap-2 rounded-md px-3 text-[13px] text-foreground transition-colors",
        checked
          ? "border border-accent bg-elevated"
          : "border border-border bg-transparent hover:border-border-strong"
      )}
    >
      <input
        checked={checked}
        className={cn(
          "h-3.5 w-3.5 cursor-pointer appearance-none rounded-sm border transition-colors",
          checked ? "border-accent bg-accent" : "border-border-strong bg-transparent"
        )}
        onChange={(event) => onChange(event.target.checked)}
        type="checkbox"
      />
      {label}
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
  icon?: ReactNode;
  id: string;
  label: string;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <label
        className="flex items-center gap-1.5 font-mono text-2xs uppercase text-muted-foreground"
        htmlFor={id}
      >
        {icon ? <span className="text-faint">{icon}</span> : null}
        {label}
      </label>
      {children}
    </div>
  );
}
