import { z } from "zod";

export const TrustedDeviceSchema = z.object({
  id: z.number().int().nonnegative(),
  user_id: z.string(),
  user_agent: z.string(),
  ip_address: z.string(),
  expires_at: z.string(),
  created_at: z.string(),
});

export const CreateTrustedDeviceSchema = z.object({
  user_id: z.string(),
  user_agent: z.string(),
  ip_address: z.string(),
  expires_at: z.string(),
});

export const UpdateTrustedDeviceSchema = z.object({
  user_id: z.string().optional(),
  user_agent: z.string().optional(),
  ip_address: z.string().optional(),
  expires_at: z.string().optional(),
});

export type TrustedDevice = z.infer<typeof TrustedDeviceSchema>;
export type CreateTrustedDeviceInput = z.infer<typeof CreateTrustedDeviceSchema>;
export type UpdateTrustedDeviceInput = z.infer<typeof UpdateTrustedDeviceSchema>;
