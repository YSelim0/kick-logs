"use client";

import { useCallback, useEffect, useState } from "react";

import { getCurrentUser } from "@/features/auth/api";
import { isUnauthorizedError } from "@/features/auth/auth-errors";
import type { AdminUser } from "@/types/api";

type AuthStatus = "loading" | "authenticated" | "unauthenticated" | "error";

export function useCurrentUser() {
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<AdminUser | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setStatus("loading");
    setError(null);

    try {
      const currentUser = await getCurrentUser();
      setUser(currentUser);
      setStatus("authenticated");
      return currentUser;
    } catch (caught) {
      setUser(null);

      if (isUnauthorizedError(caught)) {
        setStatus("unauthenticated");
        return null;
      }

      setStatus("error");
      setError(caught instanceof Error ? caught.message : "Oturum bilgisi alınamadı.");
      return null;
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return {
    error,
    refresh,
    setUser,
    status,
    user
  };
}
