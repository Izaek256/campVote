import { z } from "zod";

export const PositionSchema = z.object({
  id: z.string(),
  election_id: z.string(),
  title: z.string(),
  description: z.string(),
  question_type: z.string(),
  min_selections: z.number().int(),
  max_selections: z.number().int(),
  seats: z.number().int(),
  allow_blank: z.boolean(),
  display_order: z.number().int(),
  created_at: z.string(),
  updated_at: z.string(),
  candidates: z.unknown().optional(),
});

export const CreatePositionSchema = z.object({
  election_id: z.string(),
  title: z.string(),
  description: z.string(),
  question_type: z.string(),
  min_selections: z.number().int(),
  max_selections: z.number().int(),
  seats: z.number().int(),
  allow_blank: z.boolean(),
  display_order: z.number().int(),
});

export const UpdatePositionSchema = z.object({
  election_id: z.string().optional(),
  title: z.string().optional(),
  description: z.string().optional(),
  question_type: z.string().optional(),
  min_selections: z.number().int().optional(),
  max_selections: z.number().int().optional(),
  seats: z.number().int().optional(),
  allow_blank: z.boolean().optional(),
  display_order: z.number().int().optional(),
});

export type Position = z.infer<typeof PositionSchema>;
export type CreatePositionInput = z.infer<typeof CreatePositionSchema>;
export type UpdatePositionInput = z.infer<typeof UpdatePositionSchema>;
