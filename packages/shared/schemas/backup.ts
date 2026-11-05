import { z } from "zod";

export const BackupSchema = z.object({
  id: z.string(),
  kind: z.string(),
  status: z.string(),
  size_bytes: z.number().int(),
  table_count: z.number().int(),
  row_count: z.number().int(),
  error: z.string(),
  created_at: z.string(),
  completed_at: z.string().nullable(),
});

export const CreateBackupSchema = z.object({
  kind: z.string(),
  status: z.string(),
  size_bytes: z.number().int(),
  table_count: z.number().int(),
  row_count: z.number().int(),
  error: z.string(),
  completed_at: z.string().nullable(),
});

export const UpdateBackupSchema = z.object({
  kind: z.string().optional(),
  status: z.string().optional(),
  size_bytes: z.number().int().optional(),
  table_count: z.number().int().optional(),
  row_count: z.number().int().optional(),
  error: z.string().optional(),
  completed_at: z.string().nullable(),
});

export type Backup = z.infer<typeof BackupSchema>;
export type CreateBackupInput = z.infer<typeof CreateBackupSchema>;
export type UpdateBackupInput = z.infer<typeof UpdateBackupSchema>;
