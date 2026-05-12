"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { Loader2, Mail, Plus, ShieldCheck, UsersRound } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { createAdminUser, listAdminUsers } from "@/features/users/api";
import type { AdminUser } from "@/types/api";

export function UserAdmin() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);

  const loadUsers = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      setUsers(await listAdminUsers());
    } catch (caught) {
      setError(resolveUserAdminError(caught));
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
    if (!normalizedEmail || password.length < 8) {
      return;
    }

    setIsCreating(true);
    setError(null);

    try {
      const createdUser = await createAdminUser({
        email: normalizedEmail,
        password
      });
      setEmail("");
      setPassword("");
      setUsers((current) => mergeUser(current, createdUser));
    } catch (caught) {
      setError(resolveUserAdminError(caught));
    } finally {
      setIsCreating(false);
    }
  }

  return (
    <section className="rounded-lg border border-border bg-black p-5">
      <div className="mb-5 flex flex-wrap items-start justify-between gap-4 border-b border-border pb-4">
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-md bg-kick-background text-primary">
            <UsersRound className="h-4 w-4" />
          </div>
          <div>
            <h2 className="text-base font-semibold">Admin Kullanıcıları</h2>
            <p className="text-xs text-muted-foreground">
              Sadece super admin yeni yönetici hesabı oluşturabilir
            </p>
          </div>
        </div>

        <div className="rounded-md bg-kick-background px-3 py-2 text-xs">
          <span className="text-muted-foreground">Kullanıcı</span>
          <span className="ml-2 font-semibold text-primary">{users.length}</span>
        </div>
      </div>

      <form
        className="mb-4 grid gap-3 lg:grid-cols-[minmax(0,1fr)_220px_150px]"
        onSubmit={submitUser}
      >
        <div>
          <label
            className="mb-2 flex h-5 items-center gap-2 text-sm font-medium"
            htmlFor="admin-email"
          >
            <Mail className="h-4 w-4 text-accent" />
            Yeni admin e-postası
          </label>
          <Input
            autoComplete="off"
            id="admin-email"
            maxLength={320}
            onChange={(event) => setEmail(event.target.value)}
            placeholder="operator@example.com"
            type="email"
            value={email}
          />
        </div>

        <div>
          <label
            className="mb-2 flex h-5 items-center gap-2 text-sm font-medium"
            htmlFor="admin-password"
          >
            <ShieldCheck className="h-4 w-4 text-accent" />
            Geçici parola
          </label>
          <Input
            autoComplete="new-password"
            id="admin-password"
            maxLength={256}
            minLength={8}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="en az 8 karakter"
            type="password"
            value={password}
          />
        </div>

        <div className="flex items-end">
          <Button
            className="h-11 w-full"
            disabled={isCreating || !email.trim() || password.length < 8}
          >
            {isCreating ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            Oluştur
          </Button>
        </div>
      </form>

      {error ? (
        <div className="mb-4 rounded-md border border-accent bg-kick-background px-3 py-2 text-sm">
          {error}
        </div>
      ) : null}

      <div className="overflow-hidden rounded-md border border-border">
        <div className="hidden grid-cols-[minmax(220px,1fr)_140px_100px] bg-kick-background px-3 py-2 text-xs font-medium text-muted-foreground md:grid">
          <div>E-posta</div>
          <div>Rol</div>
          <div>Durum</div>
        </div>

        {isLoading ? (
          <div className="px-4 py-10 text-center text-sm text-muted-foreground">
            Admin kullanıcıları yükleniyor...
          </div>
        ) : users.length ? (
          users.map((user, index) => <UserRow isAlt={index % 2 === 1} key={user.id} user={user} />)
        ) : (
          <div className="px-4 py-10 text-center text-sm text-muted-foreground">
            Admin kullanıcısı bulunamadı.
          </div>
        )}
      </div>
    </section>
  );
}

function UserRow({ isAlt, user }: { isAlt: boolean; user: AdminUser }) {
  return (
    <div
      className={
        isAlt
          ? "grid gap-3 border-t border-border/70 bg-kick-background px-3 py-3 text-sm md:grid-cols-[minmax(220px,1fr)_140px_100px] md:items-center"
          : "grid gap-3 border-t border-border/70 bg-black px-3 py-3 text-sm md:grid-cols-[minmax(220px,1fr)_140px_100px] md:items-center"
      }
    >
      <div className="min-w-0">
        <div className="truncate font-medium text-foreground">{user.email}</div>
        <div className="text-xs text-muted-foreground md:hidden">{user.role}</div>
      </div>
      <div className="hidden text-foreground md:block">{user.role}</div>
      <div>
        <span
          className={
            user.is_active
              ? "inline-flex rounded-md bg-primary px-2 py-1 text-xs font-semibold text-primary-foreground"
              : "inline-flex rounded-md bg-kick-background px-2 py-1 text-xs text-muted-foreground"
          }
        >
          {user.is_active ? "Aktif" : "Pasif"}
        </span>
      </div>
    </div>
  );
}

function mergeUser(current: AdminUser[], user: AdminUser) {
  const next = current.filter((item) => item.id !== user.id);
  return [...next, user].sort((first, second) => first.email.localeCompare(second.email));
}

function resolveUserAdminError(error: unknown) {
  if (error instanceof Error) {
    return error.message;
  }

  return "Kullanıcı işlemi tamamlanırken hata oluştu.";
}
