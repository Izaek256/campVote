export interface SagaRun {
  id: string;
  name: string;
  key: string | null;
  input: unknown;
  state: unknown;
  status: string;
  cursor: number;
  attempts: number;
  last_error: string;
  available_at: string;
  claimed_by: string;
  claimed_at: string | null;
  finished_at: string | null;
  steps: unknown[];
  created_at: string;
  updated_at: string;
}
