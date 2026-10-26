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
  created_at: string;
  updated_at: string;
}
