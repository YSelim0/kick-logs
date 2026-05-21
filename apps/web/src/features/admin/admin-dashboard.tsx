"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { Activity, Database, LogOut, Radio, Settings, Users } from "lucide-react";

import { Button } from "@/components/ui/button";
import { logout } from "@/features/auth/api";
import { useCurrentUser } from "@/features/auth/use-auth";
import { ChannelAdmin } from "@/features/channels/channel-admin";
import { DataManagementPanel } from "@/features/data-management/data-management-panel";
import { OperationsDashboard } from "@/features/operations/operations-dashboard";
import { UserAdmin } from "@/features/users/user-admin";

export function AdminDashboard() {
  const router = useRouter();
  const { error, status, user } = useCurrentUser();
  const [logoutError, setLogoutError] = useState<string | null>(null);
  const [isLoggingOut, setIsLoggingOut] = useState(false);

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
    return <AdminState message="Oturum kontrol ediliyor..." />;
  }

  if (status === "error" || !user) {
    return <AdminState message={error ?? "Oturum bilgisi alınamadı."} />;
  }

  const isSuperAdmin = user.role === "super_admin";

  return (
    <div className="flex min-h-screen flex-col bg-page text-foreground">
      <header className="flex h-14 shrink-0 items-center justify-between border-b border-border px-6">
        <div className="flex items-center gap-6">
          <Link className="flex items-center gap-2" href="/">
            <div aria-hidden="true" className="h-6 w-6 rounded-md bg-accent" />
            <span className="font-sans text-[15px] font-semibold">kick logs</span>
          </Link>
          <div className="flex items-center gap-2 font-mono text-[13px]">
            <span className="text-faint">/</span>
            <span className="font-medium text-foreground">admin</span>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <span className="font-mono text-[12px] text-muted-foreground">{user.email}</span>
          {isSuperAdmin && (
            <span className="rounded border border-border bg-panel px-2 py-0.5 font-mono text-[10px] font-semibold tracking-widest text-accent">
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
            Çıkış
          </Button>
        </div>
      </header>

      <div className="flex flex-1 gap-6 p-6">
        <aside className="flex w-[220px] shrink-0 flex-col gap-1 pt-2">
          <SidebarItem icon={<Activity className="h-3.5 w-3.5" />} label="Operations" active />
          <SidebarItem icon={<Radio className="h-3.5 w-3.5" />} label="Channels" />
          {isSuperAdmin && <SidebarItem icon={<Users className="h-3.5 w-3.5" />} label="Users" />}
          <SidebarItem icon={<Database className="h-3.5 w-3.5" />} label="Data" />
          <SidebarItem icon={<Settings className="h-3.5 w-3.5" />} label="Settings" />
        </aside>

        <main className="flex min-w-0 flex-1 flex-col gap-6">
          {logoutError ? (
            <div className="rounded-md border border-danger bg-elevated px-4 py-3 text-sm">
              {logoutError}
            </div>
          ) : null}
          <OperationsDashboard />
          <ChannelAdmin />
          {isSuperAdmin ? <UserAdmin /> : null}
          <DataManagementPanel />
        </main>
      </div>
    </div>
  );
}

function SidebarItem({
  active = false,
  icon,
  label
}: {
  active?: boolean;
  icon: React.ReactNode;
  label: string;
}) {
  return (
    <div
      className={`flex cursor-pointer items-center gap-2.5 rounded-md px-3 py-2 font-sans text-[13px] transition-colors ${
        active
          ? "border border-border bg-panel font-medium text-foreground"
          : "text-muted-foreground hover:bg-panel hover:text-foreground"
      }`}
    >
      <span className={active ? "text-accent" : "text-muted-foreground"}>{icon}</span>
      {label}
    </div>
  );
}

function AdminState({ message }: { message: string }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-page">
      <div className="rounded-lg border border-border bg-panel px-6 py-4 text-[13px] text-muted-foreground">
        {message}
      </div>
    </div>
  );
}
