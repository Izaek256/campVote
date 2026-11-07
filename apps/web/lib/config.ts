/**
 * Campus identity shown in the interface.
 *
 * The deployment can rename the campus without editing every page. Defaults
 * match the synthetic demo data in `apps/api/internal/database`; they are not
 * real institution credentials.
 */
export const CAMPUS_NAME = process.env.NEXT_PUBLIC_CAMPUS_NAME?.trim() || "Campus";
export const ELECTION_AUTHORITY =
  process.env.NEXT_PUBLIC_ELECTION_AUTHORITY?.trim() || "Campus Electoral Commission";
export const STUDENT_EMAIL_DOMAIN =
  process.env.NEXT_PUBLIC_STUDENT_EMAIL_DOMAIN?.trim() || "std.must.ac.ug";
export const STAFF_EMAIL_DOMAIN =
  process.env.NEXT_PUBLIC_STAFF_EMAIL_DOMAIN?.trim() || "must.ac.ug";
export const STUDENT_EMAIL_EXAMPLE =
  process.env.NEXT_PUBLIC_STUDENT_EMAIL_EXAMPLE?.trim() || `2024BSE088@${STUDENT_EMAIL_DOMAIN}`;

/**
 * The OAuth 2.0 Web Application client used by Google Identity Services.
 *
 * It is public by design: the browser must know it to request a credential.
 * The matching backend value is GOOGLE_OAUTH_CLIENT_ID, and the two must name
 * the same client. Empty means the button explains that sign-in is unavailable
 * instead of silently falling back to another method.
 */
export const GOOGLE_CLIENT_ID = process.env.NEXT_PUBLIC_GOOGLE_CLIENT_ID?.trim() || "";
