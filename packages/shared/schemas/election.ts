import { z } from "zod";

export const ElectionSchema = z.object({
  id: z.string(),
  title: z.string(),
  description: z.string(),
  category: z.string(),
  organizing_unit: z.string(),
  status: z.string(),
  starts_at: z.string(),
  ends_at: z.string(),
  result_policy: z.string(),
  allowed_domains: z.string(),
  require_roster: z.boolean(),
  created_by_id: z.string(),
  version: z.number().int(),
  created_at: z.string(),
  updated_at: z.string(),
  positions: z.unknown().optional(),
});

export const CreateElectionSchema = z.object({
  title: z.string(),
  description: z.string(),
  category: z.string(),
  organizing_unit: z.string(),
  status: z.string(),
  starts_at: z.string(),
  ends_at: z.string(),
  result_policy: z.string(),
  allowed_domains: z.string(),
  require_roster: z.boolean(),
  created_by_id: z.string(),
  version: z.number().int(),
});

export const UpdateElectionSchema = z.object({
  title: z.string().optional(),
  description: z.string().optional(),
  category: z.string().optional(),
  organizing_unit: z.string().optional(),
  status: z.string().optional(),
  starts_at: z.string().optional(),
  ends_at: z.string().optional(),
  result_policy: z.string().optional(),
  allowed_domains: z.string().optional(),
  require_roster: z.boolean().optional(),
  created_by_id: z.string().optional(),
  version: z.number().int().optional(),
});

export type Election = z.infer<typeof ElectionSchema>;
export type CreateElectionInput = z.infer<typeof CreateElectionSchema>;
export type UpdateElectionInput = z.infer<typeof UpdateElectionSchema>;
