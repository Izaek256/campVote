"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

// The setup rail mirrors the order in which an election is built. Overview and
// Schedule share the overview page; the hash simply scrolls to that panel.
const steps = [
  { key: "", label: "Details", hash: "" },
  { key: "", label: "Eligibility", hash: "#eligibility" },
  { key: "positions", label: "Positions" },
  { key: "candidates", label: "Candidates" },
  { key: "ballot", label: "Ballot" },
  { key: "", label: "Schedule", hash: "#schedule" },
  { key: "monitor", label: "Monitor" },
  { key: "results", label: "Results" },
  { key: "turnout", label: "Turnout" },
];

export function ElectionTabs({ id }: { id: string }) {
  const pathname = usePathname();
  const base = `/staff/elections/${id}`;
  return (
    <nav className="app-tabs" aria-label="Election setup steps">
      {steps.map((step) => {
        const href = `${base}${step.key ? `/${step.key}` : ""}${step.hash ?? ""}`;
        const on = step.key
          ? pathname.startsWith(`${base}/${step.key}`)
          : pathname === base && step.label === "Details";
        return (
          <Link key={step.label} href={href} className={on ? "on" : ""}>
            {step.label}
          </Link>
        );
      })}
    </nav>
  );
}
