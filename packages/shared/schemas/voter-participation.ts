import { z } from "zod";

export const VoterParticipationSchema = z.object({
  id: z.string(),
  election_id: z.string(),
  user_id: z.string(),
  confirmation_token: z.string(),
  voted_at: z.string(),
});

export const CreateVoterParticipationSchema = z.object({
  election_id: z.string(),
  user_id: z.string(),
  confirmation_token: z.string(),
  voted_at: z.string(),
});

export const UpdateVoterParticipationSchema = z.object({
  election_id: z.string().optional(),
  user_id: z.string().optional(),
  confirmation_token: z.string().optional(),
  voted_at: z.string().optional(),
});

export type VoterParticipation = z.infer<typeof VoterParticipationSchema>;
export type CreateVoterParticipationInput = z.infer<typeof CreateVoterParticipationSchema>;
export type UpdateVoterParticipationInput = z.infer<typeof UpdateVoterParticipationSchema>;
