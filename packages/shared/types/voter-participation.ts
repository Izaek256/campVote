export interface VoterParticipation {
  id: string;
  election_id: string;
  user_id: string;
  confirmation_token: string;
  voted_at: string;
}
