import { z } from "zod";

export const FeatureFlagSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  enabled: z.boolean(),
  rules: z.unknown(),
  created_at: z.string(),
  updated_at: z.string(),
  version: z.number().int(),
});

export const CreateFeatureFlagSchema = z.object({
  name: z.string(),
  description: z.string(),
  enabled: z.boolean(),
  rules: z.unknown(),
  version: z.number().int(),
});

export const UpdateFeatureFlagSchema = z.object({
  name: z.string().optional(),
  description: z.string().optional(),
  enabled: z.boolean().optional(),
  rules: z.unknown().optional(),
  version: z.number().int().optional(),
});

export type FeatureFlag = z.infer<typeof FeatureFlagSchema>;
export type CreateFeatureFlagInput = z.infer<typeof CreateFeatureFlagSchema>;
export type UpdateFeatureFlagInput = z.infer<typeof UpdateFeatureFlagSchema>;
