"use client";

import Link from "next/link";
import { use, useEffect, useState } from "react";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { SiteHeader } from "@/components/site-header";
import { Countdown } from "@/components/countdown";
import { StatusChip } from "@/components/status-chip";
import { ApiError, elections } from "@/lib/api";
import { categoryLabel, formatEAT, formatNumber, statusLabel } from "@/lib/format";
import type { Election, EligibilityDecision } from "@/lib/types";

const railStates = [
  "DRAFT",
  "REGISTRATION",
  "PENDING_APPROVAL",
  "SCHEDULED",
  "OPEN",
  "CLOSED",
  "RESULTS_UNDER_REVIEW",
  "FINALIZED",
] as const;

const railLabels: Record<string, string> = {
  DRAFT: "Draft",
  REGISTRATION: "Registration",
  PENDING_APPROVAL: "Approval",
  SCHEDULED: "Scheduled",
  OPEN: "Open",
  CLOSED: "Closed",
  RESULTS_UNDER_REVIEW: "Review",
  FINALIZED: "Finalized",
};

function LifecycleRail({ status }: { status: Election["status"] }) {
  const current = Math.max(0, railStates.indexOf(status as (typeof railStates)[number]));
  return (
    <div className="rail" role="group" aria-label={`Election state: ${statusLabel(status)}`}>
      {railStates.map((state, index) => (
        <div
          key={state}
          className={`ls${index < current ? " done" : ""}${index === current ? " cur" : ""}`}
        >
          {railLabels[state]}
        </div>
      ))}
    </div>
  );
}

function ElectionView({ id }: { id: string }) {
  const [election, setElection] = useState<Election | null>(null);
  const [eligibility, setEligibility] = useState<EligibilityDecision | null>(null);
  const [error, setError] = useState("");

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
        if (active)
          setError(err instanceof ApiError ? err.message : "Election could not be loaded.");
      });
    elections
      .eligibility(id)
      .then((decision) => {
        if (active) setEligibility(decision);
      })
      .catch(() => {
        if (active)
          setEligibility({ eligible: false, reason: "Eligibility could not be checked." });
      });
    return () => {
      active = false;
    };
  }, [id]);

  if (error) {
    return (
      <main className="app-wrap narrow">
        <div className="banner bad" role="alert">
          <Icon name="alert" />
          <div>
            <b>We could not open this election</b>
            {error}
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

  if (!election) {
    return (
      <main className="app-wrap" aria-busy="true" aria-label="Loading election">
        <div className="skel" style={{ height: 28, width: "45%", marginBottom: 16 }} />
        <div className="panel stack">
          <span className="skel" style={{ height: 14 }} />
          <span className="skel" style={{ height: 14, width: "70%" }} />
        </div>
      </main>
    );
  }

  const voted = Boolean(election.my_vote_status?.voted);
  const canVote = Boolean(eligibility?.eligible) && election.accepting_votes && !voted;

  return (
    <main className="app-wrap">
      <nav className="crumbs" aria-label="Breadcrumb" style={{ marginBottom: 16 }}>
        <Link href="/elections">Elections</Link>
        <span>/</span>
        <span>{election.title}</span>
      </nav>

      <div className="page-head">
        <div>
          <div className="row" style={{ marginBottom: 8 }}>
            <StatusChip status={election.status} />
            <span className="tag">{categoryLabel(election.category)}</span>
          </div>
          <h1>{election.title}</h1>
          <p>{election.description}</p>
        </div>
      </div>

      <div className="split">
        <div className="stack">
          <section className="panel">
            <div className="section-h" style={{ marginTop: 0 }}>
              <h2>Schedule</h2>
              <span className="mono">{election.display_timezone}</span>
            </div>
            <dl className="kv">
              <dt>Voting opens</dt>
              <dd>{formatEAT(election.starts_at)}</dd>
              <dt>Voting closes</dt>
              <dd>{formatEAT(election.ends_at)}</dd>
              <dt>Organising unit</dt>
              <dd>{election.organizing_unit}</dd>
              <dt>Results policy</dt>
              <dd>{election.result_policy.replaceAll("_", " ").toLowerCase()}</dd>
            </dl>
            <p className="hint" style={{ marginTop: 14 }}>
              The server clock decides when voting opens and closes. Your device clock is only for
              display.
            </p>
          </section>

          <section className="panel">
            <div className="section-h" style={{ marginTop: 0 }}>
              <h2>Lifecycle</h2>
            </div>
            <LifecycleRail status={election.status} />
          </section>

          <div className="banner">
            <Icon name="lock" />
            <div>
              <b>Your vote is secret</b>
              Officials and candidates cannot see your choices. The ballot is stored separately from
              the record that you voted.
            </div>
          </div>
        </div>

        <aside className="stack">
          {election.accepting_votes ? (
            <div className="panel stack">
              <span className="lbl">Voting closes in</span>
              <Countdown endsAt={election.ends_at} />
              {eligibility === null ? (
                <span className="hint">Checking your eligibility…</span>
              ) : eligibility.eligible ? (
                <span className="field okm">
                  <Icon name="check" size="s" />
                  You are on the eligibility roster for this election.
                </span>
              ) : (
                <span className="error-text" role="alert">
                  <Icon name="alert" size="s" />
                  {eligibility.reason ?? "You are not eligible for this election."}
                </span>
              )}
              {voted ? (
                <Link className="btn block lg" href={`/elections/${election.id}/status`}>
                  <Icon name="check" />
                  View vote status
                </Link>
              ) : canVote ? (
                <Link className="btn primary block lg" href={`/elections/${election.id}/ballot`}>
                  <Icon name="ballot" />
                  Open ballot
                </Link>
              ) : (
                <button className="btn block lg" type="button" disabled>
                  Ballot unavailable
                </button>
              )}
            </div>
          ) : (
            <div className="panel stack">
              <span className="lbl">Status</span>
              <b style={{ fontSize: 20 }}>{statusLabel(election.status)}</b>
              <span className="hint">
                Voting {election.status === "SCHEDULED" ? "opens" : "opened"}{" "}
                {formatEAT(election.starts_at)}.
              </span>
              <Link className="btn block" href={`/elections/${election.id}/results`}>
                <Icon name="chart" />
                View results
              </Link>
            </div>
          )}

          <div className="panel stack">
            <span className="lbl">Eligibility</span>
            {eligibility?.eligible ? (
              <span className="field okm">
                <Icon name="check" size="s" />
                Roster entry confirmed
              </span>
            ) : (
              <span className="hint">{eligibility?.reason ?? "Not currently eligible."}</span>
            )}
            <span className="hint">
              {formatNumber(election.eligibility_frozen_count)} voters were on the frozen roster
              when voting opened.
            </span>
          </div>
        </aside>
      </div>
    </main>
  );
}

export default function ElectionPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth>
      <SiteHeader />
      <ElectionView id={id} />
    </RequireAuth>
  );
}
