export type Role =
  | "ADMIN"
  | "EDITOR"
  | "USER"
  | "CHIEF_ELECTION_ADMIN"
  | "ELECTION_OFFICIAL"
  | "STUDENT_VOTER"
  | "STAFF_VOTER"
  | "CANDIDATE"
  | "ELECTION_AUDITOR";

export interface User {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
  role: Role | string;
  active: boolean;
  email_verified_at?: string | null;
  created_at: string;
}

export interface AuthPayload {
  user: User;
  tokens: { access_token: string; refresh_token: string; expires_at: number };
}

export type ElectionStatus =
  | "DRAFT"
  | "REGISTRATION"
  | "PENDING_APPROVAL"
  | "SCHEDULED"
  | "OPEN"
  | "CLOSED"
  | "RESULTS_UNDER_REVIEW"
  | "FINALIZED"
  | "ARCHIVED"
  | "CANCELLED";

export interface Election {
  id: string;
  title: string;
  description: string;
  category: string;
  organizing_unit: string;
  status: ElectionStatus;
  starts_at: string;
  ends_at: string;
  result_policy: "LIVE" | "AFTER_CLOSE" | "CERTIFIED_ONLY";
  allowed_domains: string;
  eligibility_frozen_at?: string | null;
  eligibility_frozen_count: number;
  created_by_id: string;
  version: number;
  created_at: string;
  updated_at: string;
  phase: ElectionStatus;
  server_time: string;
  accepting_votes: boolean;
  display_timezone: string;
  allowed_transitions: ElectionStatus[];
}

export interface EligibilityDecision {
  eligible: boolean;
  reason?: string;
}

export interface BallotCandidate {
  id: string;
  full_name: string;
  manifesto: string;
  ballot_number: number;
}

export interface BallotPosition {
  id: string;
  title: string;
  description: string;
  question_type: "SINGLE_CHOICE" | "MULTI_CHOICE" | "REFERENDUM" | string;
  min_selections: number;
  max_selections: number;
  allow_blank: boolean;
  display_order: number;
  candidates: BallotCandidate[];
}

export interface Ballot {
  election_id: string;
  title: string;
  starts_at: string;
  ends_at: string;
  positions: BallotPosition[];
}

export interface SubmitResult {
  election_id: string;
  confirmation_token: string;
  message: string;
}

export interface Tally {
  election_id: string;
  position_id: string;
  candidate_id?: string;
  choice: string;
  votes: number;
  valid_votes_total: number;
  percentage: number;
  turnout_percentage: number;
  is_winner: boolean;
  is_tie: boolean;
  candidate_name?: string;
  position_title?: string;
}

export interface Results {
  election_id: string;
  status: ElectionStatus;
  label: "PROVISIONAL" | "CERTIFIED";
  tallies: Tally[];
}

export interface EligibilityReport {
  election_id: string;
  status: ElectionStatus;
  total: number;
  eligible: number;
  by_status: Record<string, number>;
  participated: number;
  turnout_percent: number;
  frozen_at?: string;
  frozen_eligible: number;
  last_import_at?: string;
  last_import_batch_id?: string;
}
