"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { useState } from "react";
import { Activity, Database, Inbox, LogOut, Menu, Radio, Settings, Users, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { logout } from "@/features/auth/api";
import { useCurrentUser } from "@/features/auth/use-auth";

const NAV_ITEMS = [
  { label: "Operations", href: "/admin/operations", icon: Activity, superAdminOnly: false },
  { label: "Channels", href: "/admin/channels", icon: Radio, superAdminOnly: false },
  { label: "Requests", href: "/admin/requests", icon: Inbox, superAdminOnly: false },
  { label: "Users", href: "/admin/users", icon: Users, superAdminOnly: true },
  { label: "Data", href: "/admin/data", icon: Database, superAdminOnly: false },
  { label: "Settings", href: null, icon: Settings, superAdminOnly: false }
] as const;

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { error, status, user } = useCurrentUser();
  const [logoutError, setLogoutError] = useState<string | null>(null);
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  useEffect(() => {
    if (status === "unauthenticated") {
      router.replace("/login?next=/admin");
    }
  }, [router, status]);

  async function submitLogout() {
    setIsLoggingOut(true);
    setLogoutError(null);
    try {
      await logout();
      router.replace("/login");
    } catch (caught) {
      setLogoutError(caught instanceof Error ? caught.message : "Çıkış yapılamadı.");
      setIsLoggingOut(false);
    }
  }

  if (status === "loading" || status === "unauthenticated") {
    return (
      <div className="flex min-h-screen items-center justify-center bg-page">
        <span className="text-[13px] text-muted-foreground">Oturum kontrol ediliyor...</span>
      </div>
    );
  }

  if (status === "error" || !user) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-page">
        <span className="text-[13px] text-muted-foreground">
          {error ?? "Oturum bilgisi alınamadı."}
        </span>
      </div>
    );
  }

  const isSuperAdmin = user.role === "super_admin";

  return (
    <div className="flex min-h-screen flex-col bg-page text-foreground">
      {/* Header */}
      <header className="sticky top-0 z-20 flex h-14 shrink-0 items-center justify-between border-b border-border bg-page px-4 md:px-6">
        <div className="flex items-center gap-3 md:gap-6">
          <button
            className="flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:bg-panel hover:text-foreground md:hidden"
            onClick={() => setIsSidebarOpen(true)}
            type="button"
            aria-label="Menüyü aç"
          >
            <Menu className="h-4 w-4" />
          </button>
          <Link className="flex items-center gap-2" href="/">
            <Image
              alt="Kick Logs"
              className="rounded-md object-contain"
              height={24}
              priority
              src="/app-logo.png"
              width={24}
            />
            <span className="font-sans text-[15px] font-semibold">kick logs</span>
          </Link>
          <div className="hidden items-center gap-2 font-mono text-[13px] md:flex">
            <span className="text-faint">/</span>
            <span className="font-medium text-foreground">admin</span>
          </div>
        </div>

        <div className="flex items-center gap-2 md:gap-3">
          <span className="hidden font-mono text-[12px] text-muted-foreground sm:block">
            {user.email}
          </span>
          {isSuperAdmin && (
            <span className="hidden rounded border border-border bg-panel px-2 py-0.5 font-mono text-[10px] font-semibold tracking-widest text-accent sm:inline">
              SUPER ADMIN
            </span>
          )}
          <Button
            disabled={isLoggingOut}
            onClick={() => void submitLogout()}
            size="sm"
            type="button"
            variant="outline"
          >
            <LogOut className="h-4 w-4" />
            <span className="hidden sm:inline">Çıkış</span>
          </Button>
        </div>
      </header>

      {/* Body */}
      <div className="flex flex-1">
        {/* Mobile sidebar backdrop */}
        {isSidebarOpen && (
          <div
            className="fixed inset-0 z-30 bg-black/50 md:hidden"
            onClick={() => setIsSidebarOpen(false)}
          />
        )}

        {/* Sidebar */}
        <aside
          className={`fixed inset-y-0 left-0 z-40 flex w-[220px] shrink-0 flex-col overflow-y-auto border-r border-border bg-page px-3 py-4 pt-16 transition-transform md:sticky md:top-14 md:h-[calc(100vh-56px)] md:translate-x-0 md:pt-4 ${
            isSidebarOpen ? "translate-x-0" : "-translate-x-full"
          }`}
        >
          <button
            className="absolute right-3 top-3 flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-elevated hover:text-foreground md:hidden"
            onClick={() => setIsSidebarOpen(false)}
            type="button"
            aria-label="Menüyü kapat"
          >
            <X className="h-4 w-4" />
          </button>
          <nav className="flex flex-col gap-1">
            {NAV_ITEMS.map((item) => {
              if (item.superAdminOnly && !isSuperAdmin) return null;

              const Icon = item.icon;
              const isActive = item.href !== null && pathname.startsWith(item.href);
              const isDisabled = item.href === null;

              if (isDisabled) {
                return (
                  <div
                    key={item.label}
                    className="flex cursor-not-allowed items-center gap-2.5 rounded-md px-3 py-2 font-sans text-[13px] text-faint opacity-50"
                  >
                    <Icon className="h-3.5 w-3.5" />
                    {item.label}
                  </div>
                );
              }

              return (
                <Link
                  key={item.label}
                  className={`flex items-center gap-2.5 rounded-md px-3 py-2 font-sans text-[13px] transition-colors ${
                    isActive
                      ? "border border-border bg-panel font-medium text-foreground"
                      : "text-muted-foreground hover:bg-panel hover:text-foreground"
                  }`}
                  href={item.href}
                  onClick={() => setIsSidebarOpen(false)}
                >
                  <Icon
                    className={`h-3.5 w-3.5 ${isActive ? "text-accent" : "text-muted-foreground"}`}
                  />
                  {item.label}
                </Link>
              );
            })}
          </nav>
        </aside>

        {/* Main content */}
        <main className="flex min-w-0 flex-1 flex-col gap-6 p-4 md:p-6">
          {logoutError ? (
            <div className="rounded-md border border-danger bg-elevated px-4 py-3 text-sm">
              {logoutError}
            </div>
          ) : null}
          {children}
        </main>
      </div>
    </div>
  );
}
