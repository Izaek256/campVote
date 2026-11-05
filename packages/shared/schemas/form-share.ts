import { z } from "zod";

export const FormShareSchema = z.object({
  id: z.string(),
  resource_name: z.string(),
  token: z.string(),
  has_password: z.boolean(),
  enabled: z.boolean(),
  submission_count: z.number().int(),
  created_by_user_id: z.string(),
  label: z.string(),
  custom_title: z.string(),
  custom_description: z.string(),
  hidden_fields: z.array(z.string()).optional(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateFormShareSchema = z.object({
  resource_name: z.string(),
  token: z.string(),
  has_password: z.boolean(),
  enabled: z.boolean(),
  submission_count: z.number().int(),
  created_by_user_id: z.string(),
  label: z.string(),
  custom_title: z.string(),
  custom_description: z.string(),
  hidden_fields: z.array(z.string()).optional(),
});

export const UpdateFormShareSchema = z.object({
  resource_name: z.string().optional(),
  token: z.string().optional(),
  has_password: z.boolean().optional(),
  enabled: z.boolean().optional(),
  submission_count: z.number().int().optional(),
  created_by_user_id: z.string().optional(),
  label: z.string().optional(),
  custom_title: z.string().optional(),
  custom_description: z.string().optional(),
  hidden_fields: z.array(z.string()).optional(),
});

export type FormShare = z.infer<typeof FormShareSchema>;
export type CreateFormShareInput = z.infer<typeof CreateFormShareSchema>;
export type UpdateFormShareInput = z.infer<typeof UpdateFormShareSchema>;
