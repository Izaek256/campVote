"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { isAdmin, isStaff, useAuth } from "./auth-provider";

export function RequireAuth({
  children,
  adminOnly = false,
  staffOnly = false,
}: {
  children: React.ReactNode;
  /** Only ADMIN accounts may view this page; everyone else is sent to their elections. */
  adminOnly?: boolean;
  /** System admins and election-assigned staff may view this page. */
  staffOnly?: boolean;
}) {
  const { user, loading } = useAuth();
  const router = useRouter();
  // A scoped staff admin reaches only the pages of the election they run; the
  // server enforces the same scope on every request.
  const allowed =
    !adminOnly && !staffOnly ? true : adminOnly ? isAdmin(user) : isAdmin(user) || isStaff(user);
  const denied = (adminOnly || staffOnly) && !!user && !allowed;

  useEffect(() => {
    if (loading) return;
    if (!user) {
      router.replace("/login");
      return;
    }
    if (!allowed) router.replace("/elections");
  }, [loading, user, allowed, router]);

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
