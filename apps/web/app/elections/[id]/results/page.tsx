"use client";

import Link from "next/link";
import { use, useEffect, useMemo, useState } from "react";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { SiteHeader } from "@/components/site-header";
import { StatusChip } from "@/components/status-chip";
import { ApiError, elections } from "@/lib/api";
import { formatNumber, formatPercent } from "@/lib/format";
import type { Results, Tally } from "@/lib/types";

function ResultsView({ id }: { id: string }) {
  const [results, setResults] = useState<Results | null>(null);
  const [error, setError] = useState("");
  const [blocked, setBlocked] = useState(false);

  useEffect(() => {
    let active = true;
    elections
      .results(id)
      .then((data) => {
        if (active) setResults(data);
      })
      .catch((err) => {
        if (!active) return;
        if (err instanceof ApiError && err.status === 403) {
          setBlocked(true);
          setError(err.message);
        } else {
          setError(err instanceof ApiError ? err.message : "Results could not be loaded.");
        }
      });
    return () => {
      active = false;
    };
  }, [id]);

  const groups = useMemo(() => {
    if (!results) return [];
    const map = new Map<string, Tally[]>();
    for (const tally of results.tallies) {
      const list = map.get(tally.position_id) ?? [];
      list.push(tally);
      map.set(tally.position_id, list);
    }
    return [...map.entries()].map(([positionID, tallies]) => ({
      positionID,
      title: tallies[0]?.position_title || "Position",
      tallies: [...tallies].sort((a, b) => b.votes - a.votes),
    }));
  }, [results]);

  return (
    <main className="app-wrap">
      <nav className="crumbs" aria-label="Breadcrumb" style={{ marginBottom: 16 }}>
        <Link href="/elections">Elections</Link>
        <span>/</span>
        <Link href={`/elections/${id}`}>Election</Link>
        <span>/</span>
        <span>Results</span>
      </nav>

      <div className="page-head">
        <div>
          <span className="lbl">Results</span>
          <h1>Published tallies</h1>
          <p>
            Percentages name their denominator. A leader is not a winner until the result is
            certified.
          </p>
        </div>
        {results ? <StatusChip status={results.status} /> : null}
      </div>

      {error ? (
        <div className={`banner ${blocked ? "hatch" : "bad"}`} role="alert">
          <Icon name={blocked ? "clock" : "alert"} />
          <div>
            <b>{blocked ? "Results are not published yet" : "We could not load results"}</b>
            {error}
          </div>
        </div>
      ) : null}

      {!results && !error ? (
        <div className="panel stack" role="status" aria-busy="true" aria-label="Loading results">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
          <span className="skel" style={{ height: 14, width: "80%" }} />
        </div>
      ) : null}

      {results ? (
        <>
          <div className="banner ok" style={{ marginBottom: 20 }}>
            <Icon name="shield" />
            <div>
              <b>{results.label === "CERTIFIED" ? "Certified final" : "Live provisional"}</b>
              {results.label === "CERTIFIED"
                ? "The result has been certified. Corrections need a recorded exception."
                : "Not a final result. A leader is not a winner until certified."}
            </div>
          </div>

          {groups.length === 0 ? (
            <div className="empty">
              <b>No tallies yet</b>
              <span className="hint">The count has not produced any published tally rows.</span>
            </div>
          ) : (
            groups.map((group) => (
              <section className="tally-group" key={group.positionID}>
                <h3>{group.title}</h3>
                <div className="tally">
                  {group.tallies.map((tally) => (
                    <div className="trow" key={`${tally.position_id}-${tally.choice}`}>
                      <span>
                        <b>{tally.candidate_name || tally.choice}</b>
                        {tally.is_winner && results.label === "PROVISIONAL" ? (
                          <span className="lead-tag">Leading</span>
                        ) : null}
                        {tally.is_winner && results.label === "CERTIFIED" ? (
                          <span className="lead-tag">Elected</span>
                        ) : null}
                      </span>
                      <div className={`bar${tally.is_winner ? " lead" : ""}`}>
                        <i style={{ width: `${Math.min(100, Math.max(0, tally.percentage))}%` }} />
                      </div>
                      <span className="vl">
                        <b>{formatNumber(tally.votes)}</b> · {formatPercent(tally.percentage)}
                      </span>
                    </div>
                  ))}
                </div>
                <p className="hint" style={{ marginTop: 10 }}>
                  Percent of {formatNumber(group.tallies[0]?.valid_votes_total ?? 0)} valid votes.
                  Blank or abstaining selections are not counted for a candidate.
                </p>
              </section>
            ))
          )}
        </>
      ) : null}
    </main>
  );
}

export default function ResultsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth>
      <SiteHeader />
      <ResultsView id={id} />
    </RequireAuth>
  );
}
