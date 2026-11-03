import type {
  AuthPayload,
  Ballot,
  BallotContent,
  Candidate,
  CandidateApplicationView,
  Election,
  ElectionRole,
  ElectionStatus,
  EligibilityDecision,
  EligibilityReport,
  EligiblePosition,
  MyCandidate,
  Position,
  PositionApplications,
  PositionWithMeta,
  RecountResult,
  ResultSnapshot,
  Results,
  ResultsHistory,
  SubmitOptions,
  SubmitResult,
  UploadRef,
  User,
  VoteStatus,
} from "./types";

export const API = (process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080").replace(/\/$/, "");

export class ApiError extends Error {
  status: number;
  /** Machine-readable code from the API envelope, e.g. VALIDATION_ERROR. */
  code?: string;
  fields?: Record<string, string>;
  /** The API's X-Request-ID, which names the matching line in the API log. */
  requestId: string;

  constructor(
    message: string,
    status: number,
    options: { fields?: Record<string, string>; code?: string; requestId?: string } = {}
  ) {
    super(displayMessage(message, status, options));
    this.name = "ApiError";
    this.status = status;
    this.fields = options.fields;
    this.code = options.code;
    this.requestId = options.requestId ?? "";
  }
}

/**
 * The most useful text we can put in front of a person for a failed request.
 *
 * This is built once here rather than in each page: every screen already shows
 * ApiError.message, so all of them gain the detail at the same time instead of
 * each one having to remember to ask for it.
 *
 * The API keeps 500 bodies deliberately generic because their text can describe
 * the schema, so for server errors we append the code and the request id: that
 * id is on the matching line of the API log.
 */
function displayMessage(
  message: string,
  status: number,
  options: { fields?: Record<string, string>; code?: string; requestId?: string }
): string {
  const parts: string[] = [message];
  if (options.fields && Object.keys(options.fields).length > 0) {
    parts.push(
      Object.entries(options.fields)
        .map(([field, problem]) => `${field.replaceAll("_", " ")}: ${problem}`)
        .join("; ")
    );
  }
  if (status >= 500) {
    if (options.code) parts.push(`[${options.code}]`);
    if (options.requestId) parts.push(`ref ${options.requestId}`);
  }
  return parts.join(" — ");
}

/** Double-submit cookie: the API issues grit_csrf and expects it echoed. */
function csrfToken(): string {
  if (typeof document === "undefined") return "";
  const match = document.cookie.match(/(?:^|;\s*)grit_csrf=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : "";
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  signal?: AbortSignal;
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const method = options.method ?? "GET";
  const headers: Record<string, string> = { Accept: "application/json" };
  if (options.body !== undefined) headers["Content-Type"] = "application/json";
  if (method !== "GET" && method !== "HEAD") {
    const token = csrfToken();
    if (token) headers["X-CSRF-Token"] = token;
  }
  const response = await fetch(`${API}/api/v1${path}`, {
    method,
    credentials: "include",
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    signal: options.signal,
  });
  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const message = payload?.error?.message ?? `Request failed (${response.status})`;
    throw new ApiError(message, response.status, {
      fields: payload?.error?.details,
      code: payload?.error?.code,
      requestId: response.headers.get("X-Request-ID") ?? undefined,
    });
  }
  return (payload?.data ?? payload) as T;
}

export const auth = {
  me: () => api<User>("/auth/me"),
  login: (email: string, password: string) =>
    api<AuthPayload>("/auth/login", { method: "POST", body: { email, password } }),
  logout: () => api<unknown>("/auth/logout", { method: "POST" }),
  /** Direct registration: create account and get tokens immediately. */
  register: (body: { first_name: string; last_name: string; email: string; password: string }) =>
    api<AuthPayload>("/auth/register", { method: "POST", body }),
  /** Two-step sign-up: step 1 — send a 6-digit verification code to the email. */
  signupInitiate: (email: string) =>
    api<{ message: string; verification_code?: string }>("/auth/signup/initiate", {
      method: "POST",
      body: { email },
    }),
  /** Two-step sign-up: step 2 — complete with the code and account details. */
  signupComplete: (body: {
    email: string;
    verification_code: string;
    first_name: string;
    last_name: string;
    password: string;
  }) => api<AuthPayload>("/auth/signup/complete", { method: "POST", body }),
  /** Verify email for an already-registered user using a code. */
  verifyEmail: (code: string) =>
    api<{ message: string }>("/auth/verify-email", { method: "POST", body: { code } }),
  /** Re-send the email verification link to the signed-in user. */
  resendVerification: () =>
    api<{ message: string }>("/auth/verify-email/resend", { method: "POST" }),
  /** Invite an administrator (creates account without password, emails reset link). */
  inviteAdmin: (body: { first_name: string; last_name: string; email: string }) =>
    api<{ data: User; message: string }>("/staff/admins/invite", { method: "POST", body }),
};

/** Multipart upload. The API sniffs the real type and enforces accepts/max_size. */
export async function upload(
  file: File,
  options: { accepts?: string; maxSize?: number } = {}
): Promise<UploadRef> {
  const query = new URLSearchParams();
  if (options.accepts) query.set("accepts", options.accepts);
  if (options.maxSize) query.set("max_size", String(options.maxSize));
  const form = new FormData();
  form.append("file", file);
  const headers: Record<string, string> = { Accept: "application/json" };
  const token = csrfToken();
  if (token) headers["X-CSRF-Token"] = token;
  const response = await fetch(`${API}/api/v1/uploads?${query.toString()}`, {
    method: "POST",
    credentials: "include",
    headers,
    body: form,
  });
  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const message = payload?.error?.message ?? `Upload failed (${response.status})`;
    throw new ApiError(message, response.status, {
      fields: payload?.error?.details,
      code: payload?.error?.code,
      requestId: response.headers.get("X-Request-ID") ?? undefined,
    });
  }
  return payload?.data as UploadRef;
}

