/**
 * Shared tokens and small text helpers for the client-rendered election
 * documents.
 *
 * These are the campVote UI kit colours and nothing else: ink for structure,
 * one green for emphasis, sunken/line for tables. Every timestamp is printed in
 * EAT, and no document here can carry a voter's identity — a ballot has none.
 */
import { EAT } from "../format";

export const INK = "#0A0F0C";
export const INK_2 = "#38423D";
export const INK_3 = "#5B6761";
export const LINE = "#D3D9D6";
export const SUNKEN = "#F3F6F4";
export const PRIMARY = "#12783F";
export const WHITE = "#FFFFFF";
/** Light greys used only on the ink header band. */
export const ON_INK = "#C9D2CD";
export const ON_INK_DIM = "#A9B4AE";
export const ON_INK_LINE = "#3E4A44";

/** Page margins, in points, shared by both documents. */
export const MARGIN_X = 40;

function parts(iso?: string | null): Record<string, string> | null {
  if (!iso) return null;
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return null;
  const out: Record<string, string> = {};
  for (const part of new Intl.DateTimeFormat("en-GB", {
    timeZone: EAT,
    weekday: "short",
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).formatToParts(date)) {
    out[part.type] = part.value;
  }
  return out;
}

/** "29 Oct 2026 • 10:52 EAT" */
export function stamp(iso?: string | null): string {
  const p = parts(iso);
  if (!p) return "—";
  return `${p.day} ${p.month} ${p.year} • ${p.hour}:${p.minute} EAT`;
}

/** "Mon 26 Oct 2026 • 08:00 EAT" */
export function dayStamp(iso?: string | null): string {
  const p = parts(iso);
  if (!p) return "—";
  return `${p.weekday} ${p.day} ${p.month} ${p.year} • ${p.hour}:${p.minute} EAT`;
}

/** "26 Oct 2026" */
export function dateOnly(iso?: string | null): string {
  const p = parts(iso);
  if (!p) return "—";
  return `${p.day} ${p.month} ${p.year}`;
}

/** "17:03" — the clock time the polls closed. */
export function clockOnly(iso?: string | null): string {
  const p = parts(iso);
  if (!p) return "—";
  return `${p.hour}:${p.minute} EAT`;
}

/** "5,860" */
export function thousands(value: number): string {
  return new Intl.NumberFormat("en-GB").format(value);
}

/** "41.2%" */
export function percent(value: number, digits = 1): string {
  return `${(Number.isFinite(value) ? value : 0).toFixed(digits)}%`;
}

/** "2,321 valid votes" — used in position footnotes. */
export function votesWord(value: number): string {
  return `${thousands(value)} valid ${value === 1 ? "vote" : "votes"}`;
}

/** "1 SEAT" / "2 SEATS" */
export function seatLabel(seats: number): string {
  return `${seats} ${seats === 1 ? "SEAT" : "SEATS"}`;
}
