export interface UserIdentity {
  id: string;
  user_id: string;
  provider: string;
  subject: string;
  email: string;
  last_login_at: string | null;
  created_at: string;
  updated_at: string;
}
