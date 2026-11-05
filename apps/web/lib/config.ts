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
