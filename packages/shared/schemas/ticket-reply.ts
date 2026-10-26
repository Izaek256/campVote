import { z } from "zod";

export const TicketReplySchema = z.object({
  id: z.string(),
  ticket_id: z.string(),
  user_id: z.string(),
  body: z.string(),
  is_admin_reply: z.boolean(),
  created_at: z.string(),
  updated_at: z.string(),
  user: z.unknown().optional(),
});

export const CreateTicketReplySchema = z.object({
  ticket_id: z.string(),
  user_id: z.string(),
  body: z.string(),
  is_admin_reply: z.boolean(),
});

export const UpdateTicketReplySchema = z.object({
  ticket_id: z.string().optional(),
  user_id: z.string().optional(),
  body: z.string().optional(),
  is_admin_reply: z.boolean().optional(),
});

export type TicketReply = z.infer<typeof TicketReplySchema>;
export type CreateTicketReplyInput = z.infer<typeof CreateTicketReplySchema>;
export type UpdateTicketReplyInput = z.infer<typeof UpdateTicketReplySchema>;
