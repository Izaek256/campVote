import { z } from "zod";

export const TicketSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  subject: z.string(),
  description: z.string(),
  status: z.string(),
  priority: z.string(),
  labels: z.string(),
  assignee_id: z.string(),
  last_reply_at: z.string().nullable(),
  closed_at: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
  user: z.unknown().optional(),
  assignee: z.unknown().optional(),
  replies: z.unknown().optional(),
});

export const CreateTicketSchema = z.object({
  user_id: z.string(),
  subject: z.string(),
  description: z.string(),
  status: z.string(),
  priority: z.string(),
  labels: z.string(),
  assignee_id: z.string(),
  last_reply_at: z.string().nullable(),
  closed_at: z.string().nullable(),
});

export const UpdateTicketSchema = z.object({
  user_id: z.string().optional(),
  subject: z.string().optional(),
  description: z.string().optional(),
  status: z.string().optional(),
  priority: z.string().optional(),
  labels: z.string().optional(),
  assignee_id: z.string().optional(),
  last_reply_at: z.string().nullable(),
  closed_at: z.string().nullable(),
});

export type Ticket = z.infer<typeof TicketSchema>;
export type CreateTicketInput = z.infer<typeof CreateTicketSchema>;
export type UpdateTicketInput = z.infer<typeof UpdateTicketSchema>;
