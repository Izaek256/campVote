export interface Position {
  id: string;
  election_id: string;
  title: string;
  description: string;
  question_type: string;
  min_selections: number;
  max_selections: number;
  seats: number;
  allow_blank: boolean;
  display_order: number;
  created_at: string;
  updated_at: string;
  candidates: unknown[];
}
