export interface AnonymousBallot {
  id: string;
  election_id: string;
  status: string;
  cast_at: string;
  selections: unknown[];
}
