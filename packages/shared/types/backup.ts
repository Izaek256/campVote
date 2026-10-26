export interface Backup {
  id: string;
  kind: string;
  status: string;
  size_bytes: number;
  table_count: number;
  row_count: number;
  error: string;
  created_at: string;
  completed_at: string | null;
}
