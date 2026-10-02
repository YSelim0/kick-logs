"use client";
import Image from "next/image";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";

type KickProfileLinkProps = {
  href: string | null;
};

export function KickProfileLink({ href }: KickProfileLinkProps) {
  const t = useTranslations("common.profile");
  if (!href) {
    return null;
  }

  return (
    <Button asChild className="w-full sm:w-auto" variant="outline">
      <a href={href} rel="noopener noreferrer" target="_blank">
        <Image alt="" aria-hidden className="h-4 w-4" height={16} src="/kick-logo.png" width={16} />
        {t("visitKick")}
      </a>
    </Button>
  );
}
