import { z } from "zod";

export const SAMLKeypairSchema = z.object({
  id: z.string(),
  cert_pem: z.string(),
  not_after: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateSAMLKeypairSchema = z.object({
  cert_pem: z.string(),
  not_after: z.string(),
});

export const UpdateSAMLKeypairSchema = z.object({
  cert_pem: z.string().optional(),
  not_after: z.string().optional(),
});

export type SAMLKeypair = z.infer<typeof SAMLKeypairSchema>;
export type CreateSAMLKeypairInput = z.infer<typeof CreateSAMLKeypairSchema>;
export type UpdateSAMLKeypairInput = z.infer<typeof UpdateSAMLKeypairSchema>;
