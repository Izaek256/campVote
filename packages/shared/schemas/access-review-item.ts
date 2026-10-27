import { z } from "zod";

export const AccessReviewItemSchema = z.object({
  id: z.string(),
  review_id: z.string(),
  user_id: z.string(),
  user_email: z.string(),
  role_id: z.string(),
  role_name: z.string(),
  decision: z.string(),
  decided_by: z.string(),
  decided_by_email: z.string(),
  decided_at: z.string().nullable(),
  note: z.string(),
  created_at: z.string(),
});

export const CreateAccessReviewItemSchema = z.object({
  review_id: z.string(),
  user_id: z.string(),
  user_email: z.string(),
  role_id: z.string(),
  role_name: z.string(),
  decision: z.string(),
  decided_by: z.string(),
  decided_by_email: z.string(),
  decided_at: z.string().nullable(),
  note: z.string(),
});

export const UpdateAccessReviewItemSchema = z.object({
  review_id: z.string().optional(),
  user_id: z.string().optional(),
  user_email: z.string().optional(),
  role_id: z.string().optional(),
  role_name: z.string().optional(),
  decision: z.string().optional(),
  decided_by: z.string().optional(),
  decided_by_email: z.string().optional(),
  decided_at: z.string().nullable(),
  note: z.string().optional(),
});

export type AccessReviewItem = z.infer<typeof AccessReviewItemSchema>;
export type CreateAccessReviewItemInput = z.infer<typeof CreateAccessReviewItemSchema>;
export type UpdateAccessReviewItemInput = z.infer<typeof UpdateAccessReviewItemSchema>;
