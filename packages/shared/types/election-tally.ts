export interface ElectionTally {
  id: string;
  election_id: string;
  position_id: string;
  candidate_id: string | null;
  choice: string;
  votes: number;
  valid_votes_total: number;
  percentage: number;
  turnout_percentage: number;
  is_winner: boolean;
  is_tie: boolean;
  computed_at: string;
}
