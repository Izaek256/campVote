/**
 * Hooks for accessing the currently-signed-in user in the admin app.
 *
 * The admin app does not yet have a full auth provider; these thin wrappers
 * call the API directly so existing pages compile and work while the auth
 * layer is being built out.
 */
import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

export interface AdminUser {
  id: string;
  first_name?: string;
  last_name?: string;
  email: string;
  role: string;
  active: boolean;
  avatar?: string | null;
  avatar_url?: string | null;
  job_title?: string | null;
  bio?: string | null;
  email_verified_at?: string | null;
  created_at?: string;
}

/** Fetch the currently-authenticated admin user. */
export function useMe() {
  return useQuery<AdminUser>({
    queryKey: ["me"],
    queryFn: async () => {
      const { data } = await apiClient.get<{ data: AdminUser }>("/api/v1/auth/me");
      return data.data;
    },
    retry: false,
    staleTime: 60_000,
  });
}
