"use client";

import { useTranslations } from "next-intl";
import { getUiErrorKey, type UiErrorKey } from "@/i18n/errors";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { Loader2, LockKeyhole, Mail, Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { createAdminUser, listAdminUsers } from "@/features/users/api";
import type { AdminUser } from "@/types/api";

export function UserAdmin() {
  const t = useTranslations("userAdmin");
  const errors = useTranslations("common.errors");
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<UiErrorKey | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);

  const loadUsers = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      setUsers(await listAdminUsers());
    } catch (caught) {
      setError(getUiErrorKey(caught, "adminMutation"));
      setUsers([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadUsers();
  }, [loadUsers]);

  async function submitUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const normalizedEmail = email.trim();
    if (!normalizedEmail || password.length < 8) return;

    setIsCreating(true);
    setError(null);

    try {
      const createdUser = await createAdminUser({ email: normalizedEmail, password });
      setEmail("");
      setPassword("");
      setUsers((current) => mergeUser(current, createdUser));
    } catch (caught) {
      setError(getUiErrorKey(caught, "adminMutation"));
    } finally {
      setIsCreating(false);
    }
  }

  return (
    <section className="rounded-lg border border-border bg-panel p-5">
      <div className="mb-5 flex flex-col gap-0.5">
        <span className="text-[14px] font-semibold text-foreground">{t("title")}</span>
        <span className="font-mono text-[11px] text-faint">
          {t("count", { count: users.length })}
        </span>
      </div>

      <form
        className="mb-5 grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]"
        onSubmit={submitUser}
      >
        <div className="flex flex-col gap-1.5">
          <label
            className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
            htmlFor="admin-email"
          >
            {t("email")}
          </label>
          <div className="flex h-[38px] items-center gap-2 rounded-md border border-border-strong bg-elevated px-3">
            <Mail className="h-3.5 w-3.5 shrink-0 text-faint" />
            <input
              autoComplete="off"
              className="min-w-0 flex-1 bg-transparent text-[13px] text-foreground outline-none placeholder:text-faint"
              id="admin-email"
              maxLength={320}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="operator@example.com"
              type="email"
              value={email}
            />
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <label
            className="font-mono text-[11px] font-medium tracking-[0.5px] text-muted-foreground"
            htmlFor="admin-password"
          >
            {t("password")}
          </label>
          <div className="flex h-[38px] items-center gap-2 rounded-md border border-border-strong bg-elevated px-3">
            <LockKeyhole className="h-3.5 w-3.5 shrink-0 text-faint" />
            <input
              autoComplete="new-password"
              className="min-w-0 flex-1 bg-transparent text-[13px] text-foreground outline-none placeholder:text-faint"
              id="admin-password"
              maxLength={256}
              minLength={8}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t("passwordPlaceholder")}
              type="password"
              value={password}
            />
          </div>
        </div>

        <div className="flex items-end">
          <Button disabled={isCreating || !email.trim() || password.length < 8} type="submit">
            {isCreating ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            {t("create")}
          </Button>
        </div>
      </form>

      {error ? (
        <div className="mb-4 rounded-md border border-danger bg-elevated px-3 py-2 text-[13px]">
          {errors(error)}
        </div>
      ) : null}

      <div className="rounded-lg border border-border">
        <div className="flex items-center border-b border-border px-3 py-2">
          <span className="flex-1 font-mono text-[10px] font-medium tracking-[0.8px] text-faint">
            {t("email")}
          </span>
          <span className="hidden w-36 font-mono text-[10px] font-medium tracking-[0.8px] text-faint sm:block">
            {t("role")}
          </span>
          <span className="hidden w-20 font-mono text-[10px] font-medium tracking-[0.8px] text-faint sm:block">
            {t("status")}
          </span>
        </div>

        {isLoading ? (
          <div className="px-4 py-10 text-center text-[13px] text-muted-foreground">
            {t("loading")}
          </div>
        ) : users.length ? (
          users.map((user) => <UserRow key={user.id} user={user} />)
        ) : (
          <div className="px-4 py-10 text-center text-[13px] text-muted-foreground">
            {t("empty")}
          </div>
        )}
      </div>
    </section>
  );
}

function UserRow({ user }: { user: AdminUser }) {
  const t = useTranslations("userAdmin");
  const statusBadge = (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 font-mono text-[10px] font-medium ${
        user.is_active ? "bg-accent-muted text-accent" : "bg-elevated text-faint"
      }`}
    >
      <span className={`h-1 w-1 rounded-full ${user.is_active ? "bg-accent" : "bg-faint"}`} />
      {user.is_active ? t("active") : t("inactive")}
    </span>
  );

  return (
    <div className="border-b border-border px-3 py-2.5 last:border-b-0 sm:flex sm:items-center">
      <div className="min-w-0 flex-1">
        <span className="truncate font-sans text-[13px] text-foreground">{user.email}</span>
        <div className="mt-1 flex items-center gap-2 sm:hidden">
          <span className="font-mono text-[11px] text-muted-foreground">
            {t(`roles.${user.role}`)}
          </span>
          {statusBadge}
        </div>
      </div>
      <div className="hidden w-36 sm:block">
        <span className="font-mono text-[12px] text-muted-foreground">
          {t(`roles.${user.role}`)}
        </span>
      </div>
      <div className="hidden w-20 sm:block">{statusBadge}</div>
    </div>
  );
}

function mergeUser(current: AdminUser[], user: AdminUser) {
  const next = current.filter((item) => item.id !== user.id);
  return [...next, user].sort((a, b) => a.email.localeCompare(b.email));
}
