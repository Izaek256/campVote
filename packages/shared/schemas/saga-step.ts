import { z } from "zod";

export const SagaStepSchema = z.object({
  id: z.string(),
  run_id: z.string(),
  idx: z.number().int(),
  name: z.string(),
  status: z.string(),
  attempts: z.number().int(),
  last_error: z.string(),
  started_at: z.string().nullable(),
  finished_at: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateSagaStepSchema = z.object({
  run_id: z.string(),
  idx: z.number().int(),
  name: z.string(),
  status: z.string(),
  attempts: z.number().int(),
  last_error: z.string(),
  started_at: z.string().nullable(),
  finished_at: z.string().nullable(),
});

export const UpdateSagaStepSchema = z.object({
  run_id: z.string().optional(),
  idx: z.number().int().optional(),
  name: z.string().optional(),
  status: z.string().optional(),
  attempts: z.number().int().optional(),
  last_error: z.string().optional(),
  started_at: z.string().nullable(),
  finished_at: z.string().nullable(),
});

export type SagaStep = z.infer<typeof SagaStepSchema>;
export type CreateSagaStepInput = z.infer<typeof CreateSagaStepSchema>;
export type UpdateSagaStepInput = z.infer<typeof UpdateSagaStepSchema>;
