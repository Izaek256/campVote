export interface AccessReview {
  id: string;
  name: string;
  note: string;
  status: string;
  created_by: string;
  created_by_email: string;
  completed_by: string;
  completed_by_email: string;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
  items: unknown[];
}