export const candidates = {
  application: (electionId: string) =>
    api<CandidateApplicationView>(`/elections/${electionId}/candidate-application`),
  apply: (electionId: string, body: Record<string, unknown>) =>
    api<Candidate>(`/elections/${electionId}/candidates/apply`, { method: "POST", body }),
  mine: () => api<{ candidates: MyCandidate[] }>("/candidates/my"),
  withdraw: (candidateId: string, reason: string) =>
    api<Candidate>(`/candidates/my/${candidateId}/withdraw`, { method: "POST", body: { reason } }),
  /**
   * Replace the image shown on the ballot (approved candidates only, before
   * voting opens). When unset the passport photo is used.
   */
  setBallotPhoto: (candidateId: string, ballotPhotoUrl: string) =>
    api<Candidate>(`/candidates/my/${candidateId}/ballot-photo`, {
      method: "POST",
      body: { ballot_photo_url: ballotPhotoUrl },
    }),
  list: (electionId: string, status?: string) =>
    api<{ candidates: Candidate[] }>(
      `/elections/${electionId}/candidates${status ? `?status=${status}` : ""}`
    ),
  approve: (electionId: string, candidateId: string, reason: string) =>
    api<Candidate>(`/elections/${electionId}/candidates/${candidateId}/approve`, {
      method: "POST",
      body: { reason },
    }),
  reject: (electionId: string, candidateId: string, reason: string) =>
    api<Candidate>(`/elections/${electionId}/candidates/${candidateId}/reject`, {
      method: "POST",
      body: { reason },
    }),
};

export const positions = {
  list: (electionId: string) =>
    api<{ positions: Position[] }>(`/elections/${electionId}/positions`),
  create: (electionId: string, body: Record<string, unknown>) =>
    api<Position>(`/elections/${electionId}/positions`, { method: "POST", body }),
  update: (electionId: string, positionId: string, body: Record<string, unknown>) =>
    api<Position>(`/elections/${electionId}/positions/${positionId}`, { method: "PUT", body }),
  remove: (electionId: string, positionId: string) =>
    api<unknown>(`/elections/${electionId}/positions/${positionId}`, { method: "DELETE" }),

  // Standalone (positions-first) endpoints. A position exists before any
  // election does; these paths are not election-scoped.
  listAll: (filter: { electionId?: string; unlinked?: boolean; status?: string } = {}) => {
    const query = new URLSearchParams();
    if (filter.electionId) query.set("election_id", filter.electionId);
    if (filter.unlinked) query.set("unlinked", "1");
    if (filter.status) query.set("status", filter.status);
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return api<{ positions: PositionWithMeta[] }>(`/positions${suffix}`);
  },
  createStandalone: (body: Record<string, unknown>) =>
    api<Position>("/positions", { method: "POST", body }),
  updateStandalone: (positionId: string, body: Record<string, unknown>) =>
    api<Position>(`/positions/${positionId}`, { method: "PUT", body }),
  removeStandalone: (positionId: string) =>
    api<unknown>(`/positions/${positionId}`, { method: "DELETE" }),
  applications: (positionId: string, status?: string) =>
    api<PositionApplications>(
      `/positions/${positionId}/applications${status ? `?status=${status}` : ""}`
    ),
  approve: (positionId: string, candidateId: string, reason: string) =>
    api<Candidate>(`/positions/${positionId}/applications/${candidateId}/approve`, {
      method: "POST",
      body: { reason },
    }),
  reject: (positionId: string, candidateId: string, reason: string) =>
    api<Candidate>(`/positions/${positionId}/applications/${candidateId}/reject`, {
      method: "POST",
      body: { reason },
    }),
  link: (positionId: string, electionId: string) =>
    api<Position>(`/positions/${positionId}/link`, {
      method: "POST",
      body: { election_id: electionId },
    }),

  // Applicant-facing.
  eligible: () => api<{ positions: EligiblePosition[] }>("/positions/eligible"),
  apply: (positionId: string, body: Record<string, unknown>) =>
    api<Candidate>(`/positions/${positionId}/apply`, { method: "POST", body }),
};

