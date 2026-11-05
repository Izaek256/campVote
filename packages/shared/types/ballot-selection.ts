export interface BallotSelection {
  id: string;
  ballot_id: string;
  position_id: string;
  candidate_id: string | null;
  choice: string;
  created_at: string;
}
