import { z } from "zod";

export const MagicLinkTokenSchema = z.object({
  id: z.number().int().nonnegative(),
  user_id: z.string(),
  used_at: z.string().nullable(),
  ip_address: z.string(),
  user_agent: z.string(),
  expires_at: z.string(),
  created_at: z.string(),
});

export const CreateMagicLinkTokenSchema = z.object({
  user_id: z.string(),
  used_at: z.string().nullable(),
  ip_address: z.string(),
  user_agent: z.string(),
  expires_at: z.string(),
});

export const UpdateMagicLinkTokenSchema = z.object({
  user_id: z.string().optional(),
  used_at: z.string().nullable(),
  ip_address: z.string().optional(),
  user_agent: z.string().optional(),
  expires_at: z.string().optional(),
});

export type MagicLinkToken = z.infer<typeof MagicLinkTokenSchema>;
export type CreateMagicLinkTokenInput = z.infer<typeof CreateMagicLinkTokenSchema>;
export type UpdateMagicLinkTokenInput = z.infer<typeof UpdateMagicLinkTokenSchema>;
