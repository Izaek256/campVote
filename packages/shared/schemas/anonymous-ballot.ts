import { z } from "zod";

export const AnonymousBallotSchema = z.object({
  id: z.string(),
  election_id: z.string(),
  status: z.string(),
  cast_at: z.string(),
  selections: z.unknown().optional(),
});

export const CreateAnonymousBallotSchema = z.object({
  election_id: z.string(),
  status: z.string(),
  cast_at: z.string(),
});

export const UpdateAnonymousBallotSchema = z.object({
  election_id: z.string().optional(),
  status: z.string().optional(),
  cast_at: z.string().optional(),
});

export type AnonymousBallot = z.infer<typeof AnonymousBallotSchema>;
export type CreateAnonymousBallotInput = z.infer<typeof CreateAnonymousBallotSchema>;
export type UpdateAnonymousBallotInput = z.infer<typeof UpdateAnonymousBallotSchema>;
