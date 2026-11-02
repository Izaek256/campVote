"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { isAdmin, useAuth } from "./auth-provider";

export function RequireAuth({
  children,
  adminOnly = false,
}: {
  children: React.ReactNode;
  /** Only ADMIN accounts may view this page; everyone else is sent to their elections. */
  adminOnly?: boolean;
}) {
  const { user, loading } = useAuth();
  const router = useRouter();
  const denied = adminOnly && !!user && !isAdmin(user);

  useEffect(() => {
    if (loading) return;
    if (!user) {
      router.replace("/login");
      return;
    }
    if (adminOnly && !isAdmin(user)) router.replace("/elections");
  }, [loading, user, adminOnly, router]);

  if (loading) {
    return (
      <div className="app-wrap" role="status" aria-busy="true" aria-label="Loading your session">
        <div className="skel" style={{ height: 24, width: "40%", marginBottom: 24 }} />
        <div className="panel stack">
          <div className="skel" style={{ height: 14 }} />
          <div className="skel" style={{ height: 14, width: "80%" }} />
          <div className="skel" style={{ height: 14, width: "60%" }} />
        </div>
      </div>
    );
  }
  if (!user || denied) return null;
  return <>{children}</>;
}
