export interface Candidate {
  id: string;
  election_id: string;
  position_id: string;
  full_name: string;
  photo_url: string;
  manifesto: string;
  bio: string;
  ballot_number: number;
  approval_status: string;
  review_notes: string;
  nominated_by?: string;
  reviewed_by?: string;
  reviewed_at?: string | null;
  withdrawn_at?: string | null;
  withdrawal_reason?: string;
  created_at: string;
  updated_at: string;
}
