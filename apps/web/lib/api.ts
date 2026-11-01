import type {
  AuthPayload,
  Ballot,
  BallotContent,
  Candidate,
  CandidateApplicationView,
  Election,
  EligibilityDecision,
  EligibilityReport,
  MyCandidate,
  Position,
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
  fields?: Record<string, string>;

  constructor(message: string, status: number, fields?: Record<string, string>) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.fields = fields;
  }
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
    throw new ApiError(message, response.status, payload?.error?.details);
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
    throw new ApiError(message, response.status);
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
};

export const staff = {
  turnout: (electionId: string) =>
    api<{
      eligible: number;
      voted: number;
      percent: number;
      suppressed: boolean;
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
  transition: (id: string, to: string, reason = "") =>
    api<Election>(`/elections/${id}/transitions`, { method: "POST", body: { to, reason } }),
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
