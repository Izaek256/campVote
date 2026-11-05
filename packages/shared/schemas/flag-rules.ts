import { z } from "zod";

export const FlagRulesSchema = z.object({
  rollout_percentage: z.number().int(),
  allowlist_user_ids: z.unknown(),
  blocklist_user_ids: z.unknown(),
  enabled_from: z.string().nullable(),
  enabled_until: z.string().nullable(),
  variants: z.unknown(),
  attributes: z.unknown(),
});

export const CreateFlagRulesSchema = z.object({
  rollout_percentage: z.number().int(),
  allowlist_user_ids: z.unknown(),
  blocklist_user_ids: z.unknown(),
  enabled_from: z.string().nullable(),
  enabled_until: z.string().nullable(),
  variants: z.unknown(),
  attributes: z.unknown(),
});

export const UpdateFlagRulesSchema = z.object({
  rollout_percentage: z.number().int().optional(),
  allowlist_user_ids: z.unknown().optional(),
  blocklist_user_ids: z.unknown().optional(),
  enabled_from: z.string().nullable(),
  enabled_until: z.string().nullable(),
  variants: z.unknown().optional(),
  attributes: z.unknown().optional(),
});

export type FlagRules = z.infer<typeof FlagRulesSchema>;
export type CreateFlagRulesInput = z.infer<typeof CreateFlagRulesSchema>;
export type UpdateFlagRulesInput = z.infer<typeof UpdateFlagRulesSchema>;
