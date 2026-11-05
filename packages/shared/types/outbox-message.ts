export interface OutboxMessage {
  id: string;
  topic: string;
  key: string | null;
  payload: unknown;
  status: string;
  attempts: number;
  last_error: string;
  available_at: string;
  claimed_by: string;
  claimed_at: string | null;
  delivered_at: string | null;
  created_at: string;
  updated_at: string;
}
