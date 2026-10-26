export interface UserActivity {
  id: string;
  user_id: string;
  action: string;
  severity: string;
  summary: string;
  resource_type: string;
  resource_id: string;
  ip_address: string;
  user_agent: string;
  metadata: string;
  created_at: string;
}
