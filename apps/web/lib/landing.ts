import type {
  LandingDocument,
  LandingIconKey,
  PublicElectionStatus,
  PublicUpcomingElection,
} from "@campvote/shared/types/index";
import { API } from "./api";

/**
 * The landing page's data layer.
 *
 * The public page is server-rendered and revalidated every minute. If the API
 * is unreachable it renders this built-in document instead, so a visitor never
 * sees an empty shop window. The admin editor saves a draft; only the published
 * snapshot is ever read here.
 */

/** The eight illustrations the CMS may pick from, in display order. */
export const LANDING_ICONS: { key: LandingIconKey; label: string }[] = [
  { key: "elections", label: "Many elections" },
  { key: "applications", label: "Candidate applications" },
  { key: "builder", label: "Ballot builder" },
  { key: "eligibility", label: "Eligibility rules" },
  { key: "roles", label: "Clear roles" },
  { key: "turnout", label: "Live turnout" },
  { key: "results", label: "Certified results" },
  { key: "notifications", label: "Email notifications" },
];

/** Mirrors the server's bounds so the editor stops before the API has to. */
export const LANDING_LIMITS = {
  ticker: 12,
  steps: 6,
  features: 12,
  faqs: 20,
} as const;

/** Mirrors `internal/landing` field limits, used as input maxLength hints. */
export const LANDING_MAX = {
  headline: 160,
  subtext: 400,
  label: 60,
  bannerText: 200,
  link: 500,
  tickerItem: 60,
  stepTitle: 80,
  stepText: 200,
  featureTitle: 60,
  featureText: 200,
  question: 200,
  answer: 1000,
  footerText: 300,
} as const;

/**
 * Mirrors `internal/landing.Default()`. Keep the two in step: a fresh install
 * and an API outage must show the same complete page. The asterisks around one
 * word are the hero's only formatting: the renderer colours that word green and
 * never interprets any other markup.
 */
export const DEFAULT_LANDING: LandingDocument = {
  hero: {
    headline: "Vote once.\nKeep it *secret.*\nTrust the result.",
    subtext:
      "campVote runs guild and campus elections end to end: eligibility, secret ballots, live turnout, and certified results, on any phone.",
    primary_label: "Sign in with campus Google",
    secondary_label: "See how voting works",
  },
  banner: {
    enabled: true,
    text: "Voting is open · check the elections you are eligible for",
    link: "/elections",
    starts_at: null,
    ends_at: null,
  },
  ticker: [
    "One voter",
    "One ballot",
    "Secret by design",
    "Auditable",
    "Mobile-first",
    "Certified results",
  ],
  how: {
    steps: [
      {
        title: "Sign in",
        text: "Use your campus Google account. Staff use a separate sign-in.",
      },
      {
        title: "We check eligibility",
        text: "Your place on the voter roster is verified for each election.",
      },
      {
        title: "Choose",
        text: "Pick one candidate per position, or leave a position blank.",
      },
      {
        title: "Confirm and keep your receipt",
        text: "Your ballot is recorded. The receipt never shows your choices.",
      },
    ],
  },
  features: [
    {
      icon: "elections",
      title: "Many elections at once",
      text: "Guild, faculty, class or club polls run side by side, each with its own roster and results.",
    },
    {
      icon: "applications",
      title: "Candidate applications",
      text: "Upload documents, get reviewed and approved by administrators, then appear on the ballot.",
    },
    {
      icon: "builder",
      title: "Custom ballot builder",
      text: "Positions, options and instructions configured per election, then versioned and locked.",
    },
    {
      icon: "eligibility",
      title: "Eligibility rules",
      text: "Restrict by faculty, programme, year or an imported roster. Only eligible voters ever see the ballot.",
    },
    {
      icon: "roles",
      title: "Clear roles",
      text: "Administrators, election officers, candidates, observers and voters each see only their part.",
    },
    {
      icon: "turnout",
      title: "Live turnout",
      text: "Watch participation by faculty as voting runs, with results hidden until the policy says otherwise.",
    },
    {
      icon: "results",
      title: "Certified results and PDFs",
      text: "Review, certify and export a signed-off results report, plus an information sheet for each election.",
    },
    {
      icon: "notifications",
      title: "Email notifications",
      text: "Codes, invitations, reminders and receipts, sent in a consistent campVote style.",
    },
  ],
  faqs: [
    {
      question: "Can anyone see how I voted?",
      answer:
        "No. Officials, candidates and administrators cannot see individual choices. The system stores that you voted separately from the ballot itself.",
    },
    {
      question: "Who can vote?",
      answer:
        "Students on the voter roster for that election, signed in with their campus Google account. Eligibility is checked on the server every time.",
    },
    {
      question: "When are results published?",
      answer:
        "Each election has its own visibility policy. Final results are shown after review and certification by the election administrators.",
    },
    {
      question: "How do administrators sign in?",
      answer: "Administrators have a separate staff sign-in reachable from the footer.",
    },
  ],
  footer: {
    text: "A university project. Built for campus elections.",
    support_email: "",
  },
};

/**
 * The API base for a server render.
 *
 * `NEXT_PUBLIC_API_URL` is the browser's base and is empty when a deployment
 * proxies `/api`; a server component cannot fetch a relative URL, so it also
 * accepts API_INTERNAL_URL / API_PROXY_TARGET. With none set we skip the fetch
 * and use the built-in document rather than guessing a host.
 */
