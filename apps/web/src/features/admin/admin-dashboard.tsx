"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { LogOut, ShieldCheck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { logout } from "@/features/auth/api";
import { useCurrentUser } from "@/features/auth/use-auth";

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

  return (
    <main className="min-h-screen bg-background px-4 py-4 text-foreground md:px-8 md:py-6">
      <div className="mx-auto flex max-w-[1440px] flex-col gap-6">
        <header className="flex flex-wrap items-center justify-between gap-4 border-b border-border bg-black px-4 py-4 md:px-6">
          <Link className="flex min-w-0 items-center gap-4" href="/admin">
            <Image
              alt="Kick Logs"
              className="h-11 w-11 shrink-0 rounded-md object-contain"
              height={44}
              priority
              src="/app-logo.png"
              width={44}
            />
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h1 className="text-lg font-semibold">Kick Logs</h1>
                <span className="rounded-md bg-kick-background px-2 py-1 text-xs text-primary">
                  /admin
                </span>
              </div>
              <p className="text-xs text-muted-foreground">Backend yönetim paneli</p>
            </div>
          </Link>

          <div className="flex flex-wrap items-center gap-3">
            <div className="rounded-md border border-border bg-kick-background px-3 py-2 text-xs">
              <div className="text-muted-foreground">Oturum</div>
              <div className="font-medium text-primary">{user.email}</div>
            </div>
            <Button asChild variant="outline">
              <Link href="/search">Search</Link>
            </Button>
            <Button
              disabled={isLoggingOut}
              onClick={() => void submitLogout()}
              type="button"
              variant="outline"
            >
              <LogOut className="h-4 w-4 text-accent" />
              Çıkış
            </Button>
          </div>
        </header>

        {logoutError ? (
          <div className="rounded-md border border-accent bg-black px-4 py-3 text-sm">
            {logoutError}
          </div>
        ) : null}

        <section className="rounded-lg border border-border bg-black p-5">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-md bg-kick-background text-primary">
              <ShieldCheck className="h-4 w-4" />
            </div>
            <div>
              <h2 className="text-base font-semibold">Admin oturumu aktif</h2>
              <p className="text-xs text-muted-foreground">
                Kanal ve kullanıcı yönetimi bu panelde tamamlanacak.
              </p>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}

function AdminState({ message }: { message: string }) {
  return (
    <main className="min-h-screen bg-background px-6 py-6 text-foreground">
      <div className="mx-auto max-w-[1440px] rounded-lg border border-border bg-black p-6 text-sm text-muted-foreground">
        {message}
      </div>
    </main>
  );
}
