"use client";

import type { PublicUpcomingElection } from "@campvote/shared/types/index";
import { useEffect, useState } from "react";
import { formatNumber, formatShortEAT } from "@/lib/format";
import { retryUpcoming } from "@/lib/landing";

function countdown(opensAt: string, now: number): string {
  const diff = new Date(opensAt).getTime() - now;
  if (!Number.isFinite(diff)) return "Date to be announced";
  if (diff <= 0) return "Opening now";
  const total = Math.floor(diff / 1000);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${Math.floor(total / 86400)}d ${pad(Math.floor((total % 86400) / 3600))}h ${pad(
    Math.floor((total % 3600) / 60)
  )}m ${pad(total % 60)}s`;
}

/**
 * Upcoming elections are server-fetched for the first paint and retried from
 * the browser. This is the one place on the landing page that shows an error;
 * every other section falls back to safe defaults.
 */
export function LandingUpcoming({
  initial,
  initialFailed,
}: {
  initial: PublicUpcomingElection[];
  initialFailed: boolean;
}) {
  const [elections, setElections] = useState(initial);
  const [failed, setFailed] = useState(initialFailed);
  const [loading, setLoading] = useState(false);
  const [now, setNow] = useState<number | null>(null);

  useEffect(() => {
    setNow(Date.now());
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  async function retry() {
    setLoading(true);
    setFailed(false);
    try {
      setElections(await retryUpcoming());
    } catch {
      setFailed(true);
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="lp-sec lp-sx" id="elections" style={{ paddingBottom: 24 }}>
      <div className="lp-w">
        <div className="lp-eh">
          <div>
            <div className="lp-mo lp-rv">Upcoming elections</div>
            <h2 className="lp-h2 lp-rv">What’s on the ballot next.</h2>
          </div>
          <span className="lp-mo">{failed ? "" : "Live from the election service"}</span>
        </div>
        <p className="lp-sub lp-rv">
          Elections you can apply to or vote in soon. Check the dates and who is eligible, then sign
          in when voting opens.
        </p>

        <div className="lp-el" aria-live="polite" aria-busy={loading}>
          {loading ? (
            <>
              <div className="lp-sk" />
              <div className="lp-sk" />
              <div className="lp-sk" />
            </>
          ) : failed ? (
            <div className="lp-msg">
              <span>We couldn’t load upcoming elections.</span>
              <button type="button" className="lp-bt lp-o" onClick={retry}>
                Try again
              </button>
            </div>
          ) : elections.length === 0 ? (
            <div className="lp-msg">
              No upcoming elections right now. Check back soon.
              <button type="button" className="lp-bt lp-o" onClick={retry} disabled={loading}>
                Refresh
              </button>
            </div>
          ) : (
            elections.map((election, index) => {
              const registration = election.status === "registration";
              return (
                <article
                  className="lp-ec lp-rv"
                  key={election.id}
                  style={{ "--d": `${index * 0.1}s` } as React.CSSProperties}
                >
                  <span className={`lp-chp${registration ? "" : " lp-s"}`}>
                    {registration ? "Applications open" : "Scheduled"}
                  </span>
                  <h3>{election.title}</h3>
                  <div className="lp-mo">
                    {election.category || "Election"}
                    {election.positions > 0 ? ` · ${election.positions} positions` : ""}
                  </div>
                  <div>
                    <div className="lp-mo">Voting opens in</div>
                    <div className="lp-cnt">
                      {now === null ? "--" : countdown(election.opens_at, now)}
                    </div>
                  </div>
                  <dl>
                    <dt>Opens</dt>
                    <dd>{formatShortEAT(election.opens_at)}</dd>
                    <dt>Closes</dt>
                    <dd>{formatShortEAT(election.closes_at)}</dd>
                    {election.organizing_unit ? (
                      <>
                        <dt>Who</dt>
                        <dd>
                          {election.organizing_unit}
                          {election.eligible_count !== undefined
                            ? ` · ${formatNumber(election.eligible_count)} eligible`
                            : ""}
                        </dd>
                      </>
                    ) : null}
                  </dl>
                  <a className="lp-bt lp-o" href={`/elections/${election.id}`}>
                    View details →
                  </a>
                </article>
              );
            })
          )}
        </div>
      </div>
    </section>
  );
}
