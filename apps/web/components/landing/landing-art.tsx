import type { LandingIconKey } from "@campvote/shared/types/index";

/**
 * The landing page's illustrations, kept in code rather than the CMS.
 *
 * Security illustrations make factual claims, so an administrator can never
 * edit or replace them. They are plain SVG with the kit's tokens as fills and
 * strokes; animation lives in landing.css and stops under reduced motion.
 */

export function LogoMark() {
  // Matches the app icon (app/icon.svg): a ballot box with a checked ballot
  // above it, drawn as a line glyph so it inherits the inverted colour the
  // .lp-mk container provides.
  return (
    <svg viewBox="0 0 32 32" fill="none" aria-hidden="true">
      <g
        transform="translate(4 4)"
        stroke="var(--bg)"
        strokeWidth={2}
        strokeLinecap="square"
        strokeLinejoin="miter"
      >
        <path d="M3 13h18v8H3zM7 13V3h10v10M10 8h4" />
      </g>
    </svg>
  );
}

/** A ballot is marked, then dropped into a sealed box. */
export function HeroArt() {
  return (
    <svg
      viewBox="0 0 360 300"
      role="img"
      aria-label="A ballot is marked, then dropped into a secure ballot box"
    >
      <defs>
        <clipPath id="lp-cp">
          <rect width="360" height="171" />
        </clipPath>
      </defs>
      <g clipPath="url(#lp-cp)">
        <g className="lp-bl">
          <rect x="100" y="24" width="160" height="130" className="lp-f-b lp-s-i" strokeWidth={2} />
          <rect x="112" y="36" width="56" height="7" className="lp-f-i" />
          <rect x="112" y="60" width="14" height="14" className="lp-s-i" strokeWidth={1.5} />
          <rect x="134" y="64" width="90" height="6" className="lp-f-l" />
          <rect x="112" y="84" width="14" height="14" className="lp-s-p" strokeWidth={1.5} />
          <path className="lp-ck lp-s-p" pathLength={1} d="M115 91l4 4 7-8" strokeWidth={2.5} />
          <rect x="134" y="88" width="70" height="6" className="lp-f-i" />
          <rect x="112" y="108" width="14" height="14" className="lp-s-i" strokeWidth={1.5} />
          <rect x="134" y="112" width="82" height="6" className="lp-f-l" />
        </g>
      </g>
      <rect x="50" y="171" width="260" height="114" className="lp-f-s lp-s-i" strokeWidth={2} />
      <rect x="100" y="167" width="160" height="9" className="lp-f-i" />
      <text x="180" y="224" textAnchor="middle" className="lp-tx">
        BALLOT BOX · SEALED
      </text>
      <g className="lp-bd">
        <circle cx="180" cy="256" r="13" className="lp-f-p" />
        <path d="M174 256l4 4 8-9" className="lp-s-b" strokeWidth={2.5} />
      </g>
    </svg>
  );
}

