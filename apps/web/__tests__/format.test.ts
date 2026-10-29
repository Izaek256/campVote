import { describe, expect, it } from "vitest";
import {
  formatEAT,
  formatNumber,
  formatPercent,
  remaining,
  statusLabel,
  statusClass,
} from "@/lib/format";

describe("EAT formatting", () => {
  it("renders a UTC timestamp in East Africa Time", () => {
    expect(formatEAT("2026-10-28T14:00:00Z")).toBe("Wed 28 Oct 2026, 17:00 EAT");
  });

  it("does not invent a date for a bad timestamp", () => {
    expect(formatEAT("not-a-date")).toBe("—");
  });

  it("keeps digits tabular-friendly and names percentages", () => {
    expect(formatNumber(5860)).toBe("5,860");
    expect(formatPercent(41.234)).toBe("41.2%");
  });
});

describe("countdown", () => {
  it("splits the remaining time into days, hours, minutes and seconds", () => {
    const now = Date.parse("2026-10-28T10:00:00Z");
    const left = remaining("2026-10-29T16:12:05Z", now);
    expect(left).toEqual({ days: 1, hours: 6, minutes: 12, seconds: 5, expired: false });
  });

  it("is expired at and after the end time", () => {
    const now = Date.parse("2026-10-28T10:00:00Z");
    expect(remaining("2026-10-28T10:00:00Z", now).expired).toBe(true);
    expect(remaining("2026-10-28T09:00:00Z", now).expired).toBe(true);
  });
});

describe("lifecycle labels", () => {
  it("maps every state to words and never colour alone", () => {
    expect(statusLabel("RESULTS_UNDER_REVIEW")).toBe("Results under review");
    expect(statusLabel("CANCELLED")).toBe("Cancelled");
    expect(statusClass("OPEN")).toBe("open");
    expect(statusClass("FINALIZED")).toBe("final");
    expect(statusClass("PENDING_APPROVAL")).toBe("review");
  });
});
