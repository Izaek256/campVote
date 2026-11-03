"use client";

import { useEffect, useMemo } from "react";

/**
 * Shown once a ballot has been accepted. It replaces the ballot entirely, plays
 * a confetti burst with a drawn tick, and then hands control back to the caller
 * (which sends the voter home). The pieces are seeded from the current time so
 * the burst is not identical on every vote, but no ballot data is used.
 */
interface VoteSuccessProps {
  title?: string;
  delayMs?: number;
  onDone: () => void;
}

const COLORS = ["#2563eb", "#16a34a", "#f59e0b", "#dc2626", "#7c3aed", "#0891b2"];

function Confetti() {
  const pieces = useMemo(
    () =>
      Array.from({ length: 72 }, (_, index) => ({
        key: index,
        left: Math.random() * 100,
        delay: Math.random() * 0.6,
        duration: 2.2 + Math.random() * 1.8,
        color: COLORS[index % COLORS.length],
        rotate: Math.random() * 360,
        round: index % 4 === 0,
      })),
    []
  );

  return (
    <div className="vote-confetti" aria-hidden="true">
      {pieces.map((piece) => (
        <i
          key={piece.key}
          style={{
            left: `${piece.left}%`,
            background: piece.color,
            borderRadius: piece.round ? "50%" : "1px",
            animationDelay: `${piece.delay}s`,
            animationDuration: `${piece.duration}s`,
            transform: `rotate(${piece.rotate}deg)`,
          }}
        />
      ))}
    </div>
  );
}

export function VoteSuccess({ title, delayMs = 3200, onDone }: VoteSuccessProps) {
  useEffect(() => {
    const timer = window.setTimeout(onDone, delayMs);
    return () => window.clearTimeout(timer);
  }, [delayMs, onDone]);

  return (
    <div className="vote-success" role="dialog" aria-modal="true" aria-label="Vote recorded">
      <Confetti />
      <div className="vote-success-card">
        <div className="stamp go">
          <div className="badge">
            <svg className="tick" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M20 6 9 17l-5-5" />
            </svg>
          </div>
        </div>
        <h2 style={{ marginTop: 18 }}>Vote cast</h2>
        <p className="hint" style={{ marginTop: 8 }}>
          {title ? `${title} — ` : ""}your ballot has been recorded and cannot be changed. Your
          choices stay secret.
        </p>
        <p className="hint" style={{ marginTop: 12 }}>
          Taking you home…
        </p>
        <button type="button" className="btn primary" style={{ marginTop: 16 }} onClick={onDone}>
          Go home now
        </button>
      </div>
    </div>
  );
}
