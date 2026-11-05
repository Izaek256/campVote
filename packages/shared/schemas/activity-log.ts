import { z } from "zod";

export const ActivityLogSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  method: z.string(),
  path: z.string(),
  status: z.number().int(),
  payload_digest: z.string(),
  ip_address: z.string(),
  user_agent: z.string(),
  duration_ms: z.number().int(),
  resource: z.string(),
  resource_ids: z.string(),
  record_count: z.number().int(),
  prev_hash: z.string(),
  hash: z.string(),
  created_at: z.string(),
});

export const CreateActivityLogSchema = z.object({
  user_id: z.string(),
  method: z.string(),
  path: z.string(),
  status: z.number().int(),
  payload_digest: z.string(),
  ip_address: z.string(),
  user_agent: z.string(),
  duration_ms: z.number().int(),
  resource: z.string(),
  resource_ids: z.string(),
  record_count: z.number().int(),
  prev_hash: z.string(),
  hash: z.string(),
});

export const UpdateActivityLogSchema = z.object({
  user_id: z.string().optional(),
  method: z.string().optional(),
  path: z.string().optional(),
  status: z.number().int().optional(),
  payload_digest: z.string().optional(),
  ip_address: z.string().optional(),
  user_agent: z.string().optional(),
  duration_ms: z.number().int().optional(),
  resource: z.string().optional(),
  resource_ids: z.string().optional(),
  record_count: z.number().int().optional(),
  prev_hash: z.string().optional(),
  hash: z.string().optional(),
});

export type ActivityLog = z.infer<typeof ActivityLogSchema>;
export type CreateActivityLogInput = z.infer<typeof CreateActivityLogSchema>;
export type UpdateActivityLogInput = z.infer<typeof UpdateActivityLogSchema>;
