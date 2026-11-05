import { z } from "zod";

export const ElectionRosterSchema = z.object({
  id: z.string(),
  election_id: z.string(),
  student_number: z.string(),
  reg_number: z.string(),
  email: z.string(),
  full_name: z.string(),
  faculty: z.string(),
  department: z.string(),
  programme: z.string(),
  study_year: z.number().int(),
  status: z.string(),
  source: z.string(),
  import_batch_id: z.string(),
  imported_by_id: z.string(),
  status_reason: z.string(),
  status_changed_at: z.string().nullable(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateElectionRosterSchema = z.object({
  election_id: z.string(),
  student_number: z.string(),
  reg_number: z.string(),
  email: z.string(),
  full_name: z.string(),
  faculty: z.string(),
  department: z.string(),
  programme: z.string(),
  study_year: z.number().int(),
  status: z.string(),
  source: z.string(),
  import_batch_id: z.string(),
  imported_by_id: z.string(),
  status_reason: z.string(),
  status_changed_at: z.string().nullable(),
});

export const UpdateElectionRosterSchema = z.object({
  election_id: z.string().optional(),
  student_number: z.string().optional(),
  reg_number: z.string().optional(),
  email: z.string().optional(),
  full_name: z.string().optional(),
  faculty: z.string().optional(),
  department: z.string().optional(),
  programme: z.string().optional(),
  study_year: z.number().int().optional(),
  status: z.string().optional(),
  source: z.string().optional(),
  import_batch_id: z.string().optional(),
  imported_by_id: z.string().optional(),
  status_reason: z.string().optional(),
  status_changed_at: z.string().nullable(),
});

export type ElectionRoster = z.infer<typeof ElectionRosterSchema>;
export type CreateElectionRosterInput = z.infer<typeof CreateElectionRosterSchema>;
export type UpdateElectionRosterInput = z.infer<typeof UpdateElectionRosterSchema>;
