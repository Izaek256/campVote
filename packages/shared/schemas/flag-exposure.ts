import { z } from "zod";

export const FlagExposureSchema = z.object({
  id: z.string(),
  flag_id: z.string(),
  flag_name: z.string(),
  user_id: z.string(),
  variant: z.string(),
  created_at: z.string(),
});

export const CreateFlagExposureSchema = z.object({
  flag_id: z.string(),
  flag_name: z.string(),
  user_id: z.string(),
  variant: z.string(),
});

export const UpdateFlagExposureSchema = z.object({
  flag_id: z.string().optional(),
  flag_name: z.string().optional(),
  user_id: z.string().optional(),
  variant: z.string().optional(),
});

export type FlagExposure = z.infer<typeof FlagExposureSchema>;
export type CreateFlagExposureInput = z.infer<typeof CreateFlagExposureSchema>;
export type UpdateFlagExposureInput = z.infer<typeof UpdateFlagExposureSchema>;
