import { z } from "zod";

export const OutboxMessageSchema = z.object({
  id: z.string(),
  topic: z.string(),
  key: z.string().nullable().optional(),
  payload: z.unknown(),
  status: z.string(),
  attempts: z.number().int(),
  last_error: z.string(),
  available_at: z.string(),
  claimed_by: z.string(),
  claimed_at: z.string().nullable(),
  delivered_at: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateOutboxMessageSchema = z.object({
  topic: z.string(),
  key: z.string().nullable().optional(),
  payload: z.unknown(),
  status: z.string(),
  attempts: z.number().int(),
  last_error: z.string(),
  available_at: z.string(),
  claimed_by: z.string(),
  claimed_at: z.string().nullable(),
  delivered_at: z.string().nullable(),
});

export const UpdateOutboxMessageSchema = z.object({
  topic: z.string().optional(),
  key: z.string().nullable().optional(),
  payload: z.unknown().optional(),
  status: z.string().optional(),
  attempts: z.number().int().optional(),
  last_error: z.string().optional(),
  available_at: z.string().optional(),
  claimed_by: z.string().optional(),
  claimed_at: z.string().nullable(),
  delivered_at: z.string().nullable(),
});

export type OutboxMessage = z.infer<typeof OutboxMessageSchema>;
export type CreateOutboxMessageInput = z.infer<typeof CreateOutboxMessageSchema>;
export type UpdateOutboxMessageInput = z.infer<typeof UpdateOutboxMessageSchema>;