export const staff = {
  turnout: (electionId: string) =>
    api<{
      eligible: number;
      voted: number;
      percent: number;
      suppressed: boolean;
      positions?: Array<{
        position_id: string;
        title: string;
        ballots_answered: number;
        blank_count: number;
        abstain_count: number;
      }>;
      last_refresh: string;
    }>(`/elections/${electionId}/turnout`),
  versions: (electionId: string) =>
    api<{
      versions: Array<{
        id: string;
        version: number;
        content_hash: string;
        status: string;
        published_by: string;
        published_at: string;
      }>;
    }>(`/elections/${electionId}/ballot/versions`),
  version: (electionId: string, versionId: string) =>
    api<{ content: BallotContent }>(`/elections/${electionId}/ballot/versions/${versionId}`),
  compare: (electionId: string, v1: string, v2: string) =>
    api<{
      same_hash: boolean;
      v1: {
        id: string;
        version: number;
        content_hash: string;
        status: string;
        published_by: string;
        published_at: string;
      };
      v2: {
        id: string;
        version: number;
        content_hash: string;
        status: string;
        published_by: string;
        published_at: string;
      };
    }>(`/elections/${electionId}/ballot/versions/compare?v1=${v1}&v2=${v2}`),
  publish: (electionId: string) =>
    api<{ version: number; content_hash: string }>(`/elections/${electionId}/ballot/publish`, {
      method: "POST",
    }),
};

export const elections = {
  list: () => api<Election[]>("/elections"),
  get: (id: string) => api<Election>(`/elections/${id}`),
  create: (body: Record<string, unknown>) => api<Election>("/elections", { method: "POST", body }),
  update: (id: string, body: Record<string, unknown>) =>
    api<Election>(`/elections/${id}`, { method: "PUT", body }),
  /**
   * Delete an election that has not yet collected a ballot. The server refuses
   * with 409 once voting has started; close it instead.
   */
  remove: (id: string) => api<unknown>(`/elections/${id}`, { method: "DELETE" }),
  transition: (id: string, to: string, reason = "") =>
    api<Election>(`/elections/${id}/transitions`, { method: "POST", body: { to, reason } }),
  readiness: (id: string) =>
    api<{
      ready: boolean;
      status: ElectionStatus;
      problems: Record<string, string>;
    }>(`/elections/${id}/readiness`),
  roles: {
    list: (id: string) => api<ElectionRole[]>(`/elections/${id}/roles`),
    lookup: (id: string, email: string) =>
      api<{ user_id: string; email: string; full_name: string; election_id: string }>(
        `/elections/${id}/roles/lookup?email=${encodeURIComponent(email)}`
      ),
    assign: (id: string, userId: string, role: string) =>
      api<ElectionRole>(`/elections/${id}/roles/${userId}`, {
        method: "PUT",
        body: { role },
      }),
    remove: (id: string, userId: string, role: string) =>
      api<{ removed: boolean }>(`/elections/${id}/roles/${userId}/${role}`, {
        method: "DELETE",
      }),
  },
  eligibility: (id: string) => api<EligibilityDecision>(`/elections/${id}/eligibility`),
  report: (id: string) => api<EligibilityReport>(`/elections/${id}/eligibility/report`),
  freeze: (id: string) =>
    api<{ frozen_at: string; frozen_count: number }>(`/elections/${id}/eligibility/freeze`, {
      method: "POST",
    }),
  ballot: (id: string) => api<Ballot>(`/elections/${id}/ballot`),
  startBallot: (id: string) =>
    api<{
      eligible: boolean;
      starts_at: string;
      ends_at: string;
      ballot_version_id?: string | null;
    }>(`/elections/${id}/ballot/start`, { method: "POST" }),
  submitBallot: (id: string, selections: Array<Record<string, unknown>>, options: SubmitOptions) =>
    api<SubmitResult>(`/elections/${id}/ballot/submit`, {
      method: "POST",
      body: {
        selections,
        ballot_version_id: options.ballotVersionId ?? undefined,
        idempotency_key: options.idempotencyKey,
      },
    }),
  voteStatus: (id: string) => api<VoteStatus>(`/elections/${id}/my-vote-status`),
  results: (id: string) => api<Results>(`/elections/${id}/results`),
  resultsHistory: (id: string) => api<ResultsHistory>(`/elections/${id}/results/history`),
  runTally: (id: string, reason?: string) =>
    api<{ election_id: string; snapshot: ResultSnapshot; correction: boolean }>(
      `/elections/${id}/results/tally`,
      { method: "POST", body: reason ? { reason } : {} }
    ),
  recount: (id: string, snapshotId: string, reason: string) =>
    api<RecountResult>(`/elections/${id}/results/recount`, {
      method: "POST",
      body: { snapshot_id: snapshotId, reason },
    }),
  certify: (id: string, snapshotId: string, reason: string) =>
    api<{ election_id: string; status: string; snapshot: ResultSnapshot }>(
      `/elections/${id}/results/certify`,
      { method: "POST", body: { snapshot_id: snapshotId, reason } }
    ),
};
