import { z } from "zod";

export const ElectionRoleSchema = z.object({
  id: z.string(),
  election_id: z.string(),
  user_id: z.string(),
  role: z.string(),
  assigned_by_id: z.string(),
  created_at: z.string(),
});

export const CreateElectionRoleSchema = z.object({
  election_id: z.string(),
  user_id: z.string(),
  role: z.string(),
  assigned_by_id: z.string(),
});

export const UpdateElectionRoleSchema = z.object({
  election_id: z.string().optional(),
  user_id: z.string().optional(),
  role: z.string().optional(),
  assigned_by_id: z.string().optional(),
});

export type ElectionRole = z.infer<typeof ElectionRoleSchema>;
export type CreateElectionRoleInput = z.infer<typeof CreateElectionRoleSchema>;
export type UpdateElectionRoleInput = z.infer<typeof UpdateElectionRoleSchema>;
