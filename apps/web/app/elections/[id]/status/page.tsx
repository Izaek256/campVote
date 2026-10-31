"use client";

import Link from "next/link";
import { use, useEffect, useState } from "react";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { SiteHeader } from "@/components/site-header";
import { StatusChip } from "@/components/status-chip";
import { ApiError, elections } from "@/lib/api";
import { formatEAT, statusLabel } from "@/lib/format";
import type { Election, VoteStatus } from "@/lib/types";

/** Shows whether a ballot was recorded, never what it contained. */
function StatusView({ id }: { id: string }) {
  const [election, setElection] = useState<Election | null>(null);
  const [status, setStatus] = useState<VoteStatus | null>(null);
  const [error, setError] = useState("");
  const [statusCode, setStatusCode] = useState(0);

  useEffect(() => {
    let active = true;
    elections
      .list()
      .then((list) => {
        const found = list.find((item) => item.id === id);
        if (!found) throw new ApiError("This election is not available to you.", 404);
        if (active) setElection(found);
      })
      .catch((err) => {
        if (active) {
          setStatusCode(err instanceof ApiError ? err.status : 0);
          setError(err instanceof ApiError ? err.message : "Election could not be loaded.");
        }
      });
    elections
      .voteStatus(id)
      .then((data) => {
        if (active) setStatus(data);
      })
      .catch((err) => {
        if (!active) return;
        setStatusCode(err instanceof ApiError ? err.status : 0);
        setError(err instanceof ApiError ? err.message : "Your status could not be checked.");
      });
    return () => {
      active = false;
    };
  }, [id]);

  if (error) {
    return (
      <main className="app-wrap narrow">
        <div className={`banner ${statusCode === 401 ? "hatch" : "bad"}`} role="alert">
          <Icon name={statusCode === 401 ? "lock" : "alert"} />
          <div>
            <b>{statusCode === 401 ? "Your session expired" : "We could not check your status"}</b>
            {error}
            {statusCode === 401 ? (
              <>
                {" "}
                <Link href="/login">Sign in again</Link> to continue.
              </>
            ) : null}
          </div>
        </div>
        <p style={{ marginTop: 16 }}>
          <Link className="btn" href="/elections">
            Back to elections
          </Link>
        </p>
      </main>
    );
  }

  if (!election || !status) {
    return (
      <main className="app-wrap narrow" aria-busy="true" aria-label="Checking your vote status">
        <div className="panel stack">
          <span className="skel" style={{ height: 14, width: "40%" }} />
          <span className="skel" style={{ height: 22 }} />
          <span className="skel" style={{ height: 14, width: "70%" }} />
        </div>
      </main>
    );
  }

  return (
    <main className="app-wrap narrow">
      <nav className="crumbs" aria-label="Breadcrumb" style={{ marginBottom: 16 }}>
        <Link href="/elections">Elections</Link>
        <span>/</span>
        <Link href={`/elections/${id}`}>{election.title}</Link>
        <span>/</span>
        <span>Vote status</span>
      </nav>

      <div className="page-head">
        <div>
          <span className="lbl">Participation record</span>
          <h1>Your vote status</h1>
          <div className="row" style={{ marginTop: 8 }}>
            <StatusChip status={election.status} />
            <span className="mono hint">Server time {formatEAT(status.server_time ?? "")}</span>
          </div>
        </div>
      </div>

      {status.voted ? (
        <section className="panel stack">
          <div className="banner ok">
            <Icon name="check" />
            <div>
              <b>A ballot was recorded for you</b>
              Recorded around {formatEAT(status.recorded_at ?? "")} (shown in 15-minute blocks so it
              cannot be matched to a ballot). A ballot is final and cannot be changed.
            </div>
          </div>
          <p className="hint">
            This page confirms participation only. It does not show, and cannot recover, your
            choices.
          </p>
          <div className="row">
            {["CLOSED", "RESULTS_UNDER_REVIEW", "FINALIZED", "ARCHIVED"].includes(
              election.status
            ) ? (
              <Link className="btn" href={`/elections/${id}/results`}>
                <Icon name="chart" size="s" />
                View results
              </Link>
            ) : null}
            <Link className="btn ghost" href="/elections">
              Back to my elections
            </Link>
          </div>
        </section>
      ) : (
        <section className="panel stack">
          <div className={`banner ${election.accepting_votes ? "hatch" : ""}`} role="status">
            <Icon name={election.accepting_votes ? "alert" : "lock"} />
            <div>
              <b>No ballot has been recorded for you</b>
              {election.accepting_votes
                ? "If you tried to vote and are unsure whether it went through, no ballot reached us. You can open the ballot and vote once."
                : `Voting is ${statusLabel(election.status).toLowerCase()}, so a new ballot can no longer be accepted.`}
            </div>
          </div>
          <p className="hint">
            If you voted and this page still says no ballot was recorded, contact the Electoral
            Commission with the time you submitted. Do not ask anyone to vote on your behalf.
          </p>
          <div className="row">
            {election.accepting_votes ? (
              <Link className="btn primary" href={`/elections/${id}/ballot`}>
                <Icon name="ballot" size="s" />
                Open ballot
              </Link>
            ) : null}
            <Link className="btn ghost" href={`/elections/${id}`}>
              Back to election
            </Link>
          </div>
        </section>
      )}
    </main>
  );
}

export default function VoteStatusPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth>
      <SiteHeader />
      <StatusView id={id} />
    </RequireAuth>
  );
}
