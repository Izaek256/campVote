import { z } from "zod";

export const DeletionJournalSchema = z.object({
  id: z.string(),
  deleted_user_id: z.string(),
  actor_id: z.string(),
  actor_email: z.string(),
  reason: z.string(),
  records_affected: z.number().int(),
  counts: z.string(),
  prev_hash: z.string(),
  hash: z.string(),
  created_at: z.string(),
});

export const CreateDeletionJournalSchema = z.object({
  deleted_user_id: z.string(),
  actor_id: z.string(),
  actor_email: z.string(),
  reason: z.string(),
  records_affected: z.number().int(),
  counts: z.string(),
  prev_hash: z.string(),
  hash: z.string(),
});

export const UpdateDeletionJournalSchema = z.object({
  deleted_user_id: z.string().optional(),
  actor_id: z.string().optional(),
  actor_email: z.string().optional(),
  reason: z.string().optional(),
  records_affected: z.number().int().optional(),
  counts: z.string().optional(),
  prev_hash: z.string().optional(),
  hash: z.string().optional(),
});

export type DeletionJournal = z.infer<typeof DeletionJournalSchema>;
export type CreateDeletionJournalInput = z.infer<typeof CreateDeletionJournalSchema>;
export type UpdateDeletionJournalInput = z.infer<typeof UpdateDeletionJournalSchema>;
