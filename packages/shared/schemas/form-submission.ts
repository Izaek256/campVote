import { z } from "zod";

export const FormSubmissionSchema = z.object({
  id: z.string(),
  share_id: z.string(),
  resource_name: z.string(),
  record_id: z.string(),
  ip: z.string(),
  user_agent: z.string(),
  created_at: z.string(),
});

export const CreateFormSubmissionSchema = z.object({
  share_id: z.string(),
  resource_name: z.string(),
  record_id: z.string(),
  ip: z.string(),
  user_agent: z.string(),
});

export const UpdateFormSubmissionSchema = z.object({
  share_id: z.string().optional(),
  resource_name: z.string().optional(),
  record_id: z.string().optional(),
  ip: z.string().optional(),
  user_agent: z.string().optional(),
});

export type FormSubmission = z.infer<typeof FormSubmissionSchema>;
export type CreateFormSubmissionInput = z.infer<typeof CreateFormSubmissionSchema>;
export type UpdateFormSubmissionInput = z.infer<typeof UpdateFormSubmissionSchema>;
