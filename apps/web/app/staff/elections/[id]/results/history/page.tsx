"use client";

import Link from "next/link";
import { use, useEffect, useState } from "react";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { ResultStatusBadge } from "@/components/result-status-badge";
import { StaffShell } from "@/components/staff-shell";
import { ApiError, elections } from "@/lib/api";
import { formatEAT } from "@/lib/format";
import type { ResultSnapshot, ResultsHistory } from "@/lib/types";

/**
 * Result history — ISSUE-09.
 *
 * Every snapshot of this election, newest first. A correction shows the
 * original and the superseding version together, and the original is never
 * hidden or editable: there are no edit or delete controls anywhere in here,
 * because a certified record is immutable by design.
 */
function History({ id }: { id: string }) {
  const [history, setHistory] = useState<ResultsHistory | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    elections
      .resultsHistory(id)
      .then((data) => {
        if (active) setHistory(data);
      })
      .catch((err) => {
        if (!active) return;
        setError(err instanceof ApiError ? err.message : "History could not be loaded.");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [id]);

  const superseded = (snapshot: ResultSnapshot) => snapshot.status === "SUPERSEDED";

  return (
    <StaffShell title="Result history">
      <nav className="crumbs" aria-label="Breadcrumb" style={{ marginBottom: 16 }}>
        <Link href="/staff">Staff</Link>
        <span>/</span>
        <Link href={`/staff/elections/${id}`}>Election</Link>
        <span>/</span>
        <Link href={`/staff/elections/${id}/results`}>Results</Link>
        <span>/</span>
        <span>History</span>
      </nav>

      <div className="page-head">
        <div>
          <span className="lbl">History</span>
          <h1>Result history and corrections</h1>
          <p>
            Every version this election has produced. A certified record cannot be edited or
            deleted; a correction is a new version that supersedes it.
          </p>
        </div>
      </div>

      {error ? (
        <div className="banner bad" role="alert">
          <Icon name="alert" />
          <div>
            <b>History could not be loaded</b>
            {error}
          </div>
        </div>
      ) : null}

      {loading && !error ? (
        <div className="panel stack" role="status" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      ) : null}

      {history && history.snapshots.length === 0 ? (
        <div className="empty">
          <div className="ic">
            <Icon name="clock" />
          </div>
          <b>No result has been recorded</b>
          <span className="hint">Run a tally from the results workspace to create one.</span>
        </div>
      ) : null}

      {history && history.snapshots.length > 0 ? (
        <div className="stack">
          {history.snapshots.map((snapshot) => (
            <section
              className={`panel${superseded(snapshot) ? " superseded" : ""}`}
              key={snapshot.id}
              aria-labelledby={`snap-${snapshot.id}`}
            >
              <div className="section-h" style={{ marginTop: 0 }}>
                <h2 id={`snap-${snapshot.id}`}>
                  Version {snapshot.version}
                  {snapshot.superseded_by_id ? " — superseded" : ""}
                </h2>
                <ResultStatusBadge status={snapshot.status} />
              </div>

              {snapshot.supersedes_id ? (
                <p className="hint" style={{ marginTop: 0 }}>
                  This version supersedes an earlier one. The original stays visible below and in
                  the audit trail.
                </p>
              ) : null}

              <dl className="kv">
                <dt>Computed</dt>
                <dd>{formatEAT(snapshot.computed_at)}</dd>
                <dt>Hash</dt>
                <dd>
                  <code className="mono">{snapshot.content_hash}</code>
                </dd>
                <dt>Turnout</dt>
                <dd>
                  {snapshot.turnout_percent.toFixed(1)}% of {snapshot.eligible_voters} eligible (
                  {snapshot.ballots_cast} cast)
                </dd>
                {snapshot.certified_at ? (
                  <>
                    <dt>Certified</dt>
                    <dd>{formatEAT(snapshot.certified_at)}</dd>
                    <dt>Certification reason</dt>
                    <dd>{snapshot.certify_reason || "—"}</dd>
                  </>
                ) : null}
              </dl>

              {/* No edit or delete control exists for a certified record. */}
              <div className="tbl-wrap" style={{ marginTop: 14 }}>
                <table className="tbl">
                  <caption className="sr">Version {snapshot.version}: outcome per position</caption>
                  <thead>
                    <tr>
                      <th scope="col">Position</th>
                      <th scope="col">Outcome</th>
                      <th scope="col" className="r">
                        Valid votes
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {snapshot.positions.map((position) => (
                      <tr key={position.position_id}>
                        <th scope="row">{position.title}</th>
                        <td>
                          {position.outcome === "WINNER"
                            ? position.winners.length > 1
                              ? `${position.winners.length} seats filled`
                              : "Result declared"
                            : position.outcome.replaceAll("_", " ").toLowerCase()}
                        </td>
                        <td className="r">{position.valid_votes}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          ))}

          {history.recounts.length > 0 ? (
            <section className="panel">
              <div className="section-h" style={{ marginTop: 0 }}>
                <h2>Recounts</h2>
              </div>
              <div className="tbl-wrap">
                <table className="tbl">
                  <caption className="sr">Recorded recounts</caption>
                  <thead>
                    <tr>
                      <th scope="col">Run</th>
                      <th scope="col">Reason</th>
                      <th scope="col">Outcome</th>
                    </tr>
                  </thead>
                  <tbody>
                    {history.recounts.map((recount) => (
                      <tr key={recount.id}>
                        <th scope="row">{formatEAT(recount.run_at)}</th>
                        <td>{recount.reason || "—"}</td>
                        <td>
                          {recount.identical
                            ? "Identical to the stored result"
                            : "Differences recorded"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          ) : null}
        </div>
      ) : null}
    </StaffShell>
  );
}

export default function ResultHistoryPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth>
      <History id={id} />
    </RequireAuth>
  );
}
