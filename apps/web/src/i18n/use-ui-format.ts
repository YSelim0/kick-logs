"use client";
import { useMemo } from "react";
import { createFormatters } from "./format";
import { useLocalePreference } from "./locale-provider";

export function useUiFormat() {
  const { locale, timeZone } = useLocalePreference();
  return useMemo(() => createFormatters(locale, timeZone), [locale, timeZone]);
}
