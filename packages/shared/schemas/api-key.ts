import { z } from "zod";

export const APIKeySchema = z.object({
  id: z.string(),
  user_id: z.string(),
  name: z.string(),
  prefix: z.string(),
  kind: z.string(),
  token: z.string(),
  scopes: z.array(z.string()).optional(),
  endpoints: z.array(z.string()).optional(),
  origins: z.array(z.string()).optional(),
  rate_limit: z.number().int(),
  last_used_at: z.string().nullable(),
  expires_at: z.string().nullable(),
  revoked_at: z.string().nullable(),
  created_at: z.string(),
});

export const CreateAPIKeySchema = z.object({
  user_id: z.string(),
  name: z.string(),
  prefix: z.string(),
  kind: z.string(),
  token: z.string(),
  scopes: z.array(z.string()).optional(),
  endpoints: z.array(z.string()).optional(),
  origins: z.array(z.string()).optional(),
  rate_limit: z.number().int(),
  last_used_at: z.string().nullable(),
  expires_at: z.string().nullable(),
  revoked_at: z.string().nullable(),
});

export const UpdateAPIKeySchema = z.object({
  user_id: z.string().optional(),
  name: z.string().optional(),
  prefix: z.string().optional(),
  kind: z.string().optional(),
  token: z.string().optional(),
  scopes: z.array(z.string()).optional(),
  endpoints: z.array(z.string()).optional(),
  origins: z.array(z.string()).optional(),
  rate_limit: z.number().int().optional(),
  last_used_at: z.string().nullable(),
  expires_at: z.string().nullable(),
  revoked_at: z.string().nullable(),
});

export type APIKey = z.infer<typeof APIKeySchema>;
export type CreateAPIKeyInput = z.infer<typeof CreateAPIKeySchema>;
export type UpdateAPIKeyInput = z.infer<typeof UpdateAPIKeySchema>;
