"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { isAdmin, useAuth } from "./auth-provider";
import { Icon, type IconName } from "./icon";

interface NavItem {
  href: string;
  label: string;
  icon: IconName;
  /** Exact match only — used for a section root that also has children. */
  exact?: boolean;
}

interface NavSection {
  heading: string;
  items: NavItem[];
}

// Every election is built in this order. On desktop these steps live in the
// sidebar so they are reachable from anywhere inside the election; on mobile
// they are the same list inside the navigation drawer.
const ELECTION_STEPS: { key: string; label: string; icon: IconName; hash?: string }[] = [
  { key: "", label: "Details", icon: "file" },
  { key: "", label: "Eligibility", icon: "shield", hash: "#eligibility" },
  { key: "positions", label: "Positions", icon: "ballot" },
  { key: "candidates", label: "Candidates", icon: "users" },
  { key: "ballot", label: "Ballot", icon: "check" },
  { key: "", label: "Schedule", icon: "cal", hash: "#schedule" },
  { key: "monitor", label: "Monitor", icon: "eye" },
  { key: "results", label: "Results", icon: "chart" },
  { key: "turnout", label: "Turnout", icon: "users" },
];

const ADMIN_SECTIONS: NavSection[] = [
  {
    heading: "Elections",
    items: [
      { href: "/staff", label: "Dashboard", icon: "chart", exact: true },
      { href: "/elections", label: "All elections", icon: "ballot" },
      { href: "/staff/positions", label: "Positions", icon: "file" },
    ],
  },
  {
    heading: "My activity",
    items: [
      { href: "/candidate", label: "My candidacy", icon: "user" },
      { href: "/account", label: "Account", icon: "shield" },
    ],
  },
];

const STUDENT_SECTIONS: NavSection[] = [
  {
    heading: "Voting",
    items: [
      { href: "/elections", label: "Elections", icon: "ballot" },
      { href: "/results", label: "Results", icon: "chart" },
      { href: "/positions", label: "Run for a position", icon: "user" },
      { href: "/candidate", label: "My candidacy", icon: "file" },
      { href: "/account", label: "Account", icon: "shield" },
    ],
  },
];

function useHash(): string {
  const [hash, setHash] = useState("");
  useEffect(() => {
    const sync = () => setHash(window.location.hash);
    sync();
    window.addEventListener("hashchange", sync);
    return () => window.removeEventListener("hashchange", sync);
  }, []);
  return hash;
}

export function AppShell({
  children,
  title,
  electionId,
}: {
  children: React.ReactNode;
  title?: string;
  electionId?: string;
}) {
  const { user, signOut } = useAuth();
  const pathname = usePathname();
  const router = useRouter();
  const hash = useHash();
  const [open, setOpen] = useState(false);

  const admin = isAdmin(user);
  // Auto-detect the election from the URL so pages do not have to pass it.
  const activeElectionId = electionId ?? pathname.match(/^\/staff\/elections\/([^/]+)/)?.[1] ?? "";
  const base = activeElectionId ? `/staff/elections/${activeElectionId}` : "";

  // Keep the drawer from scrolling the page behind it on small screens.
  useEffect(() => {
    if (!open) return;
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previous;
    };
  }, [open]);

  const close = () => setOpen(false);

  async function handleSignOut() {
    await signOut();
    router.push("/login");
  }

  const sections = useMemo(() => (admin ? ADMIN_SECTIONS : STUDENT_SECTIONS), [admin]);

  function itemActive(item: NavItem): boolean {
    if (item.exact) return pathname === item.href;
    return pathname === item.href || pathname.startsWith(`${item.href}/`);
  }

  return (
    <div className="admin-shell">
      {open ? (
        <button
          type="button"
          className="admin-scrim"
          aria-label="Close navigation"
          onClick={close}
        />
      ) : null}

      <aside className={`admin-side${open ? " open" : ""}`}>
        <div style={{ display: "flex", alignItems: "center" }}>
          <Link className="b" href={admin ? "/staff" : "/elections"} onClick={close}>
            <Icon name="moon" />
            campVote
          </Link>
          <button type="button" className="close" aria-label="Close navigation" onClick={close}>
            <Icon name="x" />
          </button>
        </div>

        {admin ? (
          <>
            <Link className="quick" href="/staff?new=1" onClick={close}>
              <Icon name="plus" size="s" />
              New election
            </Link>
            <Link className="quick" href="/staff/positions?new=1" onClick={close}>
              <Icon name="plus" size="s" />
              New position
            </Link>
          </>
        ) : null}

        {admin && activeElectionId ? (
          <>
            <span className="h">This election</span>
            {ELECTION_STEPS.map((step) => {
              const href = `${base}${step.key ? `/${step.key}` : ""}${step.hash ?? ""}`;
              const on = step.key
                ? pathname.startsWith(`${base}/${step.key}`)
                : step.hash
                  ? pathname === base && hash === step.hash
                  : pathname === base && !hash;
              return (
                <Link key={step.label} className={on ? "on" : ""} href={href} onClick={close}>
                  <Icon name={step.icon} size="s" />
                  {step.label}
                </Link>
              );
            })}
            <Link href="/staff" onClick={close}>
              <Icon name="arrow" size="s" />
              All elections
            </Link>
          </>
        ) : null}

        {sections.map((section) => (
          <div key={section.heading}>
            <span className="h">{section.heading}</span>
            {section.items.map((item) => (
              <Link
                key={item.href}
                className={itemActive(item) ? "on" : ""}
                href={item.href}
                onClick={close}
              >
                <Icon name={item.icon} size="s" />
                {item.label}
              </Link>
            ))}
          </div>
        ))}

        <div className="foot">
          <div className="who">
            <Icon name="user" size="s" />
            <span title={user?.email}>
              {user ? `${user.first_name} ${user.last_name}` : "Guest"}
            </span>
          </div>
          <button type="button" className="signout" onClick={handleSignOut}>
            <Icon name="out" size="s" />
            Sign out
          </button>
          <Link href="/how-voting-works" onClick={close}>
            <Icon name="info" size="s" />
            How voting works
          </Link>
        </div>
      </aside>

      <div className="admin-main">
        <div className="admin-top">
          <button
            type="button"
            className="admin-burger"
            aria-label="Open navigation"
            aria-expanded={open}
            onClick={() => setOpen(true)}
          >
            <Icon name="menu" />
          </button>
          <b>{title ?? (admin ? "Admin" : "campVote")}</b>
          <span className="chip draft">{admin ? "Admin" : "Student voter"}</span>
        </div>
        <main className="admin-body">{children}</main>
      </div>
    </div>
  );
}
