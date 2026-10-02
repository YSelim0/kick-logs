import type { Locale } from "./locales";

export function createFormatters(locale: Locale, timeZone: string) {
  const numbers = new Intl.NumberFormat(locale);
  const compactNumbers = new Intl.NumberFormat(locale, {
    notation: "compact",
    maximumFractionDigits: 1
  });
  const dateOptions: Intl.DateTimeFormatOptions = {
    timeZone,
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23"
  };
  const dates = new Intl.DateTimeFormat(locale, dateOptions);
  const days = new Intl.DateTimeFormat(locale, { timeZone: "UTC", day: "numeric", month: "short" });

  function number(value: number, maximumFractionDigits?: number) {
    return maximumFractionDigits === undefined
      ? numbers.format(value)
      : new Intl.NumberFormat(locale, { maximumFractionDigits }).format(value);
  }

  return {
    number,
    compact: (value: number) => compactNumbers.format(value),
    shortDate(value: string | null) {
      if (!value || Number.isNaN(new Date(value).getTime())) return "—";
      return new Intl.DateTimeFormat(locale, {
        timeZone,
        day: "2-digit",
        month: "short",
        year: "numeric"
      }).format(new Date(value));
    },
    relativeTime(value: string | null, now = Date.now()) {
      if (!value || Number.isNaN(new Date(value).getTime())) return "—";
      const seconds = Math.trunc((new Date(value).getTime() - now) / 1000);
      const abs = Math.abs(seconds);
      const unit = abs < 60 ? "second" : abs < 3600 ? "minute" : abs < 86400 ? "hour" : "day";
      const divisor =
        unit === "second" ? 1 : unit === "minute" ? 60 : unit === "hour" ? 3600 : 86400;
      return new Intl.RelativeTimeFormat(locale).format(Math.trunc(seconds / divisor), unit);
    },
    percent: (value: number, maximumFractionDigits = 0) =>
      new Intl.NumberFormat(locale, { style: "percent", maximumFractionDigits }).format(value),
    dateTime(value: string | null | undefined, options?: Intl.DateTimeFormatOptions) {
      if (!value) return "—";
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) return "—";
      return options
        ? new Intl.DateTimeFormat(locale, { ...dateOptions, ...options }).format(date)
        : dates.format(date);
    },
    dayLabel(value: string) {
      const date = new Date(value);
      return Number.isNaN(date.getTime()) ? "—" : days.format(date);
    },
    bytes(value: number) {
      if (!Number.isFinite(value) || value <= 0) return "0 B";
      const units = ["B", "KB", "MB", "GB", "TB"];
      const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
      return `${number(value / 1024 ** index, index === 0 ? 0 : 2)} ${units[index]}`;
    }
  };
}
