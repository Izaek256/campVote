import type { ElectionStatus } from "./types";

/** MMU display zone. Stored timestamps are UTC; voters read EAT. */
export const EAT = "Africa/Kampala";

const dateTime = new Intl.DateTimeFormat("en-GB", {
  timeZone: EAT,
  weekday: "short",
  day: "2-digit",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

/** "Wed 28 Oct 2026, 17:00 EAT" */
export function formatEAT(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "—";
  return `${dateTime.format(date).replace(",", "")} EAT`;
}

/** "Wed 28 Oct, 17:00 EAT" for cards and tables. */
export function formatShortEAT(iso: string): string {
  return formatEAT(iso).replace(/\s\d{4},/, ",");
}

export function formatNumber(value: number): string {
  return new Intl.NumberFormat("en-GB").format(value);
}

export function formatPercent(value: number, digits = 1): string {
  return `${value.toFixed(digits)}%`;
}

export interface Remaining {
  days: number;
  hours: number;
  minutes: number;
  seconds: number;
  expired: boolean;
}

export function remaining(endsAt: string, now = Date.now()): Remaining {
  const diff = new Date(endsAt).getTime() - now;
  if (!Number.isFinite(diff) || diff <= 0) {
    return { days: 0, hours: 0, minutes: 0, seconds: 0, expired: true };
  }
  const total = Math.floor(diff / 1000);
  return {
    days: Math.floor(total / 86400),
    hours: Math.floor((total % 86400) / 3600),
    minutes: Math.floor((total % 3600) / 60),
    seconds: total % 60,
    expired: false,
  };
}

const labels: Record<ElectionStatus, string> = {
  DRAFT: "Draft",
  REGISTRATION: "Registration",
  PENDING_APPROVAL: "Pending approval",
  SCHEDULED: "Scheduled",
  OPEN: "Open",
  CLOSED: "Closed",
  RESULTS_UNDER_REVIEW: "Results under review",
  FINALIZED: "Finalized",
  ARCHIVED: "Archived",
  CANCELLED: "Cancelled",
};

const chipClasses: Record<ElectionStatus, string> = {
  DRAFT: "draft",
  REGISTRATION: "review",
  PENDING_APPROVAL: "review",
  SCHEDULED: "sched",
  OPEN: "open",
  CLOSED: "closed",
  RESULTS_UNDER_REVIEW: "review",
  FINALIZED: "final",
  ARCHIVED: "final",
  CANCELLED: "cancel",
};

export function statusLabel(status: ElectionStatus): string {
  return labels[status] ?? status;
}

export function statusClass(status: ElectionStatus): string {
  return chipClasses[status] ?? "draft";
}

export function categoryLabel(category: string): string {
  return category.charAt(0) + category.slice(1).toLowerCase();
}

export function timezoneClock(serverTime: string): string {
  const date = new Date(serverTime);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("en-GB", {
    timeZone: EAT,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).format(date);
}
