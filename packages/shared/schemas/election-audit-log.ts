import { z } from "zod";

export const ElectionAuditLogSchema = z.object({
  id: z.string(),
  election_id: z.string(),
  actor_id: z.string(),
  action: z.string(),
  details: z.string(),
  ip_address: z.string(),
  created_at: z.string(),
});

export const CreateElectionAuditLogSchema = z.object({
  election_id: z.string(),
  actor_id: z.string(),
  action: z.string(),
  details: z.string(),
  ip_address: z.string(),
});

export const UpdateElectionAuditLogSchema = z.object({
  election_id: z.string().optional(),
  actor_id: z.string().optional(),
  action: z.string().optional(),
  details: z.string().optional(),
  ip_address: z.string().optional(),
});

export type ElectionAuditLog = z.infer<typeof ElectionAuditLogSchema>;
export type CreateElectionAuditLogInput = z.infer<typeof CreateElectionAuditLogSchema>;
export type UpdateElectionAuditLogInput = z.infer<typeof UpdateElectionAuditLogSchema>;
