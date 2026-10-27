import { z } from "zod";

export const RecoveryContactSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  kind: z.unknown(),
  destination: z.string(),
  verified_at: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateRecoveryContactSchema = z.object({
  user_id: z.string(),
  kind: z.unknown(),
  destination: z.string(),
  verified_at: z.string(),
});

export const UpdateRecoveryContactSchema = z.object({
  user_id: z.string().optional(),
  kind: z.unknown().optional(),
  destination: z.string().optional(),
  verified_at: z.string().optional(),
});

export type RecoveryContact = z.infer<typeof RecoveryContactSchema>;
export type CreateRecoveryContactInput = z.infer<typeof CreateRecoveryContactSchema>;
export type UpdateRecoveryContactInput = z.infer<typeof UpdateRecoveryContactSchema>;
