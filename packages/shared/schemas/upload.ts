import { z } from "zod";

export const UploadSchema = z.object({
  id: z.string(),
  filename: z.string(),
  original_name: z.string(),
  mime_type: z.string(),
  size: z.number().int(),
  path: z.string(),
  url: z.string(),
  thumbnail_url: z.string(),
  user_id: z.string(),
  version: z.number().int(),
  claimed_at: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateUploadSchema = z.object({
  filename: z.string(),
  original_name: z.string(),
  mime_type: z.string(),
  size: z.number().int(),
  path: z.string(),
  url: z.string(),
  thumbnail_url: z.string(),
  user_id: z.string(),
  version: z.number().int(),
  claimed_at: z.string().nullable(),
});

export const UpdateUploadSchema = z.object({
  filename: z.string().optional(),
  original_name: z.string().optional(),
  mime_type: z.string().optional(),
  size: z.number().int().optional(),
  path: z.string().optional(),
  url: z.string().optional(),
  thumbnail_url: z.string().optional(),
  user_id: z.string().optional(),
  version: z.number().int().optional(),
  claimed_at: z.string().nullable(),
});

export type Upload = z.infer<typeof UploadSchema>;
export type CreateUploadInput = z.infer<typeof CreateUploadSchema>;
export type UpdateUploadInput = z.infer<typeof UpdateUploadSchema>;
