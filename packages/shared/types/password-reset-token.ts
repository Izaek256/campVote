export interface PasswordResetToken {
  id: string;
  user_id: string;
  request_ip: string;
  expires_at: string;
  used_at: string | null;
  created_at: string;
}
