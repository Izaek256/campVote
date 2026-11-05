"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth, isAdmin } from "./auth-provider";
import { Icon } from "./icon";
import { ThemeToggle } from "./theme-toggle";

function initials(first: string, last: string) {
  return `${first.charAt(0)}${last.charAt(0)}`.toUpperCase();
}

export function SiteHeader() {
  const { user, signOut } = useAuth();
  const pathname = usePathname();
  const router = useRouter();

  async function handleSignOut() {
    await signOut();
    router.push("/login");
  }

  return (
    <div className="app-topnav">
      <div className="pubnav">
        <Link className="brand" href="/">
          <span className="mark">
            <Icon name="ballot" size="l" />
          </span>
          <span>
            camp<b>Vote</b>
          </span>
        </Link>
        <nav className="lk" aria-label="Main">
          <Link
            className={pathname === "/" || pathname.startsWith("/elections") ? "on" : ""}
            href="/"
          >
            Elections
          </Link>
          {isAdmin(user) ? <Link href="/staff">Admin panel</Link> : null}
          {user ? (
            <Link className={pathname === "/candidate" ? "on" : ""} href="/candidate">
              My candidacy
            </Link>
          ) : null}
          <Link className={pathname === "/how-voting-works" ? "on" : ""} href="/how-voting-works">
            How voting works
          </Link>
        </nav>
        <span className="row" style={{ marginLeft: "auto" }}>
          <ThemeToggle />
          {user ? (
            <>
              <Link className="av sm" href="/account" aria-label="Your account">
                {initials(user.first_name, user.last_name)}
              </Link>
              <button type="button" className="btn ghost sm" onClick={handleSignOut}>
                <Icon name="out" size="s" />
                Sign out
              </button>
            </>
          ) : (
            <Link className="btn primary sm" href="/login">
              Sign in
            </Link>
          )}
        </span>
      </div>
    </div>
  );
}
