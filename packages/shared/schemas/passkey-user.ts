import { z } from "zod";

export const PasskeyUserSchema = z.object({
  user: z.unknown(),
  credentials: z.unknown(),
});

export const CreatePasskeyUserSchema = z.object({
  user: z.unknown(),
  credentials: z.unknown(),
});

export const UpdatePasskeyUserSchema = z.object({
  user: z.unknown().optional(),
  credentials: z.unknown().optional(),
});

export type PasskeyUser = z.infer<typeof PasskeyUserSchema>;
export type CreatePasskeyUserInput = z.infer<typeof CreatePasskeyUserSchema>;
export type UpdatePasskeyUserInput = z.infer<typeof UpdatePasskeyUserSchema>;
