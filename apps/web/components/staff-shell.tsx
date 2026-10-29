"use client";

import Link from "next/link";

export function StaffShell({ children, title }: { children: React.ReactNode; title?: string }) {
  return (
    <div className="admin-shell">
      <aside className="admin-side">
        <Link className="b" href="/staff">
          <svg className="i" aria-hidden="true">
            <path d="M2 27 11 10l6 10 3-5 10 12z" fill="currentColor" stroke="none" />
            <circle cx="24" cy="8" r="4" fill="currentColor" stroke="none" />
          </svg>
          campVote
        </Link>
        <span className="h">Elections</span>
        <Link className="on" href="/staff">
          <svg className="i" aria-hidden="true">
            <path d="M3 3v18h18M7 16v-5M12 16V6M17 16v-8" />
          </svg>
          Dashboard
        </Link>
        <Link href="/elections">
          <svg className="i" aria-hidden="true">
            <path d="M3 13h18v8H3zM7 13V3h10v10M10 8h4" />
          </svg>
          Voter view
        </Link>
        <Link href="/account">
          <svg className="i" aria-hidden="true">
            <circle cx="12" cy="8" r="4" />
            <path d="M4 21c0-4.4 3.6-8 8-8s8 3.6 8 8" />
          </svg>
          Account
        </Link>
        <div className="foot">
          <Link href="/how-voting-works">
            <svg className="i" aria-hidden="true">
              <circle cx="12" cy="12" r="10" />
              <path d="M12 16v-5M12 8h.01" />
            </svg>
            How voting works
          </Link>
        </div>
      </aside>
      <div className="admin-main">
        <div className="admin-top">
          <b>{title ?? "Election dashboard"}</b>
          <span className="chip draft">Officials</span>
        </div>
        <div className="admin-body">{children}</div>
      </div>
    </div>
  );
}
