import { z } from "zod";

export const BackupScheduleSchema = z.object({
  frequency: z.string(),
  time: z.string(),
  enabled: z.boolean(),
  updated_at: z.string(),
});

export const CreateBackupScheduleSchema = z.object({
  frequency: z.string(),
  time: z.string(),
  enabled: z.boolean(),
});

export const UpdateBackupScheduleSchema = z.object({
  frequency: z.string().optional(),
  time: z.string().optional(),
  enabled: z.boolean().optional(),
});

export type BackupSchedule = z.infer<typeof BackupScheduleSchema>;
export type CreateBackupScheduleInput = z.infer<typeof CreateBackupScheduleSchema>;
export type UpdateBackupScheduleInput = z.infer<typeof UpdateBackupScheduleSchema>;