/** One illustration per "how it works" step; extra steps reuse them in order. */
export function StepArt({ index }: { index: number }) {
  const art = index % 4;
  if (art === 0) {
    return (
      <svg viewBox="0 0 160 74" aria-hidden="true">
        <circle cx="40" cy="37" r="22" className="lp-f-l" />
        <circle cx="40" cy="30" r="8" className="lp-f-b" />
        <path d="M25 56a15 15 0 0 1 30 0z" className="lp-f-b" />
        <g className="lp-pop">
          <rect x="76" y="24" width="72" height="26" className="lp-f-p" />
          <text
            x="112"
            y="41"
            textAnchor="middle"
            className="lp-tx"
            style={{ fill: "var(--on-primary)" }}
          >
            SIGN IN
          </text>
        </g>
      </svg>
    );
  }
  if (art === 1) {
    return (
      <svg viewBox="0 0 160 74" aria-hidden="true">
        <rect x="10" y="10" width="110" height="12" className="lp-f-l" />
        <path
          className="lp-dw lp-s-p"
          style={{ "--i": 0 } as React.CSSProperties}
          pathLength={1}
          d="M130 16l5 5 10-11"
          strokeWidth={2.5}
        />
        <rect x="10" y="31" width="90" height="12" className="lp-f-l" />
        <path
          className="lp-dw lp-s-p"
          style={{ "--i": 1 } as React.CSSProperties}
          pathLength={1}
          d="M130 37l5 5 10-11"
          strokeWidth={2.5}
        />
        <rect x="10" y="52" width="102" height="12" className="lp-f-l" />
        <path
          className="lp-dw lp-s-p"
          style={{ "--i": 2 } as React.CSSProperties}
          pathLength={1}
          d="M130 58l5 5 10-11"
          strokeWidth={2.5}
        />
      </svg>
    );
  }
  if (art === 2) {
    return (
      <svg viewBox="0 0 160 74" aria-hidden="true">
        <circle cx="22" cy="14" r="7" className="lp-s-i" strokeWidth={1.5} />
        <rect x="40" y="11" width="90" height="6" className="lp-f-l" />
        <circle cx="22" cy="37" r="7" className="lp-s-p" strokeWidth={1.5} />
        <circle cx="22" cy="37" r="3.5" className="lp-f-p lp-pop" />
        <rect x="40" y="34" width="70" height="6" className="lp-f-i" />
        <circle cx="22" cy="60" r="7" className="lp-s-i" strokeWidth={1.5} />
        <rect x="40" y="57" width="80" height="6" className="lp-f-l" />
      </svg>
    );
  }
  return (
    <svg viewBox="0 0 160 74" aria-hidden="true">
      <rect x="22" y="6" width="96" height="62" className="lp-f-b lp-s-i" strokeWidth={1.5} />
      <text x="70" y="30" textAnchor="middle" className="lp-tx">
        CV-7F3A-92C1
      </text>
      <rect x="38" y="40" width="64" height="5" className="lp-f-l" />
      <g className="lp-pop">
        <circle cx="124" cy="54" r="14" className="lp-f-p" />
        <path d="M117 54l5 5 9-11" className="lp-s-b" strokeWidth={3} />
      </g>
    </svg>
  );
}

