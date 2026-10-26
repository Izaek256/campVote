import { z } from "zod";

export const CandidateSchema = z.object({
  id: z.string(),
  election_id: z.string(),
  position_id: z.string(),
  full_name: z.string(),
  photo_url: z.string(),
  manifesto: z.string(),
  bio: z.string(),
  ballot_number: z.number().int(),
  approval_status: z.string(),
  review_notes: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateCandidateSchema = z.object({
  election_id: z.string(),
  position_id: z.string(),
  full_name: z.string(),
  photo_url: z.string(),
  manifesto: z.string(),
  bio: z.string(),
  ballot_number: z.number().int(),
  approval_status: z.string(),
  review_notes: z.string(),
});

export const UpdateCandidateSchema = z.object({
  election_id: z.string().optional(),
  position_id: z.string().optional(),
  full_name: z.string().optional(),
  photo_url: z.string().optional(),
  manifesto: z.string().optional(),
  bio: z.string().optional(),
  ballot_number: z.number().int().optional(),
  approval_status: z.string().optional(),
  review_notes: z.string().optional(),
});

export type Candidate = z.infer<typeof CandidateSchema>;
export type CreateCandidateInput = z.infer<typeof CreateCandidateSchema>;
export type UpdateCandidateInput = z.infer<typeof UpdateCandidateSchema>;
