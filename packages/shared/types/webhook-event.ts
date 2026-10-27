export interface WebhookEvent {
  id: string;
  provider: string;
  event_type: string;
  external_id: string | null;
  payload: unknown;
  status: string;
  handler_error: string;
  retry_count: number;
  processed_at: string | null;
  created_at: string;
}
