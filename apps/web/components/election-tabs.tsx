"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const tabs = [
  { key: "", label: "Overview" },
  { key: "positions", label: "Positions" },
  { key: "candidates", label: "Candidates" },
  { key: "ballot", label: "Ballot" },
  { key: "turnout", label: "Turnout" },
];

export function ElectionTabs({ id }: { id: string }) {
  const pathname = usePathname();
  const base = `/staff/elections/${id}`;
  return (
    <nav className="app-tabs" aria-label="Election management">
      {tabs.map((tab) => {
        const href = tab.key ? `${base}/${tab.key}` : base;
        const on = tab.key === "" ? pathname === base : pathname.startsWith(href);
        return (
          <Link key={tab.key} href={href} className={on ? "on" : ""}>
            {tab.label}
          </Link>
        );
      })}
    </nav>
  );
}
