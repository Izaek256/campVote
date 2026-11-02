"use client";

import Link from "next/link";
import { Icon } from "./icon";

export function StaffShell({ children, title }: { children: React.ReactNode; title?: string }) {
  return (
    <div className="admin-shell">
      <aside className="admin-side">
        <Link className="b" href="/staff">
          <Icon name="moon" />
          campVote
        </Link>
        <span className="h">Staff</span>
        <Link className="on" href="/staff">
          <Icon name="chart" />
          Dashboard
        </Link>
        <Link href="/staff">
          <Icon name="ballot" />
          Elections
        </Link>
        <span className="h">Voter site</span>
        <Link href="/elections">
          <Icon name="ballot" />
          All elections
        </Link>
        <Link href="/candidate">
          <Icon name="user" />
          My candidacy
        </Link>
        <Link href="/account">
          <Icon name="user" />
          Account
        </Link>
        <div className="foot">
          <Link href="/how-voting-works">
            <Icon name="info" />
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
