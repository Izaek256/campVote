"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { ApiError, auth } from "@/lib/api";
import type { User } from "@/lib/types";

interface AuthState {
  user: User | null;
  loading: boolean;
  signInAdmin: (email: string, password: string) => Promise<AdminSignInResult>;
  verifyAdminSecondFactor: (
    pendingToken: string,
    code: string,
    useBackupCode?: boolean
  ) => Promise<void>;
  signInWithGoogle: (idToken: string) => Promise<void>;
  signOut: () => Promise<void>;
}

export type AdminSignInResult =
  | { requiresSecondFactor: true; pendingToken: string; method: string }
  | { requiresSecondFactor: false };

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    auth
      .me()
      .then((me) => {
        if (active) setUser(me);
      })
      .catch((error) => {
        // 401 simply means signed out; anything else is left for the page.
        if (active && !(error instanceof ApiError && error.status === 401)) {
          console.warn("session check failed", error);
        }
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  const signInAdmin = useCallback(async (email: string, password: string) => {
    const payload = await auth.adminLogin(email, password);
    if ("totp_required" in payload && payload.totp_required) {
      return {
        requiresSecondFactor: true as const,
        pendingToken: payload.pending_token,
        method: payload.method ?? "authenticator",
      };
    }
    if (!("user" in payload)) {
      throw new Error("The sign-in response was incomplete. Please try again.");
    }
    if (payload.user.role !== "ADMIN") {
      throw new Error("This account is not an administrator.");
    }
    setUser(payload.user);
    return { requiresSecondFactor: false as const };
  }, []);

  const verifyAdminSecondFactor = useCallback(
    async (pendingToken: string, code: string, useBackupCode = false) => {
      const payload = useBackupCode
        ? await auth.verifyBackupCode(pendingToken, code)
        : await auth.verifyTOTP(pendingToken, code);
      if (payload.user.role !== "ADMIN") {
        throw new Error("This account is not an administrator.");
      }
      setUser(payload.user);
    },
    []
  );

  const signInWithGoogle = useCallback(async (idToken: string) => {
    const payload = await auth.googleSignIn(idToken);
    setUser(payload.user);
  }, []);

  const signOut = useCallback(async () => {
    try {
      await auth.logout();
    } finally {
      setUser(null);
    }
  }, []);

  const value = useMemo(
    () => ({
      user,
      loading,
      signInAdmin,
      verifyAdminSecondFactor,
      signInWithGoogle,
      signOut,
    }),
    [user, loading, signInAdmin, verifyAdminSecondFactor, signInWithGoogle, signOut]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}

export function isAdmin(user: User | null): boolean {
  if (!user) return false;
  return user.role === "ADMIN";
}
