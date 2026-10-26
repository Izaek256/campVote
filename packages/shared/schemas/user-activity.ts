import { z } from "zod";

export const UserActivitySchema = z.object({
  id: z.string(),
  user_id: z.string(),
  action: z.string(),
  severity: z.string(),
  summary: z.string(),
  resource_type: z.string(),
  resource_id: z.string(),
  ip_address: z.string(),
  user_agent: z.string(),
  metadata: z.string(),
  created_at: z.string(),
});

export const CreateUserActivitySchema = z.object({
  user_id: z.string(),
  action: z.string(),
  severity: z.string(),
  summary: z.string(),
  resource_type: z.string(),
  resource_id: z.string(),
  ip_address: z.string(),
  user_agent: z.string(),
  metadata: z.string(),
});

export const UpdateUserActivitySchema = z.object({
  user_id: z.string().optional(),
  action: z.string().optional(),
  severity: z.string().optional(),
  summary: z.string().optional(),
  resource_type: z.string().optional(),
  resource_id: z.string().optional(),
  ip_address: z.string().optional(),
  user_agent: z.string().optional(),
  metadata: z.string().optional(),
});

export type UserActivity = z.infer<typeof UserActivitySchema>;
export type CreateUserActivityInput = z.infer<typeof CreateUserActivitySchema>;
export type UpdateUserActivityInput = z.infer<typeof UpdateUserActivitySchema>;
