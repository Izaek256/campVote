import type {
  AuthPayload,
  Ballot,
  Election,
  EligibilityDecision,
  EligibilityReport,
  Results,
  SubmitOptions,
  SubmitResult,
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
  certify: (id: string) => api<{ status: string }>(`/elections/${id}/certify`, { method: "POST" }),
};
