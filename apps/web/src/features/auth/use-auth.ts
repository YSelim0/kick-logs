"use client";

import { useTranslations } from "next-intl";
import { getUiErrorKey, type UiErrorKey } from "@/i18n/errors";
import { useCallback, useEffect, useState } from "react";

import { getCurrentUser } from "@/features/auth/api";
import { isUnauthorizedError } from "@/features/auth/auth-errors";
import type { AdminUser } from "@/types/api";

type AuthStatus = "loading" | "authenticated" | "unauthenticated" | "error";

export function useCurrentUser() {
  const errors = useTranslations("common.errors");
  const [status, setStatus] = useState<AuthStatus>("loading");
  const [user, setUser] = useState<AdminUser | null>(null);
  const [error, setError] = useState<UiErrorKey | null>(null);

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
      setError(getUiErrorKey(caught, "session"));
      return null;
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return {
    error: error ? errors(error) : null,
    refresh,
    setUser,
    status,
    user
  };
}
