"use client";

import { usePathname } from "next/navigation";
import { useTranslations } from "next-intl";
import { useEffect } from "react";
import { localizeRouteMetadata } from "./route-metadata";

export function PageMetadata() {
  const pathname = usePathname();
  const t = useTranslations("common.metadata");
  const { title, description } = localizeRouteMetadata(pathname, t);

  useEffect(() => {
    document.title = title;
    document.querySelector('meta[name="description"]')?.setAttribute("content", description);
  }, [pathname, title, description]);
  return null;
}
