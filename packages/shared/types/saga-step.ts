export interface SagaStep {
  id: string;
  run_id: string;
  idx: number;
  name: string;
  status: string;
  attempts: number;
  last_error: string;
  started_at: string | null;
  finished_at: string | null;
  created_at: string;
  updated_at: string;
}
