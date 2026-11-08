export type Role = "ADMIN" | "STUDENT_VOTER";

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

export interface AuthChallenge {
  totp_required: true;
  pending_token: string;
  method?: string;
}

/**
 * The election lifecycle has three stages. DRAFT is private while it is being
 * built; the server opens it automatically at its start time; it closes at its
 * end time. There is no separate scheduled or finalised state.
 */
export type ElectionStatus = "DRAFT" | "OPEN" | "CLOSED";

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
  /** @deprecated use allowed_domain instead */
  allowed_domains?: string;
  allowed_domain: string;
  allowed_years?: string;
  allowed_courses?: string;
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
  allowed_transitions: ElectionStatus[] | null;
  my_vote_status?: VoteStatus;
}

export interface ElectionRole {
  user_id: string;
  email?: string;
  full_name?: string;
  role: string;
  created_at: string;
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
  slogan?: string;
  manifesto: string;
  manifesto_url?: string;
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

export interface BallotContent {
  positions: BallotPosition[];
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

// Participation split by faculty. Counts only: the server never sends a name,
// an address or anything that could be joined back to a choice, and it
// withholds the figures for a group smaller than the reporting threshold.
export interface FacultyTurnout {
  faculty: string;
  eligible: number;
  ballots: number;
  percent: number;
  suppressed?: boolean;
}

export interface Results {
  election_id: string;
  election_status?: ElectionStatus;
  available: boolean;
  label: ResultStatus | "UNAVAILABLE";
  reason?: string;
  snapshot?: ResultSnapshot;
  turnout_by_faculty?: FacultyTurnout[];
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

export type CandidateStatus = "NOMINATED" | "PENDING" | "APPROVED" | "REJECTED" | "WITHDRAWN";

export interface Candidate {
  id: string;
  election_id: string;
  position_id: string;
  full_name: string;
  slogan?: string;
  photo_url?: string;
  manifesto?: string;
  manifesto_url?: string;
  bio?: string;
  contact_email?: string;
  contact_phone?: string;
  ballot_number: number;
  approval_status: CandidateStatus | string;
  review_notes?: string;
  nominated_by?: string;
  reviewed_at?: string | null;
  withdrawn_at?: string | null;
  withdrawal_reason?: string;
  date_of_birth?: string | null;
  registration_number?: string;
  campus_residence?: string;
  sex?: string;
  religion?: string;
  id_document_url?: string;
  passport_photo_url?: string;
  /** Replacement image an approved candidate uploads for the ballot. */
  ballot_photo_url?: string;
  partial_transcript_url?: string;
  programme?: string;
  study_year?: number;
  created_at: string;
  updated_at: string;
}

export interface MyCandidate {
  candidate: Candidate;
  election_title: string;
  election_status: ElectionStatus;
  election_starts_at: string;
  election_ends_at: string;
  position_title: string;
  /**
   * The candidate's own line from the published count. Present only once the
   * election has closed; it is the candidate's figure, never about a voter.
   */
  result?: CandidacyResult;
}

/** A candidate's own result in a closed election, from the stored snapshot. */
export interface CandidacyResult {
  position_title: string;
  votes: number;
  share_percent: number;
  is_winner: boolean;
  is_tie: boolean;
  snapshot_status: ResultStatus | string;
}

export type PositionStatus = "OPEN" | "CLOSED";

export interface Position {
  id: string;
  /** Null while the position exists before being attached to an election. */
  election_id: string | null;
  title: string;
  description: string;
  requirements?: string;
  question_type: string;
  /** Whether the position is still accepting applications. */
  status: PositionStatus | string;
  /** Cap on approved candidates; 0 means no limit. */
  max_candidates: number;
  /** Candidate eligibility rule: required domain plus optional year/course lists. */
  allowed_domain?: string;
  allowed_years?: string;
  allowed_courses?: string;
  min_selections: number;
  max_selections: number;
  seats: number;
  allow_blank: boolean;
  display_order: number;
}

/** A position on the admin positions page, with its election name and counts. */
export interface PositionWithMeta extends Position {
  election_title?: string;
  application_count: number;
  approved_count: number;
}

/** A position as an applicant sees it: their own rule plus whether they applied. */
export interface EligiblePosition extends Position {
  election_title?: string;
  applied: boolean;
}

/** A row on the admin application-review screen. */
export interface PositionApplications {
  position: Position;
  applications: Candidate[];
  approved_count: number;
}

export interface CandidateApplicationView {
  election: {
    id: string;
    title: string;
    status: ElectionStatus;
    starts_at: string;
    ends_at: string;
    category: string;
    unit: string;
  };
  positions: Position[];
  applicant?: {
    programme?: string;
    study_year?: number;
    registration_number?: string;
  };
  eligible: boolean;
  eligibility_reason?: string;
  accepting_applications: boolean;
  application_reason?: string;
}

export interface UploadRef {
  url: string;
  key: string;
  name: string;
  mime: string;
  size: number;
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
