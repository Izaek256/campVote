import { z } from "zod";

export const RecoveryContactTokenSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  kind: z.unknown(),
  destination: z.string(),
  expires_at: z.string(),
  used_at: z.string().nullable(),
  created_at: z.string(),
});

export const CreateRecoveryContactTokenSchema = z.object({
  user_id: z.string(),
  kind: z.unknown(),
  destination: z.string(),
  expires_at: z.string(),
  used_at: z.string().nullable(),
});

export const UpdateRecoveryContactTokenSchema = z.object({
  user_id: z.string().optional(),
  kind: z.unknown().optional(),
  destination: z.string().optional(),
  expires_at: z.string().optional(),
  used_at: z.string().nullable(),
});

export type RecoveryContactToken = z.infer<typeof RecoveryContactTokenSchema>;
export type CreateRecoveryContactTokenInput = z.infer<typeof CreateRecoveryContactTokenSchema>;
export type UpdateRecoveryContactTokenInput = z.infer<typeof UpdateRecoveryContactTokenSchema>;
