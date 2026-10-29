"use client";

import { useEffect, useState } from "react";
import { remaining } from "@/lib/format";

/** Counts down to the server-supplied end time; never trusts local election data. */
export function Countdown({ endsAt, label = "Time left" }: { endsAt: string; label?: string }) {
  const [left, setLeft] = useState(() => remaining(endsAt));

  useEffect(() => {
    setLeft(remaining(endsAt));
    const timer = setInterval(() => setLeft(remaining(endsAt)), 1000);
    return () => clearInterval(timer);
  }, [endsAt]);

  if (left.expired) {
    return <span className="hint">Voting has closed.</span>;
  }

  return (
    <div
      className="cd"
      role="timer"
      aria-label={`${label}: ${left.days} days ${left.hours} hours ${left.minutes} minutes`}
    >
      <div>
        <b>{String(left.days).padStart(2, "0")}</b>
        <small>days</small>
      </div>
      <div>
        <b>{String(left.hours).padStart(2, "0")}</b>
        <small>hrs</small>
      </div>
      <div>
        <b>{String(left.minutes).padStart(2, "0")}</b>
        <small>min</small>
      </div>
      <div>
        <b>{String(left.seconds).padStart(2, "0")}</b>
        <small>sec</small>
      </div>
    </div>
  );
}
