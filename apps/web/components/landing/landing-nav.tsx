"use client";

import type { LandingBanner } from "@campvote/shared/types/index";
import Link from "next/link";
import { useState } from "react";
import { ThemeToggle } from "@/components/theme-toggle";
import { bannerVisible, safeHref } from "@/lib/landing";
import { LogoMark } from "./landing-art";

const SECTIONS = [
  { href: "#elections", label: "Elections" },
  { href: "#how", label: "How it works" },
  { href: "#security", label: "Security" },
  { href: "#features", label: "Features" },
  { href: "#faq", label: "FAQ" },
];

/** The announcement strip, then the floating navigation docked beneath it. */
export function LandingNav({ banner }: { banner: LandingBanner }) {
  const [open, setOpen] = useState(false);
  const showBanner = bannerVisible(banner);
  const href = safeHref(banner.link);

  return (
    <>
      {showBanner ? (
        <div className="lp-bn" role="status">
          <span className="lp-dot" aria-hidden="true" />
          {href ? <a href={href}>{banner.text}</a> : <span>{banner.text}</span>}
        </div>
      ) : null}

      <nav className={`lp-nav${open ? " lp-m" : ""}`} aria-label="Main">
        <a className="lp-lg" href="#top">
          <span className="lp-mk">
            <LogoMark />
          </span>
          <span>
            camp<b>Vote</b>
          </span>
        </a>

        <div className="lp-lk">
          {SECTIONS.map((section) => (
            <a key={section.href} href={section.href}>
              {section.label}
            </a>
          ))}
          <i className="lp-ind" aria-hidden="true" />
        </div>

        <Link className="lp-bt" href="/login">
          Sign in
        </Link>
        <span className="lp-th">
          <ThemeToggle />
        </span>
        <button
          type="button"
          className="lp-mb"
          aria-label="Menu"
          aria-expanded={open}
          onClick={() => setOpen((value) => !value)}
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            aria-hidden="true"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path d="M3 6h18M3 12h18M3 18h18" />
          </svg>
        </button>
      </nav>
    </>
  );
}
