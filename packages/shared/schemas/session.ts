import { z } from "zod";

export const SessionSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  user_agent: z.string(),
  ip: z.string(),
  created_at: z.string(),
  last_seen_at: z.string(),
  expires_at: z.string(),
  revoked_at: z.string().nullable(),
});

export const CreateSessionSchema = z.object({
  user_id: z.string(),
  user_agent: z.string(),
  ip: z.string(),
  last_seen_at: z.string(),
  expires_at: z.string(),
  revoked_at: z.string().nullable(),
});

export const UpdateSessionSchema = z.object({
  user_id: z.string().optional(),
  user_agent: z.string().optional(),
  ip: z.string().optional(),
  last_seen_at: z.string().optional(),
  expires_at: z.string().optional(),
  revoked_at: z.string().nullable(),
});

export type Session = z.infer<typeof SessionSchema>;
export type CreateSessionInput = z.infer<typeof CreateSessionSchema>;
export type UpdateSessionInput = z.infer<typeof UpdateSessionSchema>;
