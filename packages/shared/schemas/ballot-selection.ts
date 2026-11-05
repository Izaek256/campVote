import { z } from "zod";

export const BallotSelectionSchema = z.object({
  id: z.string(),
  ballot_id: z.string(),
  position_id: z.string(),
  candidate_id: z.string().nullable().optional(),
  choice: z.string(),
  created_at: z.string(),
});

export const CreateBallotSelectionSchema = z.object({
  ballot_id: z.string(),
  position_id: z.string(),
  candidate_id: z.string().nullable().optional(),
  choice: z.string(),
});

export const UpdateBallotSelectionSchema = z.object({
  ballot_id: z.string().optional(),
  position_id: z.string().optional(),
  candidate_id: z.string().nullable().optional(),
  choice: z.string().optional(),
});

export type BallotSelection = z.infer<typeof BallotSelectionSchema>;
export type CreateBallotSelectionInput = z.infer<typeof CreateBallotSelectionSchema>;
export type UpdateBallotSelectionInput = z.infer<typeof UpdateBallotSelectionSchema>;
