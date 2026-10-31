import { z } from "zod";

export const BallotVersionSchema = z.object({
  id: z.string(),
  election_id: z.string(),
  version: z.number().int(),
  content_hash: z.string(),
  content_json: z.string(),
  published_by: z.string(),
  published_at: z.string(),
  status: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const BallotContentSchema = z.object({
  positions: z.array(
    z.object({
      id: z.string(),
      title: z.string(),
      description: z.string(),
      question_type: z.string(),
      min_selections: z.number().int(),
      max_selections: z.number().int(),
      seats: z.number().int(),
      allow_blank: z.boolean(),
      display_order: z.number().int(),
      candidates: z.array(
        z.object({
          id: z.string(),
          full_name: z.string(),
          photo_url: z.string(),
          manifesto: z.string(),
          bio: z.string(),
          ballot_number: z.number().int(),
        })
      ),
    })
  ),
});

export type BallotVersion = z.infer<typeof BallotVersionSchema>;
export type BallotContent = z.infer<typeof BallotContentSchema>;
