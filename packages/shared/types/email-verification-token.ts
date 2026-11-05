export interface EmailVerificationToken {
  id: string;
  user_id: string;
  email: string;
  expires_at: string;
  used_at: string | null;
  created_at: string;
}
