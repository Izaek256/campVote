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
  my_vote_status?: VoteStatus;
}

export interface VoteStatus {
  election_id: string;
  voted: boolean;
  recorded_at?: string;
  phase?: ElectionStatus;
  server_time?: string;
}

export interface EligibilityDecision {
  eligible: boolean;
  reason?: string;
}

export interface BallotCandidate {
  id: string;
  full_name: string;
  manifesto: string;
  photo_url?: string;
  bio?: string;
  ballot_number: number;
}

export interface BallotPosition {
  id: string;
  title: string;
  description: string;
  question_type: "SINGLE_CHOICE" | "MULTI_CHOICE" | "REFERENDUM" | string;
  min_selections: number;
  max_selections: number;
  seats?: number;
  allow_blank: boolean;
  display_order: number;
  candidates: BallotCandidate[];
}

export interface Ballot {
  election_id: string;
  title: string;
  starts_at: string;
  ends_at: string;
  ballot_version_id?: string | null;
  ballot_version?: number | null;
  positions: BallotPosition[];
}

export interface SubmitResult {
  election_id: string;
  confirmation_token: string;
  message: string;
}

export interface SubmitOptions {
  ballotVersionId?: string | null;
  idempotencyKey: string;
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

// ISSUE-09 results. Every share, outcome and turnout figure is computed by the
// server and stored on the snapshot; the client renders it and never re-derives
// it. A leading candidate is not a winner unless `outcome` says so.
export type ResultOutcome = "WINNER" | "TIE_UNRESOLVED" | "QUORUM_NOT_MET" | "NO_VOTES";
export type ResultStatus = "PROVISIONAL" | "CERTIFIED" | "SUPERSEDED";

export interface ResultRow {
  candidate_id: string;
  name: string;
  votes: number;
  share_percent: number;
  is_winner: boolean;
  is_tie: boolean;
}

export interface PositionResult {
  position_id: string;
  title: string;
  question_type: string;
  seats: number;
  rows: ResultRow[];
  valid_votes: number;
  blank_count: number;
  abstain_count: number;
  ballots_answered: number;
  quorum_percent: number;
  quorum_met: boolean;
  outcome: ResultOutcome;
  winners: string[];
  tied_candidates?: string[];
}

export interface Turnout {
  eligible_voters: number;
  ballots_cast: number;
  percent: number;
  known: boolean;
}

export interface ResultSnapshot {
  id: string;
  version: number;
  status: ResultStatus;
  content_hash: string;
  computed_at: string;
  computed_by_id: string;
  certified_by_id?: string;
  certified_at?: string;
  certify_reason?: string;
  supersedes_id?: string;
  superseded_by_id?: string;
  eligible_voters: number;
  ballots_cast: number;
  turnout_percent: number;
  turnout: Turnout;
  positions: PositionResult[];
  ballots_spoiled: number;
}

export interface Results {
  election_id: string;
  election_status?: ElectionStatus;
  available: boolean;
  label: ResultStatus | "UNAVAILABLE";
  reason?: string;
  snapshot?: ResultSnapshot;
}

export interface ResultsHistory {
  election_id: string;
  snapshots: ResultSnapshot[];
  recounts: ResultRecount[];
}

export interface CandidateDiff {
  candidate_id: string;
  name: string;
  before: number;
  after: number;
  delta: number;
}

export interface PositionDiff {
  position_id: string;
  title: string;
  outcome_before: ResultOutcome;
  outcome_after: ResultOutcome;
  candidates?: CandidateDiff[];
}

export interface ResultDifferences {
  identical: boolean;
  position_diffs?: PositionDiff[];
  turnout_diff?: { before: number; after: number };
}

export interface ResultRecount {
  id: string;
  election_id: string;
  compared_snapshot_id: string;
  recomputed_hash: string;
  identical: boolean;
  differences?: string;
  reason: string;
  run_by_id: string;
  run_at: string;
}

export interface RecountResult {
  recount: ResultRecount;
  comparison_version: number;
  identical: boolean;
  differences: ResultDifferences;
  ballots_unchanged: boolean;
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
