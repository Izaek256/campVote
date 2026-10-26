import { z } from "zod";

export const DashboardLayoutSchema = z.object({
  id: z.string(),
  user_id: z.string(),
  cards: z.array(z.string()).optional(),
  charts: z.array(z.string()).optional(),
  tables: z.array(z.string()).optional(),
  resources: z.array(z.string()).optional(),
  section_order: z.array(z.string()).optional(),
  resource_layouts: z.unknown(),
  custom_charts: z.unknown(),
  date_preset: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
});

export const CreateDashboardLayoutSchema = z.object({
  user_id: z.string(),
  cards: z.array(z.string()).optional(),
  charts: z.array(z.string()).optional(),
  tables: z.array(z.string()).optional(),
  resources: z.array(z.string()).optional(),
  section_order: z.array(z.string()).optional(),
  resource_layouts: z.unknown(),
  custom_charts: z.unknown(),
  date_preset: z.string(),
});

export const UpdateDashboardLayoutSchema = z.object({
  user_id: z.string().optional(),
  cards: z.array(z.string()).optional(),
  charts: z.array(z.string()).optional(),
  tables: z.array(z.string()).optional(),
  resources: z.array(z.string()).optional(),
  section_order: z.array(z.string()).optional(),
  resource_layouts: z.unknown().optional(),
  custom_charts: z.unknown().optional(),
  date_preset: z.string().optional(),
});

export type DashboardLayout = z.infer<typeof DashboardLayoutSchema>;
export type CreateDashboardLayoutInput = z.infer<typeof CreateDashboardLayoutSchema>;
export type UpdateDashboardLayoutInput = z.infer<typeof UpdateDashboardLayoutSchema>;
