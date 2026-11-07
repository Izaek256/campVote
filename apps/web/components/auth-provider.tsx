"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { ApiError, auth } from "@/lib/api";
import type { User } from "@/lib/types";

interface AuthState {
  user: User | null;
  loading: boolean;
  signIn: (email: string, password: string) => Promise<void>;
  signInWithGoogle: (idToken: string) => Promise<void>;
  signOut: () => Promise<void>;
}

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

  const signIn = useCallback(async (email: string, password: string) => {
    const payload = await auth.login(email, password);
    setUser(payload.user);
  }, []);

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
    () => ({ user, loading, signIn, signInWithGoogle, signOut }),
    [user, loading, signIn, signInWithGoogle, signOut]
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
