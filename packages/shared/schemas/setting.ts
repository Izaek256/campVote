import { z } from "zod";

export const SettingSchema = z.object({
  id: z.string(),
  key: z.string(),
  scope: z.string(),
  scope_id: z.string(),
  value: z.string(),
  updated_by: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateSettingSchema = z.object({
  key: z.string(),
  scope: z.string(),
  scope_id: z.string(),
  value: z.string(),
  updated_by: z.string(),
});

export const UpdateSettingSchema = z.object({
  key: z.string().optional(),
  scope: z.string().optional(),
  scope_id: z.string().optional(),
  value: z.string().optional(),
  updated_by: z.string().optional(),
});

export type Setting = z.infer<typeof SettingSchema>;
export type CreateSettingInput = z.infer<typeof CreateSettingSchema>;
export type UpdateSettingInput = z.infer<typeof UpdateSettingSchema>;