/** The three security cards: identity/choice split, one ballot, audit chain. */
export function SecurityArt({ index }: { index: number }) {
  if (index === 0) {
    return (
      <svg
        viewBox="0 0 320 170"
        role="img"
        aria-label="A ticked name list is not linked to anonymous ballots in the box"
      >
        <text x="12" y="18" className="lp-tx">
          PARTICIPATION RECORD
        </text>
        <text x="196" y="18" className="lp-tx">
          BALLOT BOX
        </text>
        <rect x="12" y="34" width="84" height="10" className="lp-f-l" />
        <rect x="12" y="62" width="70" height="10" className="lp-f-l" />
        <rect x="12" y="90" width="92" height="10" className="lp-f-l" />
        <path
          className="lp-tk1 lp-s-p"
          style={{ "--i": 0 } as React.CSSProperties}
          d="M116 34l4 5 9-11"
          strokeWidth={2.5}
        />
        <path
          className="lp-tk1 lp-s-p"
          style={{ "--i": 1 } as React.CSSProperties}
          d="M116 62l4 5 9-11"
          strokeWidth={2.5}
        />
        <path
          className="lp-tk1 lp-s-p"
          style={{ "--i": 2 } as React.CSSProperties}
          d="M116 90l4 5 9-11"
          strokeWidth={2.5}
        />
        <path d="M140 70h50" className="lp-s-l" strokeDasharray="4 4" strokeWidth={2} />
        <text
          x="165"
          y="60"
          textAnchor="middle"
          className="lp-tx"
          style={{ fill: "var(--danger)" }}
        >
          NO LINK
        </text>
        <rect x="196" y="30" width="112" height="124" className="lp-f-s lp-s-i" strokeWidth={1.5} />
        <rect
          className="lp-dp lp-f-p"
          style={{ "--i": 0 } as React.CSSProperties}
          x="206"
          y="118"
          width="26"
          height="26"
        />
        <rect
          className="lp-dp lp-f-p"
          style={{ "--i": 1 } as React.CSSProperties}
          x="238"
          y="118"
          width="26"
          height="26"
        />
        <rect
          className="lp-dp lp-f-p"
          style={{ "--i": 2 } as React.CSSProperties}
          x="270"
          y="118"
          width="26"
          height="26"
        />
        <text x="12" y="150" className="lp-tx">
          Name ticked. Choice anonymous.
        </text>
      </svg>
    );
  }
  if (index === 1) {
    return (
      <svg
        viewBox="0 0 320 170"
        role="img"
        aria-label="A first ballot is accepted and a second attempt is rejected"
      >
        <text x="20" y="34" className="lp-tx">
          1ST BALLOT
        </text>
        <text x="20" y="158" className="lp-tx">
          2ND ATTEMPT
        </text>
        <path d="M20 76h190M20 124h150" className="lp-s-l" strokeDasharray="4 4" />
        <rect x="220" y="44" width="88" height="96" className="lp-f-s lp-s-i" strokeWidth={1.5} />
        <text x="264" y="96" textAnchor="middle" className="lp-tx">
          1 BALLOT
        </text>
        <rect x="168" y="106" width="5" height="36" className="lp-f-i" />
        <rect className="lp-t1 lp-f-p" x="20" y="64" width="24" height="24" />
        <rect className="lp-t2 lp-f-r" x="20" y="112" width="24" height="24" />
        <text className="lp-l1 lp-tx" x="170" y="34" style={{ fill: "var(--primary)" }}>
          ACCEPTED
        </text>
        <text className="lp-l2 lp-tx" x="190" y="158" style={{ fill: "var(--danger)" }}>
          REJECTED
        </text>
      </svg>
    );
  }
  return (
    <svg
      viewBox="0 0 320 170"
      role="img"
      aria-label="Linked ballot hashes; editing one breaks every later hash"
    >
      <g className="lp-hb" style={{ "--i": 0 } as React.CSSProperties}>
        <rect x="10" y="50" width="62" height="70" className="lp-f-s lp-s-i" strokeWidth={1.5} />
        <text x="18" y="68" className="lp-tx">
          #9F3C
        </text>
        <rect x="18" y="80" width="46" height="5" className="lp-f-l" />
        <rect x="18" y="92" width="30" height="5" className="lp-f-l" />
      </g>
      <g className="lp-hb" style={{ "--i": 1 } as React.CSSProperties}>
        <rect x="88" y="50" width="62" height="70" className="lp-f-s lp-s-i" strokeWidth={1.5} />
        <text x="96" y="68" className="lp-tx">
          #B71E
        </text>
        <rect x="96" y="80" width="46" height="5" className="lp-f-l" />
        <rect x="96" y="92" width="30" height="5" className="lp-f-l" />
      </g>
      <g className="lp-hb" style={{ "--i": 2 } as React.CSSProperties}>
        <rect x="166" y="50" width="62" height="70" className="lp-f-s lp-s-i" strokeWidth={1.5} />
        <text x="174" y="68" className="lp-tx">
          #04AD
        </text>
        <rect x="174" y="80" width="46" height="5" className="lp-f-l" />
        <rect x="174" y="92" width="30" height="5" className="lp-f-l" />
      </g>
      <g className="lp-hb" style={{ "--i": 3 } as React.CSSProperties}>
        <rect x="244" y="50" width="62" height="70" className="lp-f-s lp-s-i" strokeWidth={1.5} />
        <text x="252" y="68" className="lp-tx">
          #E2D0
        </text>
        <rect x="252" y="80" width="46" height="5" className="lp-f-l" />
        <rect x="252" y="92" width="30" height="5" className="lp-f-l" />
      </g>
      <path d="M72 85h16M150 85h16M228 85h16" className="lp-s-i" strokeWidth={2} />
      <circle className="lp-pl lp-f-p" cx="10" cy="136" r="4" />
      <path d="M10 136h296" className="lp-s-l" strokeDasharray="3 4" />
      <text x="10" y="22" className="lp-tx">
        EDIT ONE, EVERY LATER HASH BREAKS
      </text>
    </svg>
  );
}

