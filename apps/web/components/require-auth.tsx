"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useAuth } from "./auth-provider";

export function RequireAuth({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !user) router.replace("/login");
  }, [loading, user, router]);

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
  if (!user) return null;
  return <>{children}</>;
}
