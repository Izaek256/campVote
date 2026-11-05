import { z } from "zod";

export const TwoFactorConfigSchema = z.object({
  id: z.number().int().nonnegative(),
  user_id: z.string(),
  enabled: z.boolean(),
  method: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateTwoFactorConfigSchema = z.object({
  user_id: z.string(),
  enabled: z.boolean(),
  method: z.string(),
});

export const UpdateTwoFactorConfigSchema = z.object({
  user_id: z.string().optional(),
  enabled: z.boolean().optional(),
  method: z.string().optional(),
});

export type TwoFactorConfig = z.infer<typeof TwoFactorConfigSchema>;
export type CreateTwoFactorConfigInput = z.infer<typeof CreateTwoFactorConfigSchema>;
export type UpdateTwoFactorConfigInput = z.infer<typeof UpdateTwoFactorConfigSchema>;
