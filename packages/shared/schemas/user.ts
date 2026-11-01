import { z } from "zod";

/** Editable personal fields on a user's own profile. */
export const PersonalInfoSchema = z.object({
  first_name: z.string().min(2, "First name must be at least 2 characters"),
  last_name: z.string().min(2, "Last name must be at least 2 characters"),
  email: z.string().email("Please enter a valid email address"),
  /** Only required when the email field has changed. */
  current_password: z.string().optional(),
});

export type PersonalInfoInput = z.infer<typeof PersonalInfoSchema>;

/** Optional professional fields stored alongside the profile. */
export const ProfessionalInfoSchema = z.object({
  job_title: z.string().max(100, "Job title must be at most 100 characters").optional(),
  bio: z.string().max(500, "Bio must be at most 500 characters").optional(),
});

export type ProfessionalInfoInput = z.infer<typeof ProfessionalInfoSchema>;
