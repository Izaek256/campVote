import { z } from "zod";

export const TOTPPendingTokenSchema = z.object({
  id: z.number().int().nonnegative(),
  user_id: z.string(),
  expires_at: z.string(),
  created_at: z.string(),
});

export const CreateTOTPPendingTokenSchema = z.object({
  user_id: z.string(),
  expires_at: z.string(),
});

export const UpdateTOTPPendingTokenSchema = z.object({
  user_id: z.string().optional(),
  expires_at: z.string().optional(),
});

export type TOTPPendingToken = z.infer<typeof TOTPPendingTokenSchema>;
export type CreateTOTPPendingTokenInput = z.infer<typeof CreateTOTPPendingTokenSchema>;
export type UpdateTOTPPendingTokenInput = z.infer<typeof UpdateTOTPPendingTokenSchema>;
