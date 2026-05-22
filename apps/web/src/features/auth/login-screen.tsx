"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { ArrowLeft, LockKeyhole, Mail } from "lucide-react";
import { Suspense, useMemo, useState } from "react";

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
    <main className="flex min-h-screen items-center justify-center bg-page">
      <div className="flex w-[380px] flex-col gap-5 rounded-[10px] border border-border bg-panel p-8">
        <div className="flex flex-col items-center gap-2">
          <div className="flex items-center gap-2">
            <Image
                alt="Kick Logs"
                className="h-6 w-6 rounded-md object-contain"
                height={24}
                priority
                src="/app-logo.png"
                width={24}
              />
            <span className="font-sans text-lg font-semibold text-foreground">kick logs</span>
          </div>
          <p className="font-sans text-[13px] text-muted-foreground">
            Yönetim panelinize giriş yapın
          </p>
        </div>

        <form
          className="flex flex-col gap-3.5"
          onSubmit={(e) => {
            e.preventDefault();
            void submitLogin();
          }}
        >
          <div className="flex flex-col gap-1.5">
            <label
              className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
              htmlFor="email"
            >
              E-POSTA
            </label>
            <div className="flex h-[38px] items-center gap-2 rounded-md border border-border-strong bg-elevated px-3">
              <Mail className="h-3.5 w-3.5 shrink-0 text-faint" />
              <input
                autoComplete="email"
                className="flex-1 bg-transparent text-[13px] text-foreground outline-none placeholder:text-faint"
                id="email"
                maxLength={320}
                onChange={(e) => setEmail(e.target.value)}
                type="email"
                value={email}
              />
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <label
              className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
              htmlFor="password"
            >
              ŞİFRE
            </label>
            <div className="flex h-[38px] items-center gap-2 rounded-md border border-border-strong bg-elevated px-3">
              <LockKeyhole className="h-3.5 w-3.5 shrink-0 text-faint" />
              <input
                autoComplete="current-password"
                className="flex-1 bg-transparent text-[13px] text-foreground outline-none placeholder:text-faint"
                id="password"
                maxLength={256}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                type="password"
                value={password}
              />
            </div>
          </div>

          {error ? (
            <div className="rounded-md border border-danger bg-elevated px-3 py-2 text-[12px] text-foreground">
              {error}
            </div>
          ) : null}

          <button
            className="flex h-10 w-full items-center justify-center rounded-md bg-accent font-sans text-[13px] font-semibold text-accent-foreground disabled:opacity-50"
            disabled={isSubmitting || !email || !password}
            type="submit"
          >
            {isSubmitting ? "Giriş yapılıyor..." : "Giriş yap"}
          </button>
        </form>

        <Link
          className="flex items-center justify-center gap-1.5 font-sans text-[12px] text-faint hover:text-muted-foreground"
          href="/search"
        >
          <ArrowLeft className="h-[11px] w-[11px]" />
          Public arama sayfasına dön
        </Link>
      </div>
    </main>
  );
}

function LoginLoading() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-page">
      <div className="w-[380px] rounded-[10px] border border-border bg-panel p-8 text-[13px] text-muted-foreground">
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
