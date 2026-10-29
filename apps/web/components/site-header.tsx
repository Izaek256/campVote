"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth, isStaff } from "./auth-provider";
import { Icon } from "./icon";

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
            <Icon name="moon" size="l" />
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
          {isStaff(user) ? <Link href="/staff">Staff console</Link> : null}
          <Link className={pathname === "/how-voting-works" ? "on" : ""} href="/how-voting-works">
            How voting works
          </Link>
        </nav>
        <span className="row" style={{ marginLeft: "auto" }}>
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
              Sign in with MMU account
            </Link>
          )}
        </span>
      </div>
    </div>
  );
}
