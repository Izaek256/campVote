/**
 * Mutations for updating the signed-in user's own profile.
 */
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";

interface UpdateProfileInput {
  first_name?: string;
  last_name?: string;
  avatar_url?: string | null;
  [key: string]: unknown;
}

/** Sends a PUT /profile request and invalidates the `me` cache on success. */
export function useUpdateProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: UpdateProfileInput) => {
      const { data } = await apiClient.put("/api/v1/profile", body);
      return data;
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["me"] });
    },
  });
}

interface ChangePasswordInput {
  current_password: string;
  password: string;
}

/** POST /profile/password — change the signed-in user's password. */
export function useChangePassword() {
  return useMutation({
    mutationFn: async (body: ChangePasswordInput) => {
      const { data } = await apiClient.post("/api/v1/auth/change-password", {
        current_password: body.current_password,
        new_password: body.password,
      });
      return data;
    },
  });
}
