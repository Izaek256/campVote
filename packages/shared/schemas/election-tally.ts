import { z } from "zod";

export const ElectionTallySchema = z.object({
  id: z.string(),
  election_id: z.string(),
  position_id: z.string(),
  candidate_id: z.string().nullable().optional(),
  choice: z.string(),
  votes: z.number().int(),
  valid_votes_total: z.number().int(),
  percentage: z.number(),
  turnout_percentage: z.number(),
  is_winner: z.boolean(),
  is_tie: z.boolean(),
  computed_at: z.string(),
});

export const CreateElectionTallySchema = z.object({
  election_id: z.string(),
  position_id: z.string(),
  candidate_id: z.string().nullable().optional(),
  choice: z.string(),
  votes: z.number().int(),
  valid_votes_total: z.number().int(),
  percentage: z.number(),
  turnout_percentage: z.number(),
  is_winner: z.boolean(),
  is_tie: z.boolean(),
  computed_at: z.string(),
});

export const UpdateElectionTallySchema = z.object({
  election_id: z.string().optional(),
  position_id: z.string().optional(),
  candidate_id: z.string().nullable().optional(),
  choice: z.string().optional(),
  votes: z.number().int().optional(),
  valid_votes_total: z.number().int().optional(),
  percentage: z.number().optional(),
  turnout_percentage: z.number().optional(),
  is_winner: z.boolean().optional(),
  is_tie: z.boolean().optional(),
  computed_at: z.string().optional(),
});

export type ElectionTally = z.infer<typeof ElectionTallySchema>;
export type CreateElectionTallyInput = z.infer<typeof CreateElectionTallySchema>;
export type UpdateElectionTallyInput = z.infer<typeof UpdateElectionTallySchema>;
