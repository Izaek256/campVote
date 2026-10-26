import { z } from "zod";

export const EmailVerificationTokenSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  email: z.string(),
  expires_at: z.string(),
  used_at: z.string().nullable(),
  created_at: z.string(),
});

export const CreateEmailVerificationTokenSchema = z.object({
  user_id: z.string(),
  email: z.string(),
  expires_at: z.string(),
  used_at: z.string().nullable(),
});

export const UpdateEmailVerificationTokenSchema = z.object({
  user_id: z.string().optional(),
  email: z.string().optional(),
  expires_at: z.string().optional(),
  used_at: z.string().nullable(),
});

export type EmailVerificationToken = z.infer<typeof EmailVerificationTokenSchema>;
export type CreateEmailVerificationTokenInput = z.infer<typeof CreateEmailVerificationTokenSchema>;
export type UpdateEmailVerificationTokenInput = z.infer<typeof UpdateEmailVerificationTokenSchema>;
