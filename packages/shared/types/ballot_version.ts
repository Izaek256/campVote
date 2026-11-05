export interface BallotVersion {
  id: string;
  election_id: string;
  version: number;
  content_hash: string;
  content_json: string;
  published_by: string;
  published_at: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface BallotContentCandidate {
  id: string;
  full_name: string;
  photo_url: string;
  manifesto: string;
  bio: string;
  ballot_number: number;
}

export interface BallotContentPosition {
  id: string;
  title: string;
  description: string;
  question_type: string;
  min_selections: number;
  max_selections: number;
  seats: number;
  allow_blank: boolean;
  display_order: number;
  candidates: BallotContentCandidate[];
}

export interface BallotContent {
  positions: BallotContentPosition[];
}
