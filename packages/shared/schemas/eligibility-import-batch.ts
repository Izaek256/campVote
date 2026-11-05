import { z } from "zod";

export const EligibilityImportBatchSchema = z.object({
  id: z.string(),
  election_id: z.string(),
  imported_by_id: z.string(),
  source: z.string(),
  file_name: z.string(),
  content_hash: z.string(),
  total: z.number().int(),
  created: z.number().int(),
  updated: z.number().int(),
  revoked: z.number().int(),
  created_at: z.string(),
});

export const CreateEligibilityImportBatchSchema = z.object({
  election_id: z.string(),
  imported_by_id: z.string(),
  source: z.string(),
  file_name: z.string(),
  content_hash: z.string(),
  total: z.number().int(),
  created: z.number().int(),
  updated: z.number().int(),
  revoked: z.number().int(),
});

export const UpdateEligibilityImportBatchSchema = z.object({
  election_id: z.string().optional(),
  imported_by_id: z.string().optional(),
  source: z.string().optional(),
  file_name: z.string().optional(),
  content_hash: z.string().optional(),
  total: z.number().int().optional(),
  created: z.number().int().optional(),
  updated: z.number().int().optional(),
  revoked: z.number().int().optional(),
});

export type EligibilityImportBatch = z.infer<typeof EligibilityImportBatchSchema>;
export type CreateEligibilityImportBatchInput = z.infer<typeof CreateEligibilityImportBatchSchema>;
export type UpdateEligibilityImportBatchInput = z.infer<typeof UpdateEligibilityImportBatchSchema>;
