"use client";

import Image from "next/image";
import Link from "next/link";
import { Github, Menu, X } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type ActiveRoute = "search" | "channels" | "users" | "prediction" | "admin" | null;

type SiteHeaderProps = {
  activeRoute?: ActiveRoute;
};

export function SiteHeader({ activeRoute = "search" }: SiteHeaderProps) {
  const [isMenuOpen, setIsMenuOpen] = useState(false);

  return (
    <>
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
            <div className="hidden md:flex">
              <NavLinks activeRoute={activeRoute} />
            </div>
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
            <Button asChild variant="outline" size="sm" className="hidden md:inline-flex">
              <Link href="/admin">Admin</Link>
            </Button>
            <button
              aria-label={isMenuOpen ? "Menüyü kapat" : "Menüyü aç"}
              className="inline-flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-elevated hover:text-foreground md:hidden"
              onClick={() => setIsMenuOpen((v) => !v)}
              type="button"
            >
              {isMenuOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
            </button>
          </div>
        </div>
      </header>

      {isMenuOpen && (
        <>
          <div
            className="fixed inset-0 z-30 bg-black/50 md:hidden"
            onClick={() => setIsMenuOpen(false)}
          />
          <div className="fixed left-0 right-0 top-14 z-40 border-b border-border bg-page px-6 py-3 md:hidden">
            <nav className="flex flex-col gap-1">
              {NAV_ITEMS.map(({ route, label, href }) => (
                <Link
                  key={route}
                  href={href}
                  onClick={() => setIsMenuOpen(false)}
                  className={cn(
                    "flex h-9 items-center rounded-md px-3 text-[13px] font-medium transition-colors",
                    activeRoute === route
                      ? "border border-border bg-panel text-foreground"
                      : "text-muted-foreground hover:bg-elevated hover:text-foreground"
                  )}
                >
                  {label}
                </Link>
              ))}
              <div className="mt-1 border-t border-border pt-1">
                <Link
                  href="/admin"
                  onClick={() => setIsMenuOpen(false)}
                  className="flex h-9 items-center rounded-md px-3 text-[13px] font-medium text-muted-foreground transition-colors hover:bg-elevated hover:text-foreground"
                >
                  Admin
                </Link>
              </div>
            </nav>
          </div>
        </>
      )}
    </>
  );
}

type NavItem = { route: NonNullable<ActiveRoute>; label: string; href: string };

const NAV_ITEMS: NavItem[] = [
  { route: "search", label: "Search", href: "/search" },
  { route: "channels", label: "Channels", href: "/channels" },
  { route: "users", label: "Users", href: "/users" },
  { route: "prediction", label: "Prediction", href: "/prediction" }
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
