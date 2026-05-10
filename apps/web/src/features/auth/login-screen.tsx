"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useMemo, useState } from "react";
import { LockKeyhole, LogIn, Mail, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { login } from "@/features/auth/api";
import { getAuthErrorMessage } from "@/features/auth/auth-errors";

const DEFAULT_ADMIN_EMAIL = "admin@kicklogs.local";

export function LoginScreen() {
  return (
    <Suspense fallback={<LoginLoading />}>
      <LoginScreenInner />
    </Suspense>
  );
}

function LoginScreenInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const nextPath = useMemo(() => resolveNextPath(searchParams.get("next")), [searchParams]);
  const [email, setEmail] = useState(DEFAULT_ADMIN_EMAIL);
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function submitLogin() {
    setIsSubmitting(true);
    setError(null);

    try {
      await login({ email, password });
      router.replace(nextPath);
    } catch (caught) {
      setError(getAuthErrorMessage(caught));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="min-h-screen bg-background px-4 py-4 text-foreground md:px-8 md:py-6">
      <div className="mx-auto flex min-h-[calc(100vh-48px)] w-full max-w-[1120px] flex-col gap-6">
        <header className="flex flex-wrap items-center justify-between gap-4 border-b border-border bg-black px-4 py-4 md:px-6">
          <Link className="flex min-w-0 items-center gap-4" href="/search">
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
                  /login
                </span>
              </div>
              <p className="text-xs text-muted-foreground">Yönetim paneli oturumu</p>
            </div>
          </Link>

          <Button asChild variant="outline">
            <Link href="/search">
              <Search className="h-4 w-4 text-accent" />
              Public arama
            </Link>
          </Button>
        </header>

        <section className="grid flex-1 items-start gap-6 lg:grid-cols-[minmax(0,1fr)_420px]">
          <div className="rounded-lg border border-border bg-black p-5">
            <div className="mb-5 flex items-center gap-3 border-b border-border pb-4">
              <div className="flex h-9 w-9 items-center justify-center rounded-md bg-kick-background text-primary">
                <LockKeyhole className="h-4 w-4" />
              </div>
              <div>
                <h2 className="text-base font-semibold">Admin Girişi</h2>
                <p className="text-xs text-muted-foreground">
                  Kanal ve kullanıcı yönetimi için oturum açın
                </p>
              </div>
            </div>

            <form
              className="space-y-4"
              onSubmit={(event) => {
                event.preventDefault();
                void submitLogin();
              }}
            >
              <div>
                <label
                  className="mb-2 flex h-5 items-center gap-2 text-sm font-medium"
                  htmlFor="email"
                >
                  <Mail className="h-4 w-4 text-accent" />
                  E-posta
                </label>
                <Input
                  autoComplete="email"
                  id="email"
                  maxLength={320}
                  onChange={(event) => setEmail(event.target.value)}
                  type="email"
                  value={email}
                />
              </div>

              <div>
                <label
                  className="mb-2 flex h-5 items-center gap-2 text-sm font-medium"
                  htmlFor="password"
                >
                  <LockKeyhole className="h-4 w-4 text-accent" />
                  Parola
                </label>
                <Input
                  autoComplete="current-password"
                  id="password"
                  maxLength={256}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="admin123"
                  type="password"
                  value={password}
                />
              </div>

              {error ? (
                <div className="rounded-md border border-accent bg-kick-background px-3 py-2 text-sm text-foreground">
                  {error}
                </div>
              ) : null}

              <Button className="h-11 w-full" disabled={isSubmitting || !email || !password}>
                <LogIn className="h-4 w-4" />
                {isSubmitting ? "Giriş yapılıyor" : "Giriş yap"}
              </Button>
            </form>
          </div>

          <aside className="rounded-lg border border-border bg-black p-5">
            <div className="mb-4 text-sm font-semibold text-primary">Yerel MVP hesabı</div>
            <div className="space-y-3 text-sm">
              <InfoLine label="E-posta" value={DEFAULT_ADMIN_EMAIL} />
              <InfoLine label="Parola" value="admin123" />
              <InfoLine label="Rol" value="super_admin" />
            </div>
          </aside>
        </section>
      </div>
    </main>
  );
}

function InfoLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4 border-b border-border pb-3 last:border-b-0 last:pb-0">
      <span className="text-muted-foreground">{label}</span>
      <span className="truncate text-right text-foreground">{value}</span>
    </div>
  );
}

function LoginLoading() {
  return (
    <main className="min-h-screen bg-background px-6 py-6 text-foreground">
      <div className="mx-auto max-w-[1120px] rounded-lg border border-border bg-black p-6 text-sm text-muted-foreground">
        Login ekranı yükleniyor...
      </div>
    </main>
  );
}

function resolveNextPath(value: string | null) {
  if (!value || !value.startsWith("/") || value.startsWith("//")) {
    return "/admin";
  }

  return value;
}
