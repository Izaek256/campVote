export interface MagicLinkToken {
  id: number;
  user_id: string;
  used_at: string | null;
  ip_address: string;
  user_agent: string;
  expires_at: string;
  created_at: string;
}
