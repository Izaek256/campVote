"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
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
  const [menuOpen, setMenuOpen] = useState(false);
  const menuToggleRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!menuOpen) return;

    function closeOnEscape(event: KeyboardEvent) {
      if (event.key !== "Escape") return;
      setMenuOpen(false);
      menuToggleRef.current?.focus();
    }

    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [menuOpen]);

  async function handleSignOut() {
    setMenuOpen(false);
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
        <nav id="site-header-menu" className={`lk${menuOpen ? " open" : ""}`} aria-label="Main">
          <Link
            className={pathname === "/" || pathname.startsWith("/elections") ? "on" : ""}
            href="/"
            onClick={() => setMenuOpen(false)}
          >
            Elections
          </Link>
          {isAdmin(user) ? (
            <Link href="/staff" onClick={() => setMenuOpen(false)}>
              Admin panel
            </Link>
          ) : null}
          {user ? (
            <Link
              className={pathname === "/candidate" ? "on" : ""}
              href="/candidate"
              onClick={() => setMenuOpen(false)}
            >
              My candidacy
            </Link>
          ) : null}
          <Link
            className={pathname === "/how-voting-works" ? "on" : ""}
            href="/how-voting-works"
            onClick={() => setMenuOpen(false)}
          >
            How voting works
          </Link>
          <span className="site-nav-session">
            {user ? (
              <>
                <Link href="/account" onClick={() => setMenuOpen(false)}>
                  Your account
                </Link>
                <button type="button" onClick={handleSignOut}>
                  <Icon name="out" size="s" />
                  Sign out
                </button>
              </>
            ) : (
              <Link href="/login" onClick={() => setMenuOpen(false)}>
                Sign in
              </Link>
            )}
          </span>
        </nav>
        <button
          ref={menuToggleRef}
          type="button"
          className="site-nav-toggle"
          aria-label={menuOpen ? "Close navigation menu" : "Open navigation menu"}
          aria-controls="site-header-menu"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((open) => !open)}
        >
          <Icon name={menuOpen ? "x" : "menu"} />
        </button>
        <span className="row" style={{ marginLeft: "auto" }}>
          <ThemeToggle />
          <span className="site-header-desktop-session">
            {user ? (
              <>
                <Link className="av sm" href="/account" aria-label="Your account">
                  {initials(user.first_name, user.last_name)}
                </Link>
                <button
                  type="button"
                  className="btn ghost sm"
                  onClick={handleSignOut}
                  aria-label="Sign out"
                >
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
        </span>
      </div>
    </div>
  );
}
