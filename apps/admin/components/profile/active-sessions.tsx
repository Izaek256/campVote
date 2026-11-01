"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { getApiErrorMessage } from "@/lib/api-core";
import { LogOut } from "@/lib/icons";

interface Session {
  id: string;
  user_agent?: string;
  ip_address?: string;
  created_at: string;
  last_used_at?: string;
  current?: boolean;
}

function when(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  return Number.isNaN(d.getTime())
    ? "—"
    : d.toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" });
}

export function ActiveSessions() {
  const qc = useQueryClient();
  const [error, setError] = useState<string | null>(null);

  const { data, isPending } = useQuery<Session[]>({
    queryKey: ["sessions"],
    queryFn: async () => {
      const { data } = await apiClient.get<{ data: Session[] }>("/api/v1/auth/sessions");
      return data.data ?? [];
    },
  });

  const revoke = useMutation({
    mutationFn: (id: string) => apiClient.delete(`/api/v1/auth/sessions/${id}`),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["sessions"] }),
    onError: (e) => setError(getApiErrorMessage(e, "Could not revoke that session.")),
  });

  const sessions = data ?? [];

  return (
    <div className="rounded-xl border border-border bg-bg-elevated">
      <div className="border-b border-border px-5 py-4">
        <h2 className="font-semibold text-foreground">Active sessions</h2>
        <p className="mt-1 text-sm text-text-muted">
          Every device or browser that is currently signed in.
        </p>
      </div>

      {error && (
        <p role="alert" className="px-5 py-3 text-sm text-danger">
          {error}
        </p>
      )}

      {isPending ? (
        <p className="px-5 py-8 text-sm text-text-muted">Loading sessions…</p>
      ) : sessions.length === 0 ? (
        <p className="px-5 py-8 text-sm text-text-muted">No active sessions found.</p>
      ) : (
        <ul className="divide-y divide-border/60">
          {sessions.map((s) => (
            <li key={s.id} className="flex items-center justify-between gap-4 px-5 py-3">
              <div>
                <p className="text-sm font-medium text-foreground">
                  {s.user_agent ?? "Unknown device"}
                  {s.current ? (
                    <span className="ml-2 rounded-full bg-accent/10 px-2 py-0.5 text-xs font-medium text-accent">
                      This session
                    </span>
                  ) : null}
                </p>
                <p className="mt-0.5 text-xs text-text-muted">
                  {s.ip_address ?? "Unknown IP"} · Started {when(s.created_at)}
                  {s.last_used_at ? ` · Last seen ${when(s.last_used_at)}` : ""}
                </p>
              </div>
              {!s.current && (
                <button
                  type="button"
                  onClick={() => revoke.mutate(s.id)}
                  disabled={revoke.isPending}
                  className="inline-flex shrink-0 items-center gap-1.5 rounded-lg border border-border px-2.5 py-1 text-xs text-danger transition-colors hover:bg-danger/10"
                  aria-label={`Revoke session from ${s.ip_address ?? "unknown device"}`}
                >
                  <LogOut className="h-3 w-3" aria-hidden="true" />
                  Revoke
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
