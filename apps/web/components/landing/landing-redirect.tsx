"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { isAdmin, isStaff, useAuth } from "@/components/auth-provider";

/**
 * A signed-in visitor belongs in their dashboard, not on the shop window.
 * Redirecting after the session check avoids bouncing anonymous visitors
 * through a server-side cookie guess that breaks when the API is on another
 * origin.
 */
export function LandingSignedInRedirect() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading || !user) return;
    router.replace(isAdmin(user) || isStaff(user) ? "/staff" : "/elections");
  }, [loading, user, router]);

  return null;
}
