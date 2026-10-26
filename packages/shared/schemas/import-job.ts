import { z } from "zod";

export const ImportJobSchema = z.object({
  id: z.string(),
  resource: z.string(),
  status: z.string(),
  total: z.number().int(),
  processed: z.number().int(),
  created: z.number().int(),
  skipped: z.number().int(),
  failed: z.number().int(),
  message: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateImportJobSchema = z.object({
  resource: z.string(),
  status: z.string(),
  total: z.number().int(),
  processed: z.number().int(),
  created: z.number().int(),
  skipped: z.number().int(),
  failed: z.number().int(),
  message: z.string(),
});

export const UpdateImportJobSchema = z.object({
  resource: z.string().optional(),
  status: z.string().optional(),
  total: z.number().int().optional(),
  processed: z.number().int().optional(),
  created: z.number().int().optional(),
  skipped: z.number().int().optional(),
  failed: z.number().int().optional(),
  message: z.string().optional(),
});

export type ImportJob = z.infer<typeof ImportJobSchema>;
export type CreateImportJobInput = z.infer<typeof CreateImportJobSchema>;
export type UpdateImportJobInput = z.infer<typeof UpdateImportJobSchema>;
