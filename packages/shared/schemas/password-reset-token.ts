import { z } from "zod";

export const PasswordResetTokenSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  request_ip: z.string(),
  expires_at: z.string(),
  used_at: z.string().nullable(),
  created_at: z.string(),
});

export const CreatePasswordResetTokenSchema = z.object({
  user_id: z.string(),
  request_ip: z.string(),
  expires_at: z.string(),
  used_at: z.string().nullable(),
});

export const UpdatePasswordResetTokenSchema = z.object({
  user_id: z.string().optional(),
  request_ip: z.string().optional(),
  expires_at: z.string().optional(),
  used_at: z.string().nullable(),
});

export type PasswordResetToken = z.infer<typeof PasswordResetTokenSchema>;
export type CreatePasswordResetTokenInput = z.infer<typeof CreatePasswordResetTokenSchema>;
export type UpdatePasswordResetTokenInput = z.infer<typeof UpdatePasswordResetTokenSchema>;
