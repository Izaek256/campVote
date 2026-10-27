export interface ElectionAuditLog {
  id: string;
  election_id: string;
  actor_id: string;
  action: string;
  details: string;
  ip_address: string;
  created_at: string;
}
