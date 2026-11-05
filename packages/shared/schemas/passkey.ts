import { z } from "zod";

export const PasskeySchema = z.object({
  id: z.string(),
  user_id: z.string(),
  credential_id: z.string(),
  transports: z.string(),
  backup_eligible: z.boolean(),
  synced: z.boolean(),
  name: z.string(),
  last_used_at: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreatePasskeySchema = z.object({
  user_id: z.string(),
  credential_id: z.string(),
  transports: z.string(),
  backup_eligible: z.boolean(),
  synced: z.boolean(),
  name: z.string(),
  last_used_at: z.string().nullable(),
});

export const UpdatePasskeySchema = z.object({
  user_id: z.string().optional(),
  credential_id: z.string().optional(),
  transports: z.string().optional(),
  backup_eligible: z.boolean().optional(),
  synced: z.boolean().optional(),
  name: z.string().optional(),
  last_used_at: z.string().nullable(),
});

export type Passkey = z.infer<typeof PasskeySchema>;
export type CreatePasskeyInput = z.infer<typeof CreatePasskeySchema>;
export type UpdatePasskeyInput = z.infer<typeof UpdatePasskeySchema>;
