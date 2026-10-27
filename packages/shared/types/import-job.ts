export interface ImportJob {
  id: string;
  resource: string;
  status: string;
  total: number;
  processed: number;
  created: number;
  skipped: number;
  failed: number;
  message: string;
  created_at: string;
  updated_at: string;
}