/** The eight feature illustrations, keyed by the CMS allowlist. */
export function FeatureArt({ icon }: { icon: LandingIconKey }) {
  switch (icon) {
    case "elections":
      return (
        <svg viewBox="0 0 120 72" aria-hidden="true">
          <rect
            className="lp-sl lp-f-s lp-s-l"
            style={{ "--i": 2 } as React.CSSProperties}
            x="30"
            y="4"
            width="70"
            height="44"
          />
          <rect
            className="lp-sl lp-f-s lp-s-l"
            style={{ "--i": 1 } as React.CSSProperties}
            x="20"
            y="12"
            width="70"
            height="44"
          />
          <rect x="10" y="20" width="70" height="46" className="lp-f-b lp-s-i" strokeWidth={1.5} />
          <rect x="18" y="28" width="30" height="5" className="lp-f-p" />
          <rect x="18" y="40" width="50" height="4" className="lp-f-l" />
          <rect x="18" y="50" width="38" height="4" className="lp-f-l" />
        </svg>
      );
    case "applications":
      return (
        <svg viewBox="0 0 120 72" aria-hidden="true">
          <rect x="14" y="6" width="74" height="60" className="lp-f-b lp-s-i" strokeWidth={1.5} />
          <circle cx="34" cy="26" r="9" className="lp-f-l" />
          <rect x="50" y="20" width="28" height="5" className="lp-f-i" />
          <rect x="50" y="30" width="20" height="4" className="lp-f-l" />
          <rect x="22" y="48" width="58" height="5" className="lp-f-l" />
          <g className="lp-pop">
            <circle cx="92" cy="50" r="14" className="lp-f-p" />
            <path d="M85 50l5 5 9-11" className="lp-s-b" strokeWidth={3} />
          </g>
        </svg>
      );
    case "builder":
      return (
        <svg viewBox="0 0 120 72" aria-hidden="true">
          <g className="lp-rw" style={{ "--i": 0 } as React.CSSProperties}>
            <rect
              x="10"
              y="4"
              width="100"
              height="14"
              className="lp-f-b lp-s-i"
              strokeWidth={1.5}
            />
            <rect x="16" y="9" width="40" height="4" className="lp-f-i" />
          </g>
          <g className="lp-rw" style={{ "--i": 1 } as React.CSSProperties}>
            <rect
              x="10"
              y="22"
              width="100"
              height="14"
              className="lp-f-b lp-s-i"
              strokeWidth={1.5}
            />
            <rect x="16" y="27" width="56" height="4" className="lp-f-i" />
          </g>
          <g className="lp-rw" style={{ "--i": 2 } as React.CSSProperties}>
            <rect
              x="10"
              y="40"
              width="100"
              height="14"
              className="lp-f-b lp-s-i"
              strokeWidth={1.5}
            />
            <rect x="16" y="45" width="30" height="4" className="lp-f-i" />
          </g>
          <rect x="10" y="59" width="100" height="10" className="lp-s-p" strokeDasharray="4 3" />
        </svg>
      );
    case "eligibility":
      return (
        <svg viewBox="0 0 120 72" aria-hidden="true">
          <circle
            className="lp-dt lp-f-l"
            style={{ "--i": 0 } as React.CSSProperties}
            cx="20"
            cy="22"
            r="7"
          />
          <circle
            className="lp-dt lp-f-l"
            style={{ "--i": 2 } as React.CSSProperties}
            cx="40"
            cy="22"
            r="7"
          />
          <circle
            className="lp-dt lp-f-l"
            style={{ "--i": 4 } as React.CSSProperties}
            cx="60"
            cy="22"
            r="7"
          />
          <circle
            className="lp-dt lp-f-l"
            style={{ "--i": 1 } as React.CSSProperties}
            cx="80"
            cy="22"
            r="7"
          />
          <circle
            className="lp-dt lp-f-l"
            style={{ "--i": 3 } as React.CSSProperties}
            cx="100"
            cy="22"
            r="7"
          />
          <circle
            className="lp-dt lp-f-l"
            style={{ "--i": 3 } as React.CSSProperties}
            cx="20"
            cy="50"
            r="7"
          />
          <circle
            className="lp-dt lp-f-l"
            style={{ "--i": 1 } as React.CSSProperties}
            cx="40"
            cy="50"
            r="7"
          />
          <circle
            className="lp-dt lp-f-l"
            style={{ "--i": 4 } as React.CSSProperties}
            cx="60"
            cy="50"
            r="7"
          />
          <circle
            className="lp-dt lp-f-l"
            style={{ "--i": 2 } as React.CSSProperties}
            cx="80"
            cy="50"
            r="7"
          />
          <circle
            className="lp-dt lp-f-l"
            style={{ "--i": 0 } as React.CSSProperties}
            cx="100"
            cy="50"
            r="7"
          />
        </svg>
      );
    case "roles":
      return (
        <svg viewBox="0 0 120 72" aria-hidden="true">
          <g className="lp-fy" style={{ "--i": 0 } as React.CSSProperties}>
            <circle cx="22" cy="24" r="8" className="lp-f-i" />
            <path d="M8 54a14 14 0 0 1 28 0z" className="lp-f-p" />
          </g>
          <g className="lp-fy" style={{ "--i": 1 } as React.CSSProperties}>
            <circle cx="50" cy="24" r="8" className="lp-f-i" />
            <path d="M36 54a14 14 0 0 1 28 0z" className="lp-f-l" />
          </g>
          <g className="lp-fy" style={{ "--i": 2 } as React.CSSProperties}>
            <circle cx="78" cy="24" r="8" className="lp-f-i" />
            <path d="M64 54a14 14 0 0 1 28 0z" className="lp-f-l" />
          </g>
          <g className="lp-fy" style={{ "--i": 3 } as React.CSSProperties}>
            <circle cx="102" cy="24" r="8" className="lp-f-i" />
            <path d="M88 54a14 14 0 0 1 28 0z" className="lp-f-l" />
          </g>
        </svg>
      );
    case "turnout":
      return (
        <svg viewBox="0 0 120 72" aria-hidden="true">
          <path d="M8 66h104" className="lp-s-i" strokeWidth={1.5} />
          <rect
            className="lp-gr lp-f-l"
            style={{ "--i": 0 } as React.CSSProperties}
            x="14"
            y="38"
            width="14"
            height="28"
          />
          <rect
            className="lp-gr lp-f-l"
            style={{ "--i": 1 } as React.CSSProperties}
            x="36"
            y="26"
            width="14"
            height="40"
          />
          <rect
            className="lp-gr lp-f-l"
            style={{ "--i": 2 } as React.CSSProperties}
            x="58"
            y="34"
            width="14"
            height="32"
          />
          <rect
            className="lp-gr lp-f-l"
            style={{ "--i": 3 } as React.CSSProperties}
            x="80"
            y="18"
            width="14"
            height="48"
          />
          <rect
            className="lp-gr lp-f-p"
            style={{ "--i": 4 } as React.CSSProperties}
            x="102"
            y="8"
            width="10"
            height="58"
          />
        </svg>
      );
    case "results":
      return (
        <svg viewBox="0 0 120 72" aria-hidden="true">
          <rect x="26" y="4" width="62" height="64" className="lp-f-b lp-s-i" strokeWidth={1.5} />
          <rect x="34" y="14" width="30" height="5" className="lp-f-i" />
          <rect x="34" y="26" width="46" height="4" className="lp-f-l" />
          <rect x="34" y="36" width="40" height="4" className="lp-f-l" />
          <g className="lp-pop">
            <circle cx="88" cy="52" r="14" className="lp-f-p" />
            <path d="M81 52l5 5 9-11" className="lp-s-b" strokeWidth={3} />
          </g>
        </svg>
      );
    case "notifications":
      return (
        <svg viewBox="0 0 120 72" aria-hidden="true">
          <rect x="22" y="20" width="64" height="42" className="lp-f-b lp-s-i" strokeWidth={1.5} />
          <path d="M22 20l32 24 32-24" className="lp-s-i" strokeWidth={1.5} />
          <circle className="lp-pg lp-f-r" cx="88" cy="20" r="9" />
        </svg>
      );
    default:
      return null;
  }
}
