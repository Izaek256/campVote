import { z } from "zod";

export const WebhookEventSchema = z.object({
  id: z.string(),
  provider: z.string(),
  event_type: z.string(),
  external_id: z.string().nullable().optional(),
  payload: z.unknown(),
  status: z.string(),
  handler_error: z.string(),
  retry_count: z.number().int(),
  processed_at: z.string().nullable(),
  created_at: z.string(),
});

export const CreateWebhookEventSchema = z.object({
  provider: z.string(),
  event_type: z.string(),
  external_id: z.string().nullable().optional(),
  payload: z.unknown(),
  status: z.string(),
  handler_error: z.string(),
  retry_count: z.number().int(),
  processed_at: z.string().nullable(),
});

export const UpdateWebhookEventSchema = z.object({
  provider: z.string().optional(),
  event_type: z.string().optional(),
  external_id: z.string().nullable().optional(),
  payload: z.unknown().optional(),
  status: z.string().optional(),
  handler_error: z.string().optional(),
  retry_count: z.number().int().optional(),
  processed_at: z.string().nullable(),
});

export type WebhookEvent = z.infer<typeof WebhookEventSchema>;
export type CreateWebhookEventInput = z.infer<typeof CreateWebhookEventSchema>;
export type UpdateWebhookEventInput = z.infer<typeof UpdateWebhookEventSchema>;
