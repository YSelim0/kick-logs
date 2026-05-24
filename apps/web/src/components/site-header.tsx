import Image from "next/image";
import Link from "next/link";
import { Github } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type ActiveRoute = "search" | "channels" | "users" | "admin" | null;

type SiteHeaderProps = {
  activeRoute?: ActiveRoute;
};

export function SiteHeader({ activeRoute = "search" }: SiteHeaderProps) {
  return (
    <header className="h-14 border-b border-border bg-page">
      <div className="mx-auto flex h-full max-w-[1280px] items-center justify-between gap-4 px-6">
        <div className="flex items-center gap-4">
          <Link className="flex items-center gap-2" href="/">
            <Image
              alt="Kick Logs"
              className="h-6 w-6 rounded-md object-contain"
              height={24}
              priority
              src="/app-logo.png"
              width={24}
            />
            <span className="text-[15px] font-semibold tracking-tight">kick logs</span>
          </Link>
          <NavLinks activeRoute={activeRoute} />
        </div>

        <div className="flex items-center gap-2">
          <a
            aria-label="GitHub"
            className="inline-flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-elevated hover:text-foreground"
            href="https://github.com/YSelim0/kick-logs"
            rel="noopener noreferrer"
            target="_blank"
          >
            <Github className="h-4 w-4" />
          </a>
          <Button asChild variant="outline" size="sm">
            <Link href="/admin">Admin</Link>
          </Button>
        </div>
      </div>
    </header>
  );
}

type NavItem = { route: NonNullable<ActiveRoute>; label: string; href: string };

const NAV_ITEMS: NavItem[] = [
  { route: "search", label: "Search", href: "/search" },
  { route: "channels", label: "Channels", href: "/channels" },
  { route: "users", label: "Users", href: "/users" }
];

function NavLinks({ activeRoute }: { activeRoute: ActiveRoute }) {
  return (
    <nav aria-label="Main navigation" className="flex items-center gap-1">
      {NAV_ITEMS.map(({ route, label, href }) => (
        <Link
          key={route}
          className={cn(
            "inline-flex h-7 items-center rounded-md px-3 text-[13px] font-medium transition-colors",
            activeRoute === route
              ? "border border-border bg-panel text-foreground"
              : "text-muted-foreground hover:bg-elevated hover:text-foreground"
          )}
          href={href}
        >
          {label}
        </Link>
      ))}
    </nav>
  );
}

