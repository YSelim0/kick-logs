"use client";

import Image from "next/image";
import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { Check } from "lucide-react";
import { useEffect, useId, useRef, useState, type KeyboardEvent } from "react";
import { LANGUAGE_OPTIONS, type Locale } from "@/i18n/locales";
import { useLocalePreference } from "@/i18n/locale-provider";
import { cn } from "@/lib/utils";

export function LanguageSwitcher() {
  const t = useTranslations("common");
  const { locale, changeLocale, pending, failed } = useLocalePreference();
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const items = useRef<(HTMLButtonElement | null)[]>([]);
  const menuId = useId();
  const selected = LANGUAGE_OPTIONS.find((option) => option.locale === locale)!;

  useEffect(() => {
    if (!open) return;
    items.current[LANGUAGE_OPTIONS.findIndex((option) => option.locale === locale)]?.focus();
    const outside = (event: PointerEvent) => {
      if (event.target instanceof Node && !root.current?.contains(event.target)) setOpen(false);
    };
    document.addEventListener("pointerdown", outside);
    return () => document.removeEventListener("pointerdown", outside);
  }, [open, locale]);

  function close() {
    setOpen(false);
    trigger.current?.focus();
  }
  function choose(next: Locale) {
    close();
    if (next !== locale && !pending)
      void changeLocale(
        next,
        pathname === "/admin" || pathname.startsWith("/admin/") ? "admin" : "public"
      );
  }
  function onKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === "Escape") {
      event.preventDefault();
      close();
      return;
    }
    if (event.key === "Tab") {
      setOpen(false);
      return;
    }
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
    event.preventDefault();
    const current = items.current.findIndex((item) => item === document.activeElement);
    const index =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? LANGUAGE_OPTIONS.length - 1
          : (current + (event.key === "ArrowDown" ? 1 : -1) + LANGUAGE_OPTIONS.length) %
            LANGUAGE_OPTIONS.length;
    items.current[index]?.focus();
  }

  return (
    <div
      ref={root}
      className="fixed bottom-[max(1rem,env(safe-area-inset-bottom))] right-[max(1rem,env(safe-area-inset-right))] z-20"
    >
      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label={t("language.label")}
          onKeyDown={onKeyDown}
          className="absolute bottom-full right-0 mb-2 w-48 max-w-[calc(100vw-2rem)] rounded-lg border border-border-strong bg-panel p-2 shadow-lg"
        >
          {LANGUAGE_OPTIONS.map((option, index) => (
            <button
              key={option.locale}
              ref={(element) => {
                items.current[index] = element;
              }}
              role="menuitemradio"
              aria-checked={locale === option.locale}
              tabIndex={-1}
              type="button"
              disabled={pending}
              onClick={() => choose(option.locale)}
              className={cn(
                "flex min-h-11 w-full items-center gap-3 rounded-md px-3 text-left text-sm transition-colors hover:bg-elevated focus-visible:bg-elevated focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent motion-reduce:transition-none",
                locale === option.locale && "bg-elevated"
              )}
            >
              <Image
                src={option.flag}
                width={24}
                height={24}
                alt=""
                className="h-6 w-6 shrink-0 rounded-sm object-cover"
              />
              <span className="flex-1">{option.name}</span>
              {locale === option.locale && (
                <Check className="h-3.5 w-3.5 text-accent" aria-hidden />
              )}
            </button>
          ))}
        </div>
      )}
      {failed && !open && (
        <p
          role="status"
          className="absolute bottom-full right-0 mb-2 w-56 rounded-lg border border-border bg-panel p-3 text-xs text-foreground"
        >
          {t("errors.languageSwitch")}
        </p>
      )}
      <button
        ref={trigger}
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label={`${t("language.choose")} (${selected.name})`}
        title={`${t("language.choose")} (${selected.name})`}
        disabled={pending}
        onClick={() => setOpen((value) => !value)}
        onKeyDown={(event) => {
          if (event.key === "ArrowDown" || event.key === "ArrowUp") {
            event.preventDefault();
            setOpen(true);
          }
        }}
        className="inline-flex h-12 w-12 items-center justify-center rounded-md border border-border-strong bg-panel transition-colors hover:bg-elevated focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent disabled:cursor-wait disabled:opacity-60 motion-reduce:transition-none"
      >
        <Image
          src={selected.flag}
          width={28}
          height={28}
          alt=""
          className="h-7 w-7 rounded-sm object-cover"
        />
      </button>
    </div>
  );
}
