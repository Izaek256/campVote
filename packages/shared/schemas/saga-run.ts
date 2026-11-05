import { z } from "zod";

export const SagaRunSchema = z.object({
  id: z.string(),
  name: z.string(),
  key: z.string().nullable().optional(),
  input: z.unknown(),
  state: z.unknown(),
  status: z.string(),
  cursor: z.number().int(),
  attempts: z.number().int(),
  last_error: z.string(),
  available_at: z.string(),
  claimed_by: z.string(),
  claimed_at: z.string().nullable(),
  finished_at: z.string().nullable(),
  steps: z.unknown().optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateSagaRunSchema = z.object({
  name: z.string(),
  key: z.string().nullable().optional(),
  input: z.unknown(),
  state: z.unknown(),
  status: z.string(),
  cursor: z.number().int(),
  attempts: z.number().int(),
  last_error: z.string(),
  available_at: z.string(),
  claimed_by: z.string(),
  claimed_at: z.string().nullable(),
  finished_at: z.string().nullable(),
});

export const UpdateSagaRunSchema = z.object({
  name: z.string().optional(),
  key: z.string().nullable().optional(),
  input: z.unknown().optional(),
  state: z.unknown().optional(),
  status: z.string().optional(),
  cursor: z.number().int().optional(),
  attempts: z.number().int().optional(),
  last_error: z.string().optional(),
  available_at: z.string().optional(),
  claimed_by: z.string().optional(),
  claimed_at: z.string().nullable(),
  finished_at: z.string().nullable(),
});

export type SagaRun = z.infer<typeof SagaRunSchema>;
export type CreateSagaRunInput = z.infer<typeof CreateSagaRunSchema>;
export type UpdateSagaRunInput = z.infer<typeof UpdateSagaRunSchema>;
