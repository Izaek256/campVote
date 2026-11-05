import { z } from "zod";

export const AccessReviewSchema = z.object({
  id: z.string(),
  name: z.string(),
  note: z.string(),
  status: z.string(),
  created_by: z.string(),
  created_by_email: z.string(),
  completed_by: z.string(),
  completed_by_email: z.string(),
  completed_at: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
  items: z.unknown().optional(),
});

export const CreateAccessReviewSchema = z.object({
  name: z.string(),
  note: z.string(),
  status: z.string(),
  created_by: z.string(),
  created_by_email: z.string(),
  completed_by: z.string(),
  completed_by_email: z.string(),
  completed_at: z.string().nullable(),
});

export const UpdateAccessReviewSchema = z.object({
  name: z.string().optional(),
  note: z.string().optional(),
  status: z.string().optional(),
  created_by: z.string().optional(),
  created_by_email: z.string().optional(),
  completed_by: z.string().optional(),
  completed_by_email: z.string().optional(),
  completed_at: z.string().nullable(),
});

export type AccessReview = z.infer<typeof AccessReviewSchema>;
export type CreateAccessReviewInput = z.infer<typeof CreateAccessReviewSchema>;
export type UpdateAccessReviewInput = z.infer<typeof UpdateAccessReviewSchema>;
