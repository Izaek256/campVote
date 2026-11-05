export interface RecoveryContactToken {
  id: string;
  user_id: string;
  kind: unknown;
  destination: string;
  expires_at: string;
  used_at: string | null;
  created_at: string;
}
