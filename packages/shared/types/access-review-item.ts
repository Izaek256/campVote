export interface AccessReviewItem {
  id: string;
  review_id: string;
  user_id: string;
  user_email: string;
  role_id: string;
  role_name: string;
  decision: string;
  decided_by: string;
  decided_by_email: string;
  decided_at: string | null;
  note: string;
  created_at: string;
}
