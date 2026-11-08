/**
 * The public landing page ("/") contract.
 *
 * These types mirror the Go API exactly:
 *   - `LandingDocument` matches `internal/landing.Document` (the shape served
 *     by `GET /api/v1/public/landing` and edited through `/admin/landing`).
 *   - `PublicUpcomingElection` matches the whitelisted fields returned by
 *     `GET /api/v1/public/elections/upcoming`.
 *
 * Keep them in step with the server: the landing endpoint serves only these
 * fields, and it deliberately never carries rosters, voter lists, candidate
 * documents, emails, ballots or tallies.
 */

/** The eight feature illustrations the CMS may choose from. */
export type LandingIconKey =
  | "elections"
  | "applications"
  | "builder"
  | "eligibility"
  | "roles"
  | "turnout"
  | "results"
  | "notifications";

export interface LandingHero {
  headline: string;
  subtext: string;
  primary_label: string;
  secondary_label: string;
}

/** The announcement strip. When a bound is set the banner only shows inside it. */
export interface LandingBanner {
  enabled: boolean;
  text: string;
  link: string;
  starts_at?: string | null;
  ends_at?: string | null;
}

export interface LandingStep {
  title: string;
  text: string;
}

export interface LandingHow {
  steps: LandingStep[];
}

export interface LandingFeature {
  icon: LandingIconKey;
  title: string;
  text: string;
}

export interface LandingFAQ {
  question: string;
  answer: string;
}

export interface LandingFooter {
  text: string;
  support_email: string;
}

/** The whole published landing page. */
export interface LandingDocument {
  hero: LandingHero;
  banner: LandingBanner;
  ticker: string[];
  how: LandingHow;
  features: LandingFeature[];
  faqs: LandingFAQ[];
  footer: LandingFooter;
}

/** "registration" while applications are open, otherwise "scheduled". */
export type PublicElectionStatus = "registration" | "scheduled";

/**
 * One upcoming election as the landing page sees it: whitelisted aggregate
 * facts only, never anything that identifies a voter or their choice.
 */
export interface PublicUpcomingElection {
  id: string;
  title: string;
  category: string;
  organizing_unit: string;
  positions: number;
  /** ISO 8601 with time zone. */
  opens_at: string;
  /** ISO 8601 with time zone. */
  closes_at: string;
  status: PublicElectionStatus;
  /**
   * Present only when the election's policy lets the server count eligible
   * voters. An aggregate, never a list of people.
   */
  eligible_count?: number;
}

/* ── Admin CMS ─────────────────────────────────────────────────────────────
 * The editor works on a draft. Item ids exist so the form can key its inputs;
 * `visible` hides an item from the published page without deleting it.
 */

export interface LandingFeatureDraft extends LandingFeature {
  id: string;
  visible: boolean;
}

export interface LandingFAQDraft extends LandingFAQ {
  id: string;
  visible: boolean;
}

export interface LandingDraft {
  hero: LandingHero;
  banner: LandingBanner;
  ticker: string[];
  how: LandingHow;
  features: LandingFeatureDraft[];
  faqs: LandingFAQDraft[];
  footer: LandingFooter;
}

/** The body of `GET /api/v1/admin/landing` and the save/publish/revert replies. */
export interface LandingAdminState {
  draft: LandingDraft;
  published: LandingDocument;
  has_published: boolean;
  /** True when the draft differs from what is currently published. */
  dirty: boolean;
  updated_at?: string | null;
  published_at?: string | null;
}
