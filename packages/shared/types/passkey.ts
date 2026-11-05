export interface Passkey {
  id: string;
  user_id: string;
  credential_id: string;
  transports: string;
  backup_eligible: boolean;
  synced: boolean;
  name: string;
  last_used_at: string | null;
  created_at: string;
  updated_at: string;
}
