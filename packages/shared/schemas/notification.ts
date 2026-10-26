import { z } from "zod";

export const NotificationSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  source: z.string(),
  severity: z.string(),
  title: z.string(),
  body: z.string(),
  link: z.string(),
  count: z.number().int(),
  read_at: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateNotificationSchema = z.object({
  user_id: z.string(),
  source: z.string(),
  severity: z.string(),
  title: z.string(),
  body: z.string(),
  link: z.string(),
  count: z.number().int(),
  read_at: z.string().nullable(),
});

export const UpdateNotificationSchema = z.object({
  user_id: z.string().optional(),
  source: z.string().optional(),
  severity: z.string().optional(),
  title: z.string().optional(),
  body: z.string().optional(),
  link: z.string().optional(),
  count: z.number().int().optional(),
  read_at: z.string().nullable(),
});

export type Notification = z.infer<typeof NotificationSchema>;
export type CreateNotificationInput = z.infer<typeof CreateNotificationSchema>;
export type UpdateNotificationInput = z.infer<typeof UpdateNotificationSchema>;
