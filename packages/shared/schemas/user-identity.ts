import { z } from "zod";

export const UserIdentitySchema = z.object({
  id: z.string(),
  user_id: z.string(),
  provider: z.string(),
  subject: z.string(),
  email: z.string(),
  last_login_at: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateUserIdentitySchema = z.object({
  user_id: z.string(),
  provider: z.string(),
  subject: z.string(),
  email: z.string(),
  last_login_at: z.string().nullable(),
});

export const UpdateUserIdentitySchema = z.object({
  user_id: z.string().optional(),
  provider: z.string().optional(),
  subject: z.string().optional(),
  email: z.string().optional(),
  last_login_at: z.string().nullable(),
});

export type UserIdentity = z.infer<typeof UserIdentitySchema>;
export type CreateUserIdentityInput = z.infer<typeof CreateUserIdentitySchema>;
export type UpdateUserIdentityInput = z.infer<typeof UpdateUserIdentitySchema>;