function serverApiBase(): string {
  return (
    process.env.API_INTERNAL_URL ||
    process.env.NEXT_PUBLIC_API_URL ||
    process.env.API_PROXY_TARGET ||
    ""
  ).replace(/\/$/, "");
}

function unwrap(payload: unknown): unknown {
  if (payload && typeof payload === "object" && "data" in payload) {
    return (payload as { data: unknown }).data;
  }
  return payload;
}

/** Keep only a shape the renderer can safely walk; anything odd falls back. */
function isDocument(value: unknown): value is LandingDocument {
  if (!value || typeof value !== "object") return false;
  const v = value as Partial<LandingDocument>;
  return Boolean(v.hero && v.banner && v.how && Array.isArray(v.features));
}

export async function fetchLanding(): Promise<LandingDocument> {
  const base = serverApiBase();
  if (!base) return DEFAULT_LANDING;
  try {
    const response = await fetch(`${base}/api/v1/public/landing`, {
      headers: { Accept: "application/json" },
      next: { revalidate: 60 },
    });
    if (!response.ok) return DEFAULT_LANDING;
    const doc = unwrap(await response.json());
    return isDocument(doc) ? doc : DEFAULT_LANDING;
  } catch {
    // An unreachable API is a content outage, not a page error.
    return DEFAULT_LANDING;
  }
}

/**
 * The one mapping from the public endpoint's fields to the shared type. Both
 * the server render and the client retry go through it, so a field rename is a
 * single-line change.
 */
export function mapUpcomingElection(raw: unknown): PublicUpcomingElection | null {
  if (!raw || typeof raw !== "object") return null;
  const e = raw as Record<string, unknown>;
  const id = typeof e.id === "string" ? e.id : "";
  const title = typeof e.title === "string" ? e.title : "";
  const opens = typeof e.opens_at === "string" ? e.opens_at : "";
  const closes = typeof e.closes_at === "string" ? e.closes_at : "";
  if (!id || !title || !opens || !closes) return null;
  const status: PublicElectionStatus = e.status === "registration" ? "registration" : "scheduled";
  return {
    id,
    title,
    category: typeof e.category === "string" ? e.category : "",
    organizing_unit: typeof e.organizing_unit === "string" ? e.organizing_unit : "",
    positions: typeof e.positions === "number" ? e.positions : 0,
    opens_at: opens,
    closes_at: closes,
    status,
    eligible_count: typeof e.eligible_count === "number" ? e.eligible_count : undefined,
  };
}

export function mapUpcomingElections(payload: unknown): PublicUpcomingElection[] {
  const list = unwrap(payload);
  if (!Array.isArray(list)) return [];
  return list
    .map(mapUpcomingElection)
    .filter((e): e is PublicUpcomingElection => e !== null)
    .sort((a, b) => new Date(a.opens_at).getTime() - new Date(b.opens_at).getTime());
}

/** Server-side initial list. An error is passed to the client, which can retry. */
export async function fetchUpcoming(): Promise<{
  elections: PublicUpcomingElection[];
  failed: boolean;
}> {
  const base = serverApiBase();
  if (!base) return { elections: [], failed: true };
  try {
    const response = await fetch(`${base}/api/v1/public/elections/upcoming`, {
      headers: { Accept: "application/json" },
      next: { revalidate: 60 },
    });
    if (!response.ok) return { elections: [], failed: true };
    return {
      elections: mapUpcomingElections(await response.json()),
      failed: false,
    };
  } catch {
    return { elections: [], failed: true };
  }
}

/** Client-side retry, using the same base the rest of the browser app uses. */
export async function retryUpcoming(): Promise<PublicUpcomingElection[]> {
  const response = await fetch(`${API}/api/v1/public/elections/upcoming`, {
    headers: { Accept: "application/json" },
    cache: "no-store",
  });
  if (!response.ok) throw new Error(`Request failed (${response.status})`);
  return mapUpcomingElections(await response.json());
}

/** A banner link may only be site-relative or https; anything else is dropped. */
export function safeHref(link: string | null | undefined): string {
  const value = (link ?? "").trim();
  if (!value) return "";
  if (value.startsWith("/") && !value.startsWith("//")) return value;
  try {
    const url = new URL(value);
    return url.protocol === "https:" ? value : "";
  } catch {
    return "";
  }
}

/** Render safe support email as a mailto, or nothing. */
export function mailto(email: string | null | undefined): string {
  const value = (email ?? "").trim();
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value) ? `mailto:${value}` : "";
}

/** The banner shows only when enabled and inside its optional window. */
export function bannerVisible(banner: LandingDocument["banner"], now = Date.now()): boolean {
  if (!banner.enabled) return false;
  const from = banner.starts_at ? new Date(banner.starts_at).getTime() : 0;
  const to = banner.ends_at ? new Date(banner.ends_at).getTime() : Number.POSITIVE_INFINITY;
  return now >= from && now <= to;
}

/** The draft preview must publish the same visible items the publish call does. */
export function visibleDocument(draft: {
  hero: LandingDocument["hero"];
  banner: LandingDocument["banner"];
  ticker: string[];
  how: LandingDocument["how"];
  features: (LandingDocument["features"][number] & { visible?: boolean })[];
  faqs: (LandingDocument["faqs"][number] & { visible?: boolean })[];
  footer: LandingDocument["footer"];
}): LandingDocument {
  return {
    hero: draft.hero,
    banner: draft.banner,
    ticker: draft.ticker,
    how: draft.how,
    features: draft.features.filter((f) => f.visible !== false),
    faqs: draft.faqs.filter((f) => f.visible !== false),
    footer: draft.footer,
  };
}
