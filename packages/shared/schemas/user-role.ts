import { z } from "zod";

export const UserRoleSchema = z.object({
  user_id: z.string(),
  role_id: z.string(),
  created_at: z.string(),
});

export const CreateUserRoleSchema = z.object({
  user_id: z.string(),
  role_id: z.string(),
});

export const UpdateUserRoleSchema = z.object({
  user_id: z.string().optional(),
  role_id: z.string().optional(),
});

export type UserRole = z.infer<typeof UserRoleSchema>;
export type CreateUserRoleInput = z.infer<typeof CreateUserRoleSchema>;
export type UpdateUserRoleInput = z.infer<typeof UpdateUserRoleSchema>;
