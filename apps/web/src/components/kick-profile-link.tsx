import Image from "next/image";

import { Button } from "@/components/ui/button";

type KickProfileLinkProps = {
  href: string | null;
};

export function KickProfileLink({ href }: KickProfileLinkProps) {
  if (!href) {
    return null;
  }

  return (
    <Button asChild className="w-full sm:w-auto" variant="outline">
      <a href={href} rel="noopener noreferrer" target="_blank">
        <Image alt="" aria-hidden className="h-4 w-4" height={16} src="/kick-logo.png" width={16} />
        Kick hesabını ziyaret
      </a>
    </Button>
  );
}
