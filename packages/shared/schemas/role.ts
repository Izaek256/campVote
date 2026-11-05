import { z } from "zod";

export const RoleSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  is_system: z.boolean(),
  version: z.number().int(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateRoleSchema = z.object({
  name: z.string(),
  description: z.string(),
  is_system: z.boolean(),
  version: z.number().int(),
});

export const UpdateRoleSchema = z.object({
  name: z.string().optional(),
  description: z.string().optional(),
  is_system: z.boolean().optional(),
  version: z.number().int().optional(),
});

export type Role = z.infer<typeof RoleSchema>;
export type CreateRoleInput = z.infer<typeof CreateRoleSchema>;
export type UpdateRoleInput = z.infer<typeof UpdateRoleSchema>;
