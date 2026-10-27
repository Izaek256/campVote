import { z } from "zod";

export const WebAuthnSessionSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  purpose: z.string(),
  expires_at: z.string(),
  created_at: z.string(),
});

export const CreateWebAuthnSessionSchema = z.object({
  user_id: z.string(),
  purpose: z.string(),
  expires_at: z.string(),
});

export const UpdateWebAuthnSessionSchema = z.object({
  user_id: z.string().optional(),
  purpose: z.string().optional(),
  expires_at: z.string().optional(),
});

export type WebAuthnSession = z.infer<typeof WebAuthnSessionSchema>;
export type CreateWebAuthnSessionInput = z.infer<typeof CreateWebAuthnSessionSchema>;
export type UpdateWebAuthnSessionInput = z.infer<typeof UpdateWebAuthnSessionSchema>;
