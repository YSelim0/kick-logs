"use client";

import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { useEffect } from "react";

export function PageMetadata() {
  const pathname = usePathname();
  const t = useTranslations("common.metadata");
  const page =
    pathname === "/channels"
      ? "channels"
      : pathname === "/users"
        ? "users"
        : pathname === "/prediction"
          ? "prediction"
          : null;
  const title = page ? t(`${page}.title`) : "Kick Logs";
  const description = page ? t(`${page}.description`) : t("description");

  useEffect(() => {
    document.title = title;
    document.querySelector('meta[name="description"]')?.setAttribute("content", description);
  }, [pathname, title, description]);
  return null;
}
