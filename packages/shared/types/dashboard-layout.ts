export interface DashboardLayout {
  id: string;
  user_id: string;
  cards: string[];
  charts: string[];
  tables: string[];
  resources: string[];
  section_order: string[];
  resource_layouts: unknown;
  custom_charts: unknown;
  date_preset: string;
  created_at: string;
  updated_at: string;
}
