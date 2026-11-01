"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { ShieldCheck, ShieldOff } from "@/lib/icons";

interface TotpStatus {
  enabled: boolean;
  backup_codes_remaining: number;
}

export function TwoFactorCard() {
  const { data, isPending } = useQuery<TotpStatus>({
    queryKey: ["totp-status"],
    queryFn: async () => {
      const { data } = await apiClient.get<{ data: TotpStatus }>("/api/v1/auth/totp/status");
      return data.data;
    },
  });

  return (
    <div className="rounded-xl border border-border bg-bg-elevated">
      <div className="flex items-start justify-between gap-4 border-b border-border px-5 py-4">
        <div>
          <h2 className="font-semibold text-foreground">Two-factor authentication</h2>
          <p className="mt-1 text-sm text-text-muted">
            Require a code from your authenticator app when signing in.
          </p>
        </div>
        {!isPending && (
          <span
            className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ${
              data?.enabled
                ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                : "bg-bg-secondary text-text-muted"
            }`}
          >
            {data?.enabled ? (
              <>
                <ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />
                Enabled
              </>
            ) : (
              <>
                <ShieldOff className="h-3.5 w-3.5" aria-hidden="true" />
                Off
              </>
            )}
          </span>
        )}
      </div>
      <div className="px-5 py-4 text-sm text-text-muted">
        {isPending ? (
          "Loading…"
        ) : data?.enabled ? (
          <>
            Two-factor authentication is active. You have{" "}
            <strong className="text-foreground">{data.backup_codes_remaining}</strong> backup{" "}
            {data.backup_codes_remaining === 1 ? "code" : "codes"} remaining.
          </>
        ) : (
          "Two-factor authentication is not enabled. Set it up to make your account more secure."
        )}
      </div>
    </div>
  );
}
