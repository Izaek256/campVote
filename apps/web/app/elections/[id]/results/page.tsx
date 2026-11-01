"use client";

import Link from "next/link";
import { use, useEffect, useState } from "react";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { SiteHeader } from "@/components/site-header";
import { StatusChip } from "@/components/status-chip";
import { ApiError, elections } from "@/lib/api";
import { formatNumber, formatPercent } from "@/lib/format";
import type { PositionResult, Results } from "@/lib/types";

function PositionTally({ position, certified }: { position: PositionResult; certified: boolean }) {
  return (
    <section className="panel" style={{ marginBottom: 16 }}>
      <div className="section-h" style={{ marginTop: 0 }}>
        <h2>{position.title}</h2>
        <span className="mono">
          {position.outcome.replaceAll("_", " ").toLowerCase()} ·{" "}
          {formatNumber(position.valid_votes)} valid votes
        </span>
      </div>
      <div className="tally">
        {position.rows
          .slice()
          .sort((a, b) => b.votes - a.votes)
          .map((row) => (
            <div className="trow" key={row.candidate_id}>
              <span>
                <b>{row.name}</b>
                {row.is_winner ? (
                  <span className="lead-tag">{certified ? "Elected" : "Leading"}</span>
                ) : null}
                {row.is_tie ? <span className="lead-tag">Tie</span> : null}
              </span>
              <div className={`bar${row.is_winner ? " lead" : ""}`}>
                <i style={{ width: `${Math.min(100, Math.max(0, row.share_percent))}%` }} />
              </div>
              <span className="vl">
                <b>{formatNumber(row.votes)}</b> · {formatPercent(row.share_percent)}
              </span>
            </div>
          ))}
      </div>
      <p className="hint" style={{ marginTop: 10 }}>
        Share of {formatNumber(position.valid_votes)} valid votes. Blank or abstaining selections
        are not counted for a candidate.
      </p>
    </section>
  );
}

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

  const snapshot = results?.snapshot;
  const certified = snapshot?.status === "CERTIFIED";

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
        {results?.election_status ? <StatusChip status={results.election_status} /> : null}
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

      {results && !snapshot ? (
        <EmptyState title="No results published yet">
          {results.reason ?? "The count has not produced a published snapshot."}
        </EmptyState>
      ) : null}

      {results && snapshot ? (
        <>
          <div className={`banner ${certified ? "ok" : ""}`} style={{ marginBottom: 20 }}>
            <Icon name={certified ? "shield" : "clock"} />
            <div>
              <b>{certified ? "Certified final" : "Provisional result"}</b>
              {certified
                ? "The result has been certified. Corrections need a recorded exception."
                : "Not a final result. A leader is not a winner until certified."}
            </div>
          </div>

          <div className="kpis" style={{ marginBottom: 20 }}>
            <div className="kpi">
              <span className="l">Eligible voters</span>
              <span className="v">{formatNumber(snapshot.eligible_voters)}</span>
            </div>
            <div className="kpi">
              <span className="l">Ballots cast</span>
              <span className="v">{formatNumber(snapshot.ballots_cast)}</span>
            </div>
            <div className="kpi">
              <span className="l">Turnout</span>
              <span className="v">{formatPercent(snapshot.turnout_percent)}</span>
            </div>
          </div>

          {snapshot.positions.length === 0 ? (
            <EmptyState title="No tallies yet">
              The count has not produced any published tally rows.
            </EmptyState>
          ) : (
            snapshot.positions.map((position) => (
              <PositionTally key={position.position_id} position={position} certified={certified} />
            ))
          )}

          <p className="hint">
            Snapshot version {snapshot.version} · computed{" "}
            {new Date(snapshot.computed_at).toLocaleString()}
          </p>
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
